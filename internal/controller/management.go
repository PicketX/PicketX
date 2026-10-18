// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/internal/strictjson"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const MaxManagementRequestBytes = 12 << 20

type ManagementLimits struct{ RequestsPerSecond, Burst int }
type APIError struct {
	Code string `json:"code"`
}
type PublicationRequest struct {
	Policy string `json:"policy" validate:"required,max=4194304"`
	Facts  string `json:"facts" validate:"required,max=4194304"`
	Reason string `json:"reason" validate:"required,max=512"`
}
type AuditResponse struct {
	Events    []publication.AuditEvent `json:"events"`
	NextAfter int64                    `json:"next_after"`
}
type HTTPAuditResponse struct {
	Events    []publication.HTTPAuditEvent `json:"events"`
	NextAfter int64                        `json:"next_after"`
}
type IdentityResponse struct {
	ID         string   `json:"id"`
	Roles      []string `json:"roles"`
	Operations []string `json:"operations"`
}

type managementHTTP struct {
	service  *management.Service
	validate *validator.Validate
	resolve  func(*echo.Context) (management.Principal, error)
}

func registerManagement(e *echo.Echo, service *management.Service, limits ManagementLimits) {
	h := &managementHTTP{service: service, validate: validator.New()}
	group := e.Group("/api/v1")
	group.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error { c.Response().Header().Set("Cache-Control", "no-store"); return next(c) }
	})
	group.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: float64(limits.RequestsPerSecond), Burst: limits.Burst}),
		IdentifierExtractor: func(c *echo.Context) (string, error) {
			host, _, e := net.SplitHostPort(c.Request().RemoteAddr)
			if e != nil {
				return "", e
			}
			return host, nil
		},
		ErrorHandler: func(c *echo.Context, _ error) error { return apiError(c, 400, "invalid_peer") },
		DenyHandler:  func(c *echo.Context, _ string, _ error) error { return apiError(c, 429, "rate_limited") },
	}))
	group.GET("/publication", h.getPublication)
	group.PUT("/publication", h.putPublication)
	group.GET("/audit", h.audit)
	group.GET("/http-decisions", h.httpAudit)
	group.GET("/me", h.me)
	group.GET("/grants", h.grants)
	group.POST("/grants", h.issueGrant)
	group.POST("/grants/:uid/revoke", h.revokeGrant)
	group.GET("/inventory", h.inventory)
	group.POST("/resources", h.createResource)
	group.POST("/bindings", h.createBinding)
	group.POST("/http-routes", h.createHTTPRoute)
}
func apiError(c *echo.Context, status int, code string) error {
	return c.JSON(status, APIError{Code: code})
}
func mappedError(c *echo.Context, e error) error {
	switch {
	case errors.Is(e, management.ErrUnauthorized):
		if strings.HasPrefix(c.Request().URL.Path, "/api/v1/") {
			c.Response().Header().Set("WWW-Authenticate", "Bearer")
		}
		return apiError(c, 401, "unauthorized")
	case errors.Is(e, management.ErrForbidden):
		return apiError(c, 403, "forbidden")
	case errors.Is(e, publication.ErrPrecondition):
		return apiError(c, 412, "revision_mismatch")
	case errors.Is(e, management.ErrInvalid):
		return apiError(c, 400, "invalid_publication")
	case errors.Is(e, management.ErrGrantNotFound):
		return apiError(c, 404, "grant_not_found")
	default:
		return apiError(c, 503, "management_unavailable")
	}
}
func (h *managementHTTP) principal(c *echo.Context, operation string) (management.Principal, error) {
	if h.resolve != nil {
		p, e := h.resolve(c)
		if e == nil && operation != "" {
			e = h.service.Check(p, operation)
		}
		return p, e
	}
	values := c.Request().Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return management.Principal{}, management.ErrUnauthorized
	}
	p, e := h.service.Authenticate(strings.TrimPrefix(values[0], "Bearer "))
	if e != nil {
		return p, e
	}
	if operation != "" {
		e = h.service.Check(p, operation)
	}
	return p, e
}

// getPublication godoc
// @Summary Read accepted desired configuration (not Agent application)
// @ID readPublication
// @Tags management
// @Security ManagementBearer
// @Produce json
// @Success 200 {object} publication.Document
// @Header 200 {string} ETag "Strong accepted revision"
// @Failure 401,403,429,503 {object} APIError
// @Router /api/v1/publication [get]
func (h *managementHTTP) getPublication(c *echo.Context) error {
	p, e := h.principal(c, "publication:read")
	if e != nil {
		return mappedError(c, e)
	}
	d, e := h.service.Snapshot(p)
	if e != nil {
		return mappedError(c, e)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, d.Revision))
	return c.JSON(200, d)
}

// putPublication godoc
// @Summary Atomically replace desired configuration with audit and revision precondition
// @ID replacePublication
// @Description JSON rejects unknown/duplicate/null fields; raw policy/facts preserve presence semantics. Body limit12MiB. Database mode only. Success is not Agent application.
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong current revision, e.g. \"1\"; wildcards and weak tags rejected"
// @Param publication body PublicationRequest true "Complete policy YAML, facts JSON and audit reason"
// @Success 200 {object} publication.Document
// @Header 200 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/publication [put]
func (h *managementHTTP) putPublication(c *echo.Context) error {
	p, e := h.principal(c, "publication:write")
	if e != nil {
		return mappedError(c, e)
	}
	values := c.Request().Header.Values("If-Match")
	if len(values) == 0 {
		return apiError(c, 428, "revision_required")
	}
	if len(values) != 1 {
		return apiError(c, 400, "invalid_revision")
	}
	value := values[0]
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		return apiError(c, 400, "invalid_revision")
	}
	digits := value[1 : len(value)-1]
	revision, e := strconv.ParseUint(digits, 10, 63)
	if e != nil || strconv.FormatUint(revision, 10) != digits {
		return apiError(c, 400, "invalid_revision")
	}
	content, _, e := mime.ParseMediaType(c.Request().Header.Get("Content-Type"))
	// ParseMediaType returns parameters separately; require JSON, never form/query binding.
	if e != nil || content != "application/json" {
		return apiError(c, 415, "json_required")
	}
	body := http.MaxBytesReader(c.Response(), c.Request().Body, MaxManagementRequestBytes)
	data, e := io.ReadAll(body)
	if e != nil {
		var tooBig *http.MaxBytesError
		if errors.As(e, &tooBig) {
			return apiError(c, 413, "body_too_large")
		}
		return apiError(c, 400, "invalid_json")
	}
	var dto PublicationRequest
	if e = strictjson.Decode(data, &dto); e != nil {
		return apiError(c, 400, "invalid_json")
	}
	if e = h.validate.Struct(dto); e != nil {
		return apiError(c, 400, "invalid_fields")
	}
	d, e := h.service.Replace(p, revision, dto.Reason, dto.Policy, dto.Facts)
	if e != nil {
		return mappedError(c, e)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, d.Revision))
	return c.JSON(200, d)
}

// audit godoc
// @Summary Read change audit in ascending keyset order
// @ID readAudit
// @Tags management
// @Security ManagementBearer
// @Produce json
// @Param after query integer false "Exclusive event ID (default0)" minimum(0)
// @Param limit query integer false "Page size (default50)" minimum(1) maximum(100)
// @Success 200 {object} AuditResponse
// @Failure 400,401,403,429,503 {object} APIError
// @Router /api/v1/audit [get]
func (h *managementHTTP) audit(c *echo.Context) error {
	return h.auditPage(c, false)
}
func (h *managementHTTP) auditPage(c *echo.Context, httpDecisions bool) error {
	p, e := h.principal(c, "audit:read")
	if e != nil {
		return mappedError(c, e)
	}
	q, e := url.ParseQuery(c.Request().URL.RawQuery)
	if e != nil {
		return apiError(c, 400, "invalid_query")
	}
	after, limit := int64(0), 50
	for key, values := range q {
		if len(values) != 1 {
			return apiError(c, 400, "invalid_query")
		}
		switch key {
		case "after":
			after, e = strconv.ParseInt(values[0], 10, 64)
		case "limit":
			limit, e = strconv.Atoi(values[0])
		default:
			return apiError(c, 400, "invalid_query")
		}
		if e != nil {
			return apiError(c, 400, "invalid_query")
		}
	}
	if after < 0 || limit < 1 || limit > 100 {
		return apiError(c, 400, "invalid_query")
	}
	if httpDecisions {
		events, e := h.service.HTTPAudit(p, after, limit)
		if e != nil {
			return mappedError(c, e)
		}
		next := after
		if len(events) > 0 {
			next = events[len(events)-1].ID
		}
		return c.JSON(200, HTTPAuditResponse{Events: events, NextAfter: next})
	}
	events, e := h.service.Audit(p, after, limit)
	if e != nil {
		return mappedError(c, e)
	}
	next := after
	if len(events) > 0 {
		next = events[len(events)-1].ID
	}
	return c.JSON(200, AuditResponse{Events: events, NextAfter: next})
}

// httpAudit godoc
// @Summary Read durable minimal HTTP decision audit in ascending keyset order
// @ID httpDecisionAudit
// @Tags management
// @Security ManagementBearer
// @Produce json
// @Param after query integer false "Exclusive event ID, default0" minimum(0)
// @Param limit query integer false "Page size, default50" minimum(1) maximum(100)
// @Success 200 {object} HTTPAuditResponse
// @Failure 400,401,403,429,503 {object} APIError
// @Router /api/v1/http-decisions [get]
func (h *managementHTTP) httpAudit(c *echo.Context) error { return h.auditPage(c, true) }

// me godoc
// @Summary Read authenticated API principal and current permitted operations
// @ID readCurrentPrincipal
// @Tags management
// @Security ManagementBearer
// @Produce json
// @Success 200 {object} IdentityResponse
// @Failure 401,429,503 {object} APIError
// @Router /api/v1/me [get]
func (h *managementHTTP) me(c *echo.Context) error {
	p, e := h.principal(c, "")
	if e != nil {
		return mappedError(c, e)
	}
	operations := []string{}
	for _, op := range []string{"publication:read", "publication:write", "audit:read"} {
		e = h.service.Check(p, op)
		if e == nil {
			operations = append(operations, op)
		} else if !errors.Is(e, management.ErrForbidden) {
			return mappedError(c, e)
		}
	}
	return c.JSON(200, IdentityResponse{ID: p.ID, Roles: p.Roles, Operations: operations})
}
