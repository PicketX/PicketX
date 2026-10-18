// SPDX-License-Identifier: Apache-2.0
package controller

import (
 "fmt"
 "github.com/labstack/echo/v5"
)

type RemoveResourceRequest struct {
 Reason string `json:"reason" validate:"required,max=512"`
 AcknowledgeRemoval bool `json:"acknowledge_removal" validate:"required"`
}

// removeResource godoc
// @Summary Remove an unreferenced Resource definition; never cascade or stop services
// @ID removeResource
// @Tags management
// @Security ManagementBearer
// @Accept json
// @Produce json
// @Param uid path string true "Resource UID, not name"
// @Param If-Match header string true "Strong current revision, checked before lookup"
// @Param removal body RemoveResourceRequest true "Reason and explicit removal acknowledgement"
// @Success 200 {object} management.InventoryChange
// @Header 200 {string} ETag "Strong accepted revision, not applied state"
// @Failure 400,401,403,404,409,412,413,415,428,429,503 {object} APIError
// @Router /api/v1/resources/{uid}/remove [post]
func (h *managementHTTP) removeResource(c *echo.Context) error {
 p,err:=h.principal(c,"publication:write")
 if err!=nil {return mappedError(c,err)}
 revision,status:=requiredRevision(c.Request())
 if status!=0 {return revisionError(c,status)}
 uid:=c.Param("uid")
 if !grantUID.MatchString(uid) {return apiError(c,400,"invalid_fields")}
 var dto RemoveResourceRequest
 if status,code:=decodeGrantJSON(c,&dto);status!=0 {return apiError(c,status,code)}
 if err=h.validate.Struct(dto);err!=nil {return apiError(c,400,"invalid_fields")}
 result,err:=h.service.RemoveResource(p,revision,dto.Reason,uid)
 if err!=nil {return mappedError(c,err)}
 c.Response().Header().Set("ETag",fmt.Sprintf(`"%d"`,result.Revision))
 return c.JSON(200,result)
}

// browserRemoveResource godoc
// @Summary Remove an unreferenced Resource with operator session; never cascade
// @ID browserRemoveResource
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param uid path string true "Resource UID"
// @Param If-Match header string true "Strong current revision"
// @Param Origin header string true "Exact configured origin"
// @Param X-CSRF-Token header string true "Session CSRF token"
// @Param removal body RemoveResourceRequest true "Reason and explicit removal acknowledgement"
// @Success 200 {object} management.InventoryChange
// @Header 200 {string} ETag "Strong accepted revision, not applied state"
// @Failure 400,401,403,404,409,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/resources/{uid}/remove [post]
func (h *managementHTTP) browserRemoveResource(c *echo.Context) error {return h.removeResource(c)}
