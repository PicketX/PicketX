// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"fmt"
	"github.com/labstack/echo/v5"
)

type RemoveHTTPRouteRequest struct {
	Reason             string `json:"reason" validate:"required,max=512"`
	AcknowledgeRemoval bool   `json:"acknowledge_removal" validate:"required"`
}

// removeHTTPRoute godoc
// @Summary Remove one HTTPRoute; preserve network Grants and Bindings
// @ID removeHTTPRoute
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param uid path string true "HTTPRoute UID, not name"
// @Param If-Match header string true "Strong current revision, checked before lookup"
// @Param removal body RemoveHTTPRouteRequest true "Strict JSON16KiB; reason and explicit acknowledgement"
// @Success 200 {object} management.InventoryChange
// @Header 200 {string} ETag "Strong accepted revision; not stream termination"
// @Failure 400,401,403,404,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/http-routes/{uid}/remove [post]
func (h *managementHTTP) removeHTTPRoute(c *echo.Context) error {
	p, err := h.principal(c, "publication:write")
	if err != nil {
		return mappedError(c, err)
	}
	revision, status := requiredRevision(c.Request())
	if status != 0 {
		return revisionError(c, status)
	}
	uid := c.Param("uid")
	if !grantUID.MatchString(uid) {
		return apiError(c, 400, "invalid_fields")
	}
	var dto RemoveHTTPRouteRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	result, err := h.service.RemoveHTTPRoute(p, revision, dto.Reason, uid)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, result.Revision))
	return c.JSON(200, result)
}

// browserRemoveHTTPRoute godoc
// @Summary Remove one HTTPRoute with operator session and CSRF protection
// @ID browserRemoveHTTPRoute
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param uid path string true "HTTPRoute UID, not name"
// @Param If-Match header string true "Strong current revision"
// @Param Origin header string true "Exact configured origin"
// @Param X-CSRF-Token header string true "Session CSRF token"
// @Param removal body RemoveHTTPRouteRequest true "Reason and explicit acknowledgement"
// @Success 200 {object} management.InventoryChange
// @Header 200 {string} ETag "Strong accepted revision; not stream termination"
// @Failure 400,401,403,404,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/http-routes/{uid}/remove [post]
func (h *managementHTTP) browserRemoveHTTPRoute(c *echo.Context) error { return h.removeHTTPRoute(c) }
