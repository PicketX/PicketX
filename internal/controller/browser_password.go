// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"github.com/PicketX/PicketX/internal/password"
	"github.com/PicketX/PicketX/internal/strictjson"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"io"
	"mime"
)

type PasswordLoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=64"`
	Password string `json:"password" validate:"required"`
}

// loginPassword godoc
// @Summary Exchange local human credentials for a renewed SCS session
// @Description Exact Host/Origin and Echo CSRF required; strict4096byte JSON. Password12–72 UTF-8 bytes, no trimming. Direct-peer local limiter6/minute burst5, four concurrent bcrypt checks per process. Generic401 for unknown/disabled/wrong credentials. Password/username rotation, disable/removal invalidate sessions on next request. No raw password/hash stored in session; Bearer API remains key-only. See HUMAN_LOGIN.md for provisioning/deployment limits.
// @ID browserPasswordLogin
// @Tags console
// @Accept json
// @Produce json
// @Param Origin header string true "Exact configured public origin"
// @Param X-CSRF-Token header string false "Bootstrap token for legacy clients"
// @Param login body PasswordLoginRequest true "Local human credentials"
// @Success 200 {object} SessionIdentityResponse
// @Failure 400,401,403,413,415,429,503 {object} APIError
// @Router /console/api/v1/login/password [post]
func (b *Browser) loginPassword(c *echo.Context, validation *validator.Validate) error {
	media, _, e := mime.ParseMediaType(c.Request().Header.Get("Content-Type"))
	if e != nil || media != "application/json" {
		return apiError(c, 415, "json_required")
	}
	raw, e := io.ReadAll(io.LimitReader(c.Request().Body, 4097))
	if e != nil || len(raw) > 4096 {
		return apiError(c, 413, "request_too_large")
	}
	var input PasswordLoginRequest
	if strictjson.Decode(raw, &input) != nil || validation.Struct(input) != nil || !password.Valid(input.Password) {
		return apiError(c, 400, "invalid_login")
	}
	p, fp, e := b.service.AuthenticatePassword(input.Username, input.Password)
	if e != nil {
		return mappedError(c, e)
	}
	return b.establishSession(c, p, fp)
}
