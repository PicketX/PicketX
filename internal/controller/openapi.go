// SPDX-License-Identifier: Apache-2.0
package controller

// @title PicketX management API
// @version 1alpha1
// @description Database-authoritative desired-state publication. External ingress owns HTTPS; success is not Agent application. Bearer routes accept only independent Tokens. Opt-in console routes accept only SCS cookies, check public Host/unsafe Origin, and use native Echo CSRF (Sec-Fetch-Site or legacy cookie/header). All routes are disabled in file mode.
// @securityDefinitions.apikey ManagementBearer
// @in header
// @name Authorization

// @securityDefinitions.apikey BrowserCookie
// @in cookie
// @name picketx_session

// @securityDefinitions.apikey IngressBearer
// @in header
// @name Authorization
// @externalDocs.description Management deployment and limitations
// @externalDocs.url https://github.com/PicketX/PicketX/blob/main/docs/MANAGEMENT_API.md
