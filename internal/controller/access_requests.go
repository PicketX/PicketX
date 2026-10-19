// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"fmt"
	"github.com/PicketX/PicketX/internal/accessrequest"
	"github.com/labstack/echo/v5"
	"net/url"
	"strconv"
)

type CreateAccessRequest struct {
	Source          string   `json:"source" validate:"required,max=128"`
	ResourceName    string   `json:"resource_name" validate:"required,max=1024"`
	ResourceUID     string   `json:"resource_uid" validate:"required,max=128"`
	EndpointNames   []string `json:"endpoint_names" validate:"required,min=1,max=64,dive,required,max=1024"`
	DurationSeconds int      `json:"duration_seconds" validate:"required,gte=60,lte=86400"`
	Reason          string   `json:"reason" validate:"required,max=512"`
}
type DecideAccessRequest struct {
	Version       uint64   `json:"version" validate:"required,gte=1,lte=2147483647"`
	Action        string   `json:"action" validate:"required,oneof=approve reject"`
	EndpointNames []string `json:"endpoint_names,omitempty" validate:"omitempty,min=1,max=64,dive,required,max=1024"`
	ExpiresAt     string   `json:"expires_at,omitempty" validate:"omitempty,max=64"`
	Reason        string   `json:"reason" validate:"required,max=400"`
}

// requestCatalog godoc
// @Summary Limited requestable Resource catalog, no policy/facts or permissions
// @ID accessRequestCatalog
// @Tags requests
// @Security ManagementBearer
// @Produce json
// @Success 200 {object} management.RequestCatalog
// @Header 200 {string} ETag "Current publication revision"
// @Failure 401,403,429,503 {object} APIError
// @Router /api/v1/access-request-catalog [get]
// @Router /console/api/v1/access-request-catalog [get]
func (h *managementHTTP) requestCatalog(c *echo.Context) error {
	p, err := h.principal(c, "request:create")
	if err != nil {
		return mappedError(c, err)
	}
	out, err := h.service.RequestCatalog(p)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, out.Revision))
	return c.JSON(200, out)
}

// createAccessRequest godoc
// @Summary Submit an immutable source request; creates no permission
// @ID createAccessRequest
// @Tags requests
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param If-Match header string true "Current catalog publication revision"
// @Param request body CreateAccessRequest true "One explicit CIDR and pinned named entries; server fixes deadline once"
// @Success 201 {object} accessrequest.Record
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/access-requests [post]
// @Router /console/api/v1/access-requests [post]
func (h *managementHTTP) createAccessRequest(c *echo.Context) error {
	p, err := h.principal(c, "request:create")
	if err != nil {
		return mappedError(c, err)
	}
	expected, status := requiredRevision(c.Request())
	if status != 0 {
		return revisionError(c, status)
	}
	var dto CreateAccessRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	out, err := h.service.CreateAccessRequest(p, expected, accessrequest.Input{Source: dto.Source, ResourceName: dto.ResourceName, ResourceUID: dto.ResourceUID, EndpointNames: dto.EndpointNames, DurationSeconds: dto.DurationSeconds, Reason: dto.Reason})
	if err != nil {
		return mappedError(c, err)
	}
	return c.JSON(201, out)
}

// accessRequests godoc
// @Summary Bounded own requests or authorized review inbox; desired state, not application
// @ID listAccessRequests
// @Tags requests
// @Security ManagementBearer
// @Produce json
// @Param after query integer false "Exclusive record ID" minimum(0)
// @Param limit query integer false "Page size default50" minimum(1) maximum(100)
// @Success 200 {object} accessrequest.Page
// @Failure 400,401,403,429,503 {object} APIError
// @Router /api/v1/access-requests [get]
// @Router /console/api/v1/access-requests [get]
func (h *managementHTTP) accessRequests(c *echo.Context) error {
	p, err := h.principal(c, "")
	if err != nil {
		return mappedError(c, err)
	}
	q, err := url.ParseQuery(c.Request().URL.RawQuery)
	if err != nil {
		return apiError(c, 400, "invalid_query")
	}
	after, limit := int64(0), 50
	for key, values := range q {
		if len(values) != 1 {
			return apiError(c, 400, "invalid_query")
		}
		switch key {
		case "after":
			after, err = strconv.ParseInt(values[0], 10, 64)
			if err != nil || after < 0 {
				return apiError(c, 400, "invalid_query")
			}
		case "limit":
			limit, err = strconv.Atoi(values[0])
			if err != nil || limit < 1 || limit > 100 {
				return apiError(c, 400, "invalid_query")
			}
		default:
			return apiError(c, 400, "invalid_query")
		}
	}
	out, err := h.service.AccessRequests(p, after, limit)
	if err != nil {
		return mappedError(c, err)
	}
	return c.JSON(200, out)
}

// decideAccessRequest godoc
// @Summary Independently approve a narrowed absolute Grant or reject a pending request
// @Description Approval, workflow event, publication and audit commit atomically; success is NOT Agent application. Self-approval forbidden. Rejection needs no publication If-Match or target existence.
// @ID decideAccessRequest
// @Tags requests
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param uid path string true "Request UID"
// @Param If-Match header string false "Required for approve: current publication revision"
// @Param decision body DecideAccessRequest true "Pending version, action, reason; approval requires subset and absolute expires_at"
// @Success 200 {object} accessrequest.Record
// @Header 200 {string} ETag "Approved publication revision only"
// @Failure 400,401,403,404,409,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/access-requests/{uid}/decision [post]
// @Router /console/api/v1/access-requests/{uid}/decision [post]
func (h *managementHTTP) decideAccessRequest(c *echo.Context) error {
	p, err := h.principal(c, "request:review")
	if err != nil {
		return mappedError(c, err)
	}
	uid := c.Param("uid")
	if !grantUID.MatchString(uid) {
		return apiError(c, 400, "invalid_fields")
	}
	var dto DecideAccessRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	var expected uint64
	if dto.Action == "approve" {
		var status int
		expected, status = requiredRevision(c.Request())
		if status != 0 {
			return revisionError(c, status)
		}
	}
	out, err := h.service.DecideAccessRequest(p, uid, expected, accessrequest.Decision{Version: dto.Version, Action: dto.Action, EndpointNames: dto.EndpointNames, ExpiresAt: dto.ExpiresAt, Reason: dto.Reason})
	if err != nil {
		return mappedError(c, err)
	}
	if out.State == "approved" {
		c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, out.Revision))
	}
	return c.JSON(200, out)
}
