// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"fmt"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/pkg/configuration"
	"github.com/labstack/echo/v5"
)

type ResourceEndpointRequest struct {
	Name      string   `json:"name" validate:"required,max=128"`
	Protocol  string   `json:"protocol" validate:"required,oneof=TCP UDP"`
	Port      uint16   `json:"port" validate:"required,min=1"`
	Addresses []string `json:"addresses,omitempty" validate:"omitempty,min=1,max=64,dive,required,max=128"`
}
type CreateResourceRequest struct {
	Name      string                    `json:"name" validate:"required,max=128"`
	Endpoints []ResourceEndpointRequest `json:"endpoints" validate:"required,min=1,max=64,dive"`
	Reason    string                    `json:"reason" validate:"required,max=512"`
}
type CreateBindingRequest struct {
	Name          string   `json:"name" validate:"required,max=128"`
	ResourceName  string   `json:"resource_name" validate:"required,max=1024"`
	ResourceUID   string   `json:"resource_uid" validate:"required,max=128"`
	EndpointNames []string `json:"endpoint_names" validate:"required,min=1,max=64,dive,required,max=1024"`
	AgentName     string   `json:"agent_name" validate:"required,max=1024"`
	ContextID     string   `json:"context_id" validate:"required,max=1024"`
	Reason        string   `json:"reason" validate:"required,max=512"`
}

// inventory godoc
// @Summary Read complete bounded desired Resource/Binding/context inventory (not live status)
// @ID readInventory
// @Tags management
// @Security ManagementBearer
// @Produce json
// @Success 200 {object} management.InventoryPage
// @Header 200 {string} ETag "Strong publication revision"
// @Failure 400,401,403,429,503 {object} APIError
// @Router /api/v1/inventory [get]
func (h *managementHTTP) inventory(c *echo.Context) error {
	p, err := h.principal(c, "publication:read")
	if err != nil {
		return mappedError(c, err)
	}
	if c.Request().URL.RawQuery != "" {
		return apiError(c, 400, "invalid_query")
	}
	page, err := h.service.Inventory(p)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, page.Revision))
	return c.JSON(200, page)
}

// createResource godoc
// @Summary Create a Resource with named network Endpoints; does not grant or confirm access
// @ID createResource
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong current revision"
// @Param resource body CreateResourceRequest true "Strict JSON16KiB; omitted addresses use binding scope, empty/null rejected"
// @Success 201 {object} management.InventoryChange
// @Header 201 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/resources [post]
func (h *managementHTTP) createResource(c *echo.Context) error {
	p, err := h.principal(c, "publication:write")
	if err != nil {
		return mappedError(c, err)
	}
	revision, status := requiredRevision(c.Request())
	if status != 0 {
		return revisionError(c, status)
	}
	var dto CreateResourceRequest
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
	result, err := h.service.CreateResource(p, revision, dto.Reason, dto.Name, entries)
	if err != nil {
		return mappedError(c, err)
	}
	return inventoryCreated(c, result)
}

// createBinding godoc
// @Summary Bind named Endpoints to an exact existing host context without changing its settings
// @ID createBinding
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong current revision"
// @Param binding body CreateBindingRequest true "Pinned resource, explicit endpoints, agent/context identity, audit reason"
// @Success 201 {object} management.InventoryChange
// @Header 201 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/bindings [post]
func (h *managementHTTP) createBinding(c *echo.Context) error {
	p, err := h.principal(c, "publication:write")
	if err != nil {
		return mappedError(c, err)
	}
	revision, status := requiredRevision(c.Request())
	if status != 0 {
		return revisionError(c, status)
	}
	var dto CreateBindingRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	result, err := h.service.CreateBinding(p, revision, dto.Reason, configuration.BindingInput{Name: dto.Name, ResourceName: dto.ResourceName, ResourceUID: dto.ResourceUID, EndpointNames: dto.EndpointNames, AgentName: dto.AgentName, ContextID: dto.ContextID})
	if err != nil {
		return mappedError(c, err)
	}
	return inventoryCreated(c, result)
}
func inventoryCreated(c *echo.Context, result management.InventoryChange) error {
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, result.Revision))
	return c.JSON(201, result)
}

// browserInventory godoc
// @Summary Read desired resource inventory in the operator session
// @ID browserInventory
// @Tags console
// @Security BrowserCookie
// @Produce json
// @Success 200 {object} management.InventoryPage
// @Header 200 {string} ETag "Strong revision"
// @Failure 400,401,403,429,503 {object} APIError
// @Router /console/api/v1/inventory [get]
func (h *managementHTTP) browserInventory(c *echo.Context) error { return h.inventory(c) }

// browserCreateResource godoc
// @Summary Create Resource using the operator session and CSRF protection
// @ID browserCreateResource
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong revision"
// @Param Origin header string true "Exact configured origin"
// @Param X-CSRF-Token header string true "Session CSRF token"
// @Param resource body CreateResourceRequest true "Resource and reason"
// @Success 201 {object} management.InventoryChange
// @Header 201 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/resources [post]
func (h *managementHTTP) browserCreateResource(c *echo.Context) error { return h.createResource(c) }

// browserCreateBinding godoc
// @Summary Create explicit-context Binding using the operator session and CSRF protection
// @ID browserCreateBinding
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong revision"
// @Param Origin header string true "Exact configured origin"
// @Param X-CSRF-Token header string true "Session CSRF token"
// @Param binding body CreateBindingRequest true "Binding and reason"
// @Success 201 {object} management.InventoryChange
// @Header 201 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/bindings [post]
func (h *managementHTTP) browserCreateBinding(c *echo.Context) error { return h.createBinding(c) }
