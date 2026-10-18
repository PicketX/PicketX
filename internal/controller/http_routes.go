// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"github.com/PicketX/PicketX/pkg/configuration"
	"github.com/labstack/echo/v5"
)

type CreateHTTPRouteRequest struct {
	Name         string   `json:"name" validate:"required,max=128"`
	ResourceName string   `json:"resource_name" validate:"required,max=1024"`
	ResourceUID  string   `json:"resource_uid" validate:"required,max=128"`
	EndpointName string   `json:"endpoint_name" validate:"required,max=1024"`
	Host         string   `json:"host" validate:"required,max=253"`
	Methods      []string `json:"methods" validate:"required,min=1,max=7,unique,dive,oneof=GET HEAD POST PUT PATCH DELETE OPTIONS"`
	Paths        []string `json:"paths" validate:"required,min=1,max=128,unique,dive,required,max=8192"`
	Reason       string   `json:"reason" validate:"required,max=512"`
}

// createHTTPRoute godoc
// @Summary Create an exact HTTPRoute for a pinned TCP endpoint; does not grant access or configure ingress
// @ID createHTTPRoute
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong current revision"
// @Param route body CreateHTTPRouteRequest true "Strict JSON16KiB; explicit exact host, methods and paths; server-generated UID"
// @Success 201 {object} management.InventoryChange
// @Header 201 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/http-routes [post]
func (h *managementHTTP) createHTTPRoute(c *echo.Context) error {
	p, err := h.principal(c, "publication:write")
	if err != nil {
		return mappedError(c, err)
	}
	revision, status := requiredRevision(c.Request())
	if status != 0 {
		return revisionError(c, status)
	}
	var dto CreateHTTPRouteRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	result, err := h.service.CreateHTTPRoute(p, revision, dto.Reason, configuration.InventoryHTTPRoute{
		Name: dto.Name, ResourceName: dto.ResourceName, ResourceUID: dto.ResourceUID, EndpointName: dto.EndpointName, Host: dto.Host, Methods: dto.Methods, Paths: dto.Paths,
	})
	if err != nil {
		return mappedError(c, err)
	}
	return inventoryCreated(c, result)
}

// browserCreateHTTPRoute godoc
// @Summary Create an exact HTTPRoute using operator session and CSRF protection
// @ID browserCreateHTTPRoute
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong current revision"
// @Param Origin header string true "Exact configured origin"
// @Param X-CSRF-Token header string true "Session CSRF token"
// @Param route body CreateHTTPRouteRequest true "Pinned TCP endpoint, exact selectors and audit reason"
// @Success 201 {object} management.InventoryChange
// @Header 201 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/http-routes [post]
func (h *managementHTTP) browserCreateHTTPRoute(c *echo.Context) error { return h.createHTTPRoute(c) }
