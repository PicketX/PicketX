// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// NewHandler serves process liveness only. Authorization is not implemented yet.
func NewHandler() http.Handler {
	e := echo.New()
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
	return e
}
