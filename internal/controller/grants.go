// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/strictjson"
	"github.com/labstack/echo/v5"
)

var grantUID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type IssueGrantRequest struct {
	Name            string   `json:"name" validate:"required,max=128"`
	Source          string   `json:"source" validate:"required,max=128"`
	ResourceName    string   `json:"resource_name" validate:"required,max=1024"`
	ResourceUID     string   `json:"resource_uid" validate:"required,max=128"`
	EndpointNames   []string `json:"endpoint_names" validate:"required,min=1,max=256,dive,required,max=1024"`
	DurationSeconds int      `json:"duration_seconds" validate:"required,gte=60,lte=86400"`
	Reason          string   `json:"reason" validate:"required,max=512"`
}
type RevokeGrantRequest struct {
	Reason string `json:"reason" validate:"required,max=512"`
}

func requiredRevision(r *http.Request) (uint64, int) {
	values := r.Header.Values("If-Match")
	if len(values) == 0 {
		return 0, 428
	}
	if len(values) != 1 {
		return 0, 400
	}
	value := values[0]
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		return 0, 400
	}
	digits := value[1 : len(value)-1]
	v, err := strconv.ParseUint(digits, 10, 63)
	if err != nil || strconv.FormatUint(v, 10) != digits {
		return 0, 400
	}
	return v, 0
}
func revisionError(c *echo.Context, status int) error {
	if status == 428 {
		return apiError(c, status, "revision_required")
	}
	return apiError(c, status, "invalid_revision")
}
func decodeGrantJSON(c *echo.Context, dst any) (int, string) {
	content, _, err := mime.ParseMediaType(c.Request().Header.Get("Content-Type"))
	if err != nil || content != "application/json" {
		return 415, "json_required"
	}
	data, err := io.ReadAll(http.MaxBytesReader(c.Response(), c.Request().Body, 16<<10))
	if err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			return 413, "body_too_large"
		}
		return 400, "invalid_json"
	}
	if err = strictjson.Decode(data, dst); err != nil {
		return 400, "invalid_json"
	}
	return 0, ""
}

// grants godoc
// @Summary Read source Grants and existing Resource catalog (desired intent only)
// @ID readGrants
// @Tags management
// @Security ManagementBearer
// @Produce json
// @Param after query string false "Exclusive Grant UID; continuation requires If-Match"
// @Param limit query integer false "Page size default50" minimum(1) maximum(100)
// @Param If-Match header string false "Strong first-page revision; mandatory with after"
// @Success 200 {object} management.GrantPage
// @Header 200 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,428,429,503 {object} APIError
// @Router /api/v1/grants [get]
func (h *managementHTTP) grants(c *echo.Context) error {
	p, err := h.principal(c, "publication:read")
	if err != nil {
		return mappedError(c, err)
	}
	q, err := url.ParseQuery(c.Request().URL.RawQuery)
	if err != nil {
		return apiError(c, 400, "invalid_query")
	}
	after, limit := "", 50
	for key, values := range q {
		if len(values) != 1 {
			return apiError(c, 400, "invalid_query")
		}
		switch key {
		case "after":
			after = values[0]
			if !grantUID.MatchString(after) {
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
	var expected *uint64
	if after != "" || len(c.Request().Header.Values("If-Match")) > 0 {
		v, status := requiredRevision(c.Request())
		if status != 0 {
			return revisionError(c, status)
		}
		expected = &v
	}
	page, err := h.service.Grants(p, after, limit, expected)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, page.Revision))
	return c.JSON(200, page)
}

// issueGrant godoc
// @Summary Issue one independent bounded source Grant to explicit existing Endpoints
// @Description Absolute expiry is calculated once by the server. This is desired state, not applied evidence. Strict JSON max16KiB. Duplicate Grant name rejected. No automatic retry or renewal.
// @ID issueGrant
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong current revision"
// @Param grant body IssueGrantRequest true "Pinned resource, named endpoints, duration60..86400s and reason"
// @Success 201 {object} management.GrantChange
// @Header 201 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/grants [post]
func (h *managementHTTP) issueGrant(c *echo.Context) error {
	p, err := h.principal(c, "publication:write")
	if err != nil {
		return mappedError(c, err)
	}
	revision, status := requiredRevision(c.Request())
	if status != 0 {
		return revisionError(c, status)
	}
	var dto IssueGrantRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	result, err := h.service.IssueGrant(p, revision, dto.Reason, management.IssueGrant{Name: dto.Name, Source: dto.Source, ResourceName: dto.ResourceName, ResourceUID: dto.ResourceUID, EndpointNames: dto.EndpointNames, DurationSeconds: dto.DurationSeconds})
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, result.Revision))
	return c.JSON(201, result)
}

// revokeGrant godoc
// @Summary Revoke only the identified Grant without changing other contributions
// @ID revokeGrant
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param uid path string true "Grant UID (not NetworkDeny)"
// @Param If-Match header string true "Strong current revision, checked before idempotency"
// @Param revoke body RevokeGrantRequest true "Audit reason"
// @Success 200 {object} management.GrantChange
// @Header 200 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,404,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/grants/{uid}/revoke [post]
func (h *managementHTTP) revokeGrant(c *echo.Context) error {
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
	var dto RevokeGrantRequest
	if status, code := decodeGrantJSON(c, &dto); status != 0 {
		return apiError(c, status, code)
	}
	if err = h.validate.Struct(dto); err != nil {
		return apiError(c, 400, "invalid_fields")
	}
	result, err := h.service.RevokeGrant(p, revision, dto.Reason, uid)
	if err != nil {
		return mappedError(c, err)
	}
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%d"`, result.Revision))
	return c.JSON(200, result)
}

// browserGrants godoc
// @Summary Read source Grants in the operator session
// @ID browserGrants
// @Tags console
// @Security BrowserCookie
// @Produce json
// @Param after query string false "Exclusive UID; requires If-Match"
// @Param limit query integer false "Default50,max100"
// @Param If-Match header string false "Strong revision, required with after"
// @Success 200 {object} management.GrantPage
// @Failure 400,401,403,412,428,429,503 {object} APIError
// @Router /console/api/v1/grants [get]
func (h *managementHTTP) browserGrants(c *echo.Context) error { return h.grants(c) }

// browserIssueGrant godoc
// @Summary Issue a bounded source Grant in the operator session
// @ID browserIssueGrant
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param If-Match header string true "Strong revision"
// @Param Origin header string true "Exact configured public origin"
// @Param X-CSRF-Token header string false "Session CSRF token (legacy clients)"
// @Param grant body IssueGrantRequest true "Grant and reason"
// @Success 201 {object} management.GrantChange
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/grants [post]
func (h *managementHTTP) browserIssueGrant(c *echo.Context) error { return h.issueGrant(c) }

// browserRevokeGrant godoc
// @Summary Revoke a source Grant in the operator session
// @ID browserRevokeGrant
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param uid path string true "Grant UID"
// @Param If-Match header string true "Strong revision"
// @Param Origin header string true "Exact configured public origin"
// @Param X-CSRF-Token header string false "Session CSRF token (legacy clients)"
// @Param revoke body RevokeGrantRequest true "Reason"
// @Success 200 {object} management.GrantChange
// @Failure 400,401,403,404,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/grants/{uid}/revoke [post]
func (h *managementHTTP) browserRevokeGrant(c *echo.Context) error { return h.revokeGrant(c) }
