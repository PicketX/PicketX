// SPDX-License-Identifier: Apache-2.0
package controller

import "github.com/labstack/echo/v5"

// browserMe godoc
// @Summary Read current trusted session identity and allowed operations
// @ID browserIdentity
// @Tags console
// @Security BrowserCookie
// @Produce json
// @Success 200 {object} IdentityResponse
// @Failure 400,401,403,429,503 {object} APIError
// @Router /console/api/v1/me [get]
func (h *managementHTTP) browserMe(c *echo.Context) error { return h.me(c) }

// browserPublication godoc
// @Summary Read accepted desired configuration, not Agent application
// @ID browserPublication
// @Tags console
// @Security BrowserCookie
// @Produce json
// @Success 200 {object} publication.Document
// @Header 200 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,429,503 {object} APIError
// @Router /console/api/v1/publication [get]
func (h *managementHTTP) browserPublication(c *echo.Context) error { return h.getPublication(c) }

// browserReplace godoc
// @Summary Atomically save desired configuration and audit with revision precondition
// @Description Requires exact configured public Origin. Native Echo CSRF accepts same-origin Sec-Fetch-Site; legacy clients must send the bootstrap token in X-CSRF-Token with its HttpOnly CSRF cookie. Body limit12MiB; unknown/duplicate/null fields rejected. Success is NOT node application.
// @ID browserReplace
// @Tags console
// @Security BrowserCookie
// @Accept json
// @Produce json
// @Param Origin header string true "Exact configured public origin"
// @Param X-CSRF-Token header string false "Bootstrap csrf token (legacy clients)"
// @Param If-Match header string true "Strong current revision"
// @Param publication body PublicationRequest true "Complete desired configuration and reason"
// @Success 200 {object} publication.Document
// @Header 200 {string} ETag "Strong accepted revision"
// @Failure 400,401,403,412,413,415,428,429,503 {object} APIError
// @Router /console/api/v1/publication [put]
func (h *managementHTTP) browserReplace(c *echo.Context) error { return h.putPublication(c) }

// browserAudit godoc
// @Summary Read bounded append-only publication audit (exclusive cursor)
// @ID browserAudit
// @Tags console
// @Security BrowserCookie
// @Produce json
// @Param after query integer false "Exclusive audit event cursor"
// @Param limit query integer false "1..100, default50"
// @Success 200 {object} AuditResponse
// @Failure 400,401,403,429,503 {object} APIError
// @Router /console/api/v1/audit [get]
func (h *managementHTTP) browserAudit(c *echo.Context) error { return h.audit(c) }
