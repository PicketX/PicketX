// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"fmt"
	"github.com/PicketX/PicketX/pkg/configuration"
	"github.com/labstack/echo/v5"
)

type UpdateResourceNetworkRequest struct {
	Endpoints []ResourceEndpointRequest `json:"endpoints" validate:"required,min=1,max=64,dive"`
	Reason    string                    `json:"reason" validate:"required,max=512"`
}

// updateResourceNetwork godoc
// @Summary Edit existing Endpoint networks only on an unreferenced Resource
// @Description Keeps UID/name/Endpoint membership/type/labels. Normalizes selected document; does not configure the service or confirm enforcement. All references including revoked/expired block.
// @ID updateResourceNetwork
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param uid path string true "Resource UID, never name"
// @Param If-Match header string true "Strong current revision; stale checked before lookup"
// @Param network body UpdateResourceNetworkRequest true "Complete existing named Endpoints; omitted addresses use binding scope; empty/null rejected"
// @Success 200 {object} management.InventoryChange
// @Header 200 {string} ETag "Strong accepted revision, not applied state"
// @Failure 400,401,403,404,409,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/resources/{uid}/network [put]
func (h *managementHTTP) updateResourceNetwork(c *echo.Context) error {
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
	var dto UpdateResourceNetworkRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	entries := make([]configuration.InventoryEndpoint, 0, len(dto.Endpoints))
	for _, e := range dto.Endpoints {
		entries = append(entries, configuration.InventoryEndpoint{Name: e.Name, Protocol: e.Protocol, Port: e.Port, Addresses: e.Addresses})
	}
	result, err := h.service.UpdateResourceNetwork(p, revision, dto.Reason, uid, entries)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, result.Revision))
	return c.JSON(200, result)
}

// browserUpdateResourceNetwork godoc
// @Summary Edit unreferenced Resource networks with operator session and CSRF
// @ID browserUpdateResourceNetwork
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param uid path string true "Immutable Resource UID"
// @Param If-Match header string true "Strong current revision"
// @Param Origin header string true "Exact configured origin"
// @Param X-CSRF-Token header string true "Session CSRF token"
// @Param network body UpdateResourceNetworkRequest true "Complete existing Endpoint networks and audit reason"
// @Success 200 {object} management.InventoryChange
// @Header 200 {string} ETag "Strong accepted revision, not applied state"
// @Failure 400,401,403,404,409,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/resources/{uid}/network [put]
func (h *managementHTTP) browserUpdateResourceNetwork(c *echo.Context) error {
	return h.updateResourceNetwork(c)
}
