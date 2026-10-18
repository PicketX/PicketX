// SPDX-License-Identifier: Apache-2.0
package controller

import "github.com/labstack/echo/v5"

// agentObservations godoc
// @Summary Read credentialed Agent scopes and unverified execution assertions
// @Description Process-local point-in-time view. APPLIED is Agent-reported, not kernel proof. Freshness uses Controller receive time and current reporting session, strictly below60 seconds. matches_desired is separate; new revision or reconnect never refreshes an old report. No heartbeat guarantee. Revisions are decimal strings. No query parameters.
// @ID agentObservations
// @Tags management
// @Security ManagementBearer
// @Produce json
// @Success 200 {object} agentsync.Observations
// @Failure 400,401,403,429,503 {object} APIError
// @Router /api/v1/agent-observations [get]
func (h *managementHTTP) agentObservations(c *echo.Context) error {
	p, e := h.principal(c, "publication:read")
	if e != nil {
		return mappedError(c, e)
	}
	if c.Request().URL.RawQuery != "" {
		return apiError(c, 400, "invalid_query")
	}
	v, e := h.service.AgentObservations(p)
	if e != nil {
		return mappedError(c, e)
	}
	return c.JSON(200, v)
}

// browserAgentObservations godoc
// @Summary Read Agent observations with current operator cookie identity
// @Description Same assertion/freshness boundaries as the Bearer route. publication:read required. No query parameters.
// @ID browserAgentObservations
// @Tags browser
// @Security BrowserCookie
// @Produce json
// @Success 200 {object} agentsync.Observations
// @Failure 400,401,403,429,503 {object} APIError
// @Router /console/api/v1/agent-observations [get]
func (h *managementHTTP) browserAgentObservations(c *echo.Context) error {
	return h.agentObservations(c)
}
