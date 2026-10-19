// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"fmt"
	"github.com/labstack/echo/v5"
)

type UpdateBindingEndpointsRequest struct {
	EndpointNames          []string `json:"endpoint_names" validate:"required,min=1,max=64,dive,required,max=1024"`
	Reason                 string   `json:"reason" validate:"required,max=512"`
	AcknowledgeScopeChange bool     `json:"acknowledge_scope_change" validate:"required"`
}

// updateBindingEndpoints godoc
// @Summary Edit protected Endpoint names on a fixed-scope host Binding
// @Description Explicit single Resource/Agent host-local original-destination Bindings only. Keeps identity/metadata/Agent/enforcement and other authorizations. Adding protection may deny traffic; removing protection may expose traffic. Publication is not application.
// @ID updateBindingEndpoints
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param uid path string true "Binding UID, never name"
// @Param If-Match header string true "Strong current revision; stale checked before lookup"
// @Param endpoints body UpdateBindingEndpointsRequest true "Complete nonempty Endpoint names, acknowledgement and audit reason"
// @Success 200 {object} management.InventoryChange
// @Header 200 {string} ETag "Strong accepted revision, not applied state"
// @Failure 400,401,403,404,409,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/bindings/{uid}/endpoints [put]
func (h *managementHTTP) updateBindingEndpoints(c *echo.Context) error {
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
	var dto UpdateBindingEndpointsRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	result, err := h.service.UpdateBindingEndpoints(p, revision, dto.Reason, uid, dto.EndpointNames)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, result.Revision))
	return c.JSON(200, result)
}

// browserUpdateBindingEndpoints godoc
// @Summary Edit protected Endpoint names with operator session and CSRF
// @ID browserUpdateBindingEndpoints
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param uid path string true "Immutable Binding UID"
// @Param If-Match header string true "Strong current revision"
// @Param Origin header string true "Exact configured origin"
// @Param X-CSRF-Token header string true "Session CSRF token"
// @Param endpoints body UpdateBindingEndpointsRequest true "Complete named selection, acknowledgement and audit reason"
// @Success 200 {object} management.InventoryChange
// @Header 200 {string} ETag "Strong accepted revision, not applied state"
// @Failure 400,401,403,404,409,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/bindings/{uid}/endpoints [put]
func (h *managementHTTP) browserUpdateBindingEndpoints(c *echo.Context) error {
	return h.updateBindingEndpoints(c)
}
