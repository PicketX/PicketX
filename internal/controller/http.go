// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/PicketX/PicketX/internal/management"
	"net/http"

	"github.com/labstack/echo/v5"
)

// NewHandler serves process liveness only. Authorization is not implemented yet.
func NewHandler() http.Handler {
	return NewHandlerWithReadiness(nil)
}

// NewHandlerWithReadiness adds a detail-free module readiness probe.
func NewHandlerWithReadiness(ready func() bool) http.Handler {
	return NewHandlerWithManagement(ready, nil, ManagementLimits{})
}

func NewHandlerWithManagement(ready func() bool, service *management.Service, limits ManagementLimits) http.Handler {
	return NewHandlerWithBrowser(ready, service, limits, nil)
}

func NewHandlerWithBrowser(ready func() bool, service *management.Service, limits ManagementLimits, browser *Browser) http.Handler {
	e := echo.New()
	if service != nil {
		registerManagement(e, service, limits)
	}
	if browser != nil {
		browser.register(e, limits)
	}
	e.GET("/healthz", func(c *echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store")
		return c.JSON(http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ok"})
	})
	e.HEAD("/healthz", func(c *echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("Content-Type", "application/json")
		return c.NoContent(http.StatusOK)
	})
	if ready != nil {
		probe := func(c *echo.Context) error {
			c.Response().Header().Set("Cache-Control", "no-store")
			code := http.StatusOK
			if !ready() {
				code = http.StatusServiceUnavailable
			}
			return c.NoContent(code)
		}
		e.GET("/readyz", probe)
		e.HEAD("/readyz", probe)
	}
	if browser != nil {
		return browser.wrap(e)
	}
	return e
}
