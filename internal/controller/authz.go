// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"errors"
	"github.com/PicketX/PicketX/internal/authz"
	"github.com/PicketX/PicketX/internal/strictjson"
	"github.com/PicketX/PicketX/pkg/httpauth"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"io"
	"mime"
	"net/http"
	"net/netip"
	"strings"
)

type HTTPDecisionRequest struct {
	Route  string `json:"route" validate:"required,max=128"`
	Source string `json:"source" validate:"required,max=45"`
	Host   string `json:"host" validate:"required,max=253"`
	Method string `json:"method" validate:"required,oneof=GET HEAD POST PUT PATCH DELETE OPTIONS"`
	URI    string `json:"uri" validate:"required,max=8192"`
}
type authorizationHTTP struct {
	service  *authz.Service
	validate *validator.Validate
}

func NewAuthorizationHandler(service *authz.Service) http.Handler {
	e := echo.New()
	h := authorizationHTTP{service: service, validate: validator.New()}
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set("Cache-Control", "no-store")
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			return next(c)
		}
	})
	// A global single listener budget bounds valid/invalid callers alike. It is
	// independent of ordinary management and Agent communication counters.
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: 128, Burst: 256}), IdentifierExtractor: func(*echo.Context) (string, error) { return "local", nil }, DenyHandler: func(c *echo.Context, _ string, _ error) error { return apiError(c, 429, "rate_limited") }}))
	e.POST("/authz/v1/decisions", h.json)
	e.GET("/authz/v1/nginx", h.nginx)
	return e
}
func (h authorizationHTTP) decide(c *echo.Context, dto HTTPDecisionRequest, nginx bool) error {
	values := c.Request().Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return apiError(c, 401, "unauthorized")
	}
	if c.Request().Header.Get("Cookie") != "" || c.Request().URL.RawQuery != "" || h.validate.Struct(dto) != nil {
		return apiError(c, 400, "invalid_context")
	}
	source, e := netip.ParseAddr(dto.Source)
	if e != nil || source.String() != dto.Source {
		return apiError(c, 400, "invalid_context")
	}
	q := httpauth.Query{Route: dto.Route, Source: source, Host: dto.Host, Method: dto.Method, URI: dto.URI}
	d, e := h.service.Decide(strings.TrimPrefix(values[0], "Bearer "), q)
	switch {
	case errors.Is(e, authz.ErrUnauthorized):
		c.Response().Header().Set("WWW-Authenticate", "Bearer")
		return apiError(c, 401, "unauthorized")
	case errors.Is(e, authz.ErrForbidden):
		return apiError(c, 403, "forbidden")
	case errors.Is(e, authz.ErrInvalid):
		return apiError(c, 400, "invalid_context")
	case e != nil:
		return apiError(c, 503, "authorization_unavailable")
	}
	if nginx {
		if d.Allowed {
			return c.NoContent(204)
		}
		return c.NoContent(403)
	}
	return c.JSON(200, d)
}

// json godoc
// @Summary Decide an exact HTTP request using source Grants and route constraints
// @Description Dedicated loopback/Unix listener only. Caller supplies trusted context; exact canonical Host/method/path required. Source is not a verified user. Server time, no caching; durable minimal audit before allow. Query is not audited. No administrative routes on this listener.
// @ID decideHTTP
// @Tags authorization
// @Security IngressBearer
// @Accept json
// @Produce json
// @Param context body HTTPDecisionRequest true "Trusted request context (no client time/identity claims)"
// @Success 200 {object} publication.HTTPDecision
// @Failure 400,401,403,413,415,429,503 {object} APIError
// @Router /authz/v1/decisions [post]
func (h authorizationHTTP) json(c *echo.Context) error {
	content, _, e := mime.ParseMediaType(c.Request().Header.Get("Content-Type"))
	if e != nil || content != "application/json" {
		return apiError(c, 415, "json_required")
	}
	data, e := io.ReadAll(http.MaxBytesReader(c.Response(), c.Request().Body, 16<<10))
	if e != nil {
		var big *http.MaxBytesError
		if errors.As(e, &big) {
			return apiError(c, 413, "body_too_large")
		}
		return apiError(c, 400, "invalid_json")
	}
	var dto HTTPDecisionRequest
	if e = strictjson.Decode(data, &dto); e != nil {
		return apiError(c, 400, "invalid_json")
	}
	return h.decide(c, dto, false)
}

// nginx godoc
// @Summary Nginx internal auth_request callback, no body or cache
// @Description Each X-PicketX-* header must occur exactly once. Internal ingress sets a fixed route UID and actual source/Host/original method/request URI; never forward client authorization headers to this listener.204 allows,403 denies; other statuses must fail closed.
// @ID nginxHTTPAuthorization
// @Tags authorization
// @Security IngressBearer
// @Param X-PicketX-Route header string true "Pinned route UID"
// @Param X-PicketX-Source header string true "Trusted unmapped unzoned source IP"
// @Param X-PicketX-Host header string true "Actual routing authority"
// @Param X-PicketX-Method header string true "Original HTTP method"
// @Param X-PicketX-URI header string true "Original request URI"
// @Success 204 "Allowed"
// @Failure 400,401,403,429,503 {object} APIError
// @Router /authz/v1/nginx [get]
func (h authorizationHTTP) nginx(c *echo.Context) error {
	var dto HTTPDecisionRequest
	for name, target := range map[string]*string{"X-PicketX-Route": &dto.Route, "X-PicketX-Source": &dto.Source, "X-PicketX-Host": &dto.Host, "X-PicketX-Method": &dto.Method, "X-PicketX-URI": &dto.URI} {
		v := c.Request().Header.Values(name)
		if len(v) != 1 {
			return apiError(c, 400, "invalid_context")
		}
		*target = v[0]
	}
	if c.Request().ContentLength > 0 || len(c.Request().TransferEncoding) > 0 {
		return apiError(c, 400, "body_not_allowed")
	}
	return h.decide(c, dto, true)
}
