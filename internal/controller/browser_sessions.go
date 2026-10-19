// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/strictjson"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

type SessionResponse struct {
	PasswordLogin bool                     `json:"password_login"`
	Authenticated bool                     `json:"authenticated"`
	CSRF          string                   `json:"csrf"`
	Identity      *SessionIdentityResponse `json:"identity,omitempty"`
}
type SessionIdentityResponse struct {
	ID    string   `json:"id"`
	Roles []string `json:"roles"`
}
type LoginRequest struct {
	Token string `json:"token" validate:"required,min=32,max=256"`
}

// session godoc
// @Summary Bootstrap CSRF and report whether a browser session is authenticated
// @Description Requires configured public Host. No raw credential or fingerprint is returned. csrf is a legacy token or Echo's modern Sec-Fetch-Site marker. Cache-Control no-store.
// @ID browserSession
// @Tags console
// @Produce json
// @Success 200 {object} SessionResponse
// @Failure 400,403,429,503 {object} APIError
// @Router /console/api/v1/session [get]
func (b *Browser) session(c *echo.Context) error {
	p, err := b.principal(c)
	if err != nil && !errors.Is(err, management.ErrUnauthorized) {
		return mappedError(c, err)
	}
	enabled, e := b.service.PasswordLoginEnabled()
	if e != nil {
		return mappedError(c, e)
	}
	response := SessionResponse{Authenticated: err == nil, CSRF: c.Get("csrf").(string), PasswordLogin: enabled}
	if err == nil {
		response.Identity = &SessionIdentityResponse{ID: p.ID, Roles: p.Roles}
	}
	return c.JSON(200, response)
}

// login godoc
// @Summary Exchange independent management access key for a renewed SCS session
// @Description Exact Origin required. Native Echo CSRF protects login: same-origin Sec-Fetch-Site, or bootstrap X-CSRF-Token plus CSRF cookie for legacy clients. Login body limited to4096bytes. Absolute8h/idle30m; restart invalidates sessions. Current credentials and roles rechecked each request. Access key is not saved in session.
// @ID browserLogin
// @Tags console
// @Accept json
// @Produce json
// @Param Origin header string true "Exact configured public origin"
// @Param X-CSRF-Token header string false "Bootstrap token for legacy clients"
// @Param login body LoginRequest true "Independent management access key"
// @Success 200 {object} SessionIdentityResponse
// @Failure 400,401,403,413,415,429,503 {object} APIError
// @Router /console/api/v1/login [post]
func (b *Browser) login(c *echo.Context, validation *validator.Validate) error {
	media, _, err := mime.ParseMediaType(c.Request().Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return apiError(c, 415, "json_required")
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request().Body, 4097))
	if err != nil || len(raw) > 4096 {
		return apiError(c, 413, "request_too_large")
	}
	var input LoginRequest
	if strictjson.Decode(raw, &input) != nil || validation.Struct(input) != nil {
		return apiError(c, 400, "invalid_login")
	}
	p, err := b.service.Authenticate(input.Token)
	if err != nil {
		return mappedError(c, err)
	}
	sum := sha256.Sum256([]byte(input.Token))
	return b.establishSession(c, p, hex.EncodeToString(sum[:]))
}
func (b *Browser) establishSession(c *echo.Context, p management.Principal, fingerprint string) error {
	ctx := c.Request().Context()
	if err := b.sessions.Destroy(ctx); err != nil {
		return mappedError(c, err)
	}
	if err := b.sessions.RenewToken(ctx); err != nil {
		return mappedError(c, err)
	}
	b.sessions.Put(ctx, "id", p.ID)
	b.sessions.Put(ctx, "credential", fingerprint)
	return c.JSON(200, SessionIdentityResponse{ID: p.ID, Roles: p.Roles})
}

// logout godoc
// @Summary Destroy browser session, including protection from in-flight idle re-commit
// @Description Idempotent even without an authenticated session. Requires exact Origin and native Echo CSRF. Only current session is destroyed; credential removal/rotation revokes other sessions on next request.
// @ID browserLogout
// @Tags console
// @Param Origin header string true "Exact configured public origin"
// @Param X-CSRF-Token header string false "Bootstrap token for legacy clients"
// @Success 204
// @Failure 400,403,429,503 {object} APIError
// @Router /console/api/v1/logout [post]
func (b *Browser) logout(c *echo.Context) error {
	if err := b.sessions.Destroy(c.Request().Context()); err != nil {
		return mappedError(c, err)
	}
	return c.NoContent(204)
}
