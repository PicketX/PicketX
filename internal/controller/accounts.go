// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"fmt"
	"github.com/PicketX/PicketX/internal/account"
	"github.com/labstack/echo/v5"
)

type CreateAccountRequest struct {
	Username string   `json:"username" validate:"required,min=3,max=64"`
	Password string   `json:"password" validate:"required,max=72"`
	Roles    []string `json:"roles" validate:"required,min=1,max=4,unique,dive,oneof=role:admin role:viewer role:requester role:blocked"`
	Reason   string   `json:"reason" validate:"required,max=512"`
}
type UpdateAccountRequest struct {
	Roles    []string `json:"roles" validate:"required,min=1,max=4,unique,dive,oneof=role:admin role:viewer role:requester role:blocked"`
	Disabled *bool    `json:"disabled" validate:"required"`
	Password string   `json:"password,omitempty" validate:"omitempty,max=72"`
	Reason   string   `json:"reason" validate:"required,max=512"`
}

// accounts godoc
// @Summary Read bounded human directory and latest50 credential-free audit events
// @Description Explicit database users mode only; revision is independent of publication. No usernames are recycled. Requires account:read.
// @ID listAccounts
// @Tags accounts
// @Security ManagementBearer
// @Produce json
// @Success 200 {object} account.Page
// @Header 200 {string} ETag "Strong account directory revision"
// @Failure 400,401,403,429,503 {object} APIError
// @Router /api/v1/accounts [get]
// @Router /console/api/v1/accounts [get]
func (h *managementHTTP) accounts(c *echo.Context) error {
	p, err := h.principal(c, "account:read")
	if err != nil {
		return mappedError(c, err)
	}
	if c.Request().URL.RawQuery != "" {
		return apiError(c, 400, "invalid_query")
	}
	page, err := h.service.Accounts(p)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, page.Revision))
	return c.JSON(200, page)
}

// createAccount godoc
// @Summary Create a local human account with immutable username and audited CAS
// @Description Strict JSON16KiB; password12–72 UTF-8 bytes, bcrypt12; max128 accounts. Requires account:write. Does not create a network Grant.
// @ID createAccount
// @Tags accounts
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong account directory revision, not publication revision"
// @Param account body CreateAccountRequest true "Password is write-only and never logged or audited"
// @Success 201 {object} account.Account
// @Header 201 {string} ETag "New account directory revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/accounts [post]
// @Router /console/api/v1/accounts [post]
func (h *managementHTTP) createAccount(c *echo.Context) error { return h.saveAccount(c, true) }

// updateAccount godoc
// @Summary Change roles, enable/disable or reset password; revoke all old human sessions
// @Description Every committed edit advances persistent account version, even if values do not change. Re-enabling never revives old cookies. No delete/rename. Requires account:write.
// @ID updateAccount
// @Tags accounts
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param id path string true "Immutable account principal ID"
// @Param If-Match header string true "Strong account directory revision"
// @Param account body UpdateAccountRequest true "disabled required; omitted or empty password retains hash"
// @Success 200 {object} account.Account
// @Header 200 {string} ETag "New account directory revision"
// @Failure 400,401,403,404,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/accounts/{id} [put]
// @Router /console/api/v1/accounts/{id} [put]
func (h *managementHTTP) updateAccount(c *echo.Context) error { return h.saveAccount(c, false) }

func (h *managementHTTP) saveAccount(c *echo.Context, create bool) error {
	p, err := h.principal(c, "account:write")
	if err != nil {
		return mappedError(c, err)
	}
	expected, status := requiredRevision(c.Request())
	if status != 0 {
		return revisionError(c, status)
	}
	var change account.Change
	id := ""
	if create {
		var dto CreateAccountRequest
		if status, code := decodeGrantJSON(c, &dto); status != 0 {
			return apiError(c, status, code)
		}
		if err = h.validate.Struct(dto); err != nil {
			return apiError(c, 400, "invalid_fields")
		}
		change = account.Change{Username: dto.Username, Password: dto.Password, Roles: dto.Roles, Reason: dto.Reason}
	} else {
		var dto UpdateAccountRequest
		if status, code := decodeGrantJSON(c, &dto); status != 0 {
			return apiError(c, status, code)
		}
		if err = h.validate.Struct(dto); err != nil {
			return apiError(c, 400, "invalid_fields")
		}
		id = c.Param("id")
		change = account.Change{Password: dto.Password, Roles: dto.Roles, Disabled: *dto.Disabled, Reason: dto.Reason}
	}
	out, revision, err := h.service.SaveAccount(p, expected, id, change)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, revision))
	status = 200
	if create {
		status = 201
	}
	return c.JSON(status, out)
}
