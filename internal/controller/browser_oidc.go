// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"github.com/PicketX/PicketX/internal/oidclogin"
	"github.com/labstack/echo/v5"
	"net/http"
	"net/url"
)

// SetOIDC is assembly-only: inject before registering/serving any handler.
func (b *Browser) SetOIDC(client *oidclogin.Client) { b.oidc = client }

const oidcCookiePath = "/console/api/v1/oidc/callback"

func (b *Browser) oidcCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{Name: "picketx_oidc", Value: value, Path: oidcCookiePath, MaxAge: maxAge, HttpOnly: true, Secure: b.origin.Scheme == "https", SameSite: http.SameSiteLaxMode}
}

// startOIDC godoc
// @Summary Start one browser-bound OIDC authorization-code transaction
// @Description Exact Origin/Host and Echo CSRF required. Revokes the originating local session before external navigation. Pending5min, max128, S256 PKCE and nonce. No external token is returned; provider is opt-in and local roles stay authoritative.
// @ID browserOIDCStart
// @Tags console
// @Produce json
// @Param Origin header string true "Exact public origin"
// @Param X-CSRF-Token header string false "Bootstrap token for legacy clients"
// @Success 200 {object} OIDCStartResponse
// @Failure 400,403,429,503 {object} APIError
// @Router /console/api/v1/oidc/start [post]
func (b *Browser) startOIDC(c *echo.Context) error {
	// The external callback omits the Strict SCS cookie. Revoke its originating
	// session here, under wrap's same-token lock, before emitting a usable URL.
	if err := b.sessions.Destroy(c.Request().Context()); err != nil {
		return apiError(c, 503, "session_unavailable")
	}
	auth, _, correlation, e := b.oidc.Start()
	if e != nil {
		return apiError(c, 503, "oidc_unavailable")
	}
	c.SetCookie(b.oidcCookie(correlation, 300))
	return c.JSON(200, OIDCStartResponse{URL: auth})
}

type OIDCStartResponse struct {
	URL string `json:"url"`
}

// callbackOIDC godoc
// @Summary Consume one browser-bound verified OIDC code and renew local SCS session
// @Description GET callback permits cross-site top-level navigation; one-use state, path-scoped HttpOnly Lax correlation, PKCE/nonce/issuer/audience/signature/expiry are mandatory. Fixed local success/failure redirect; no provider diagnostics or tokens returned. Local logout does not end the IdP session.
// @ID browserOIDCCallback
// @Tags console
// @Param state query string true "One-use state"
// @Param code query string true "Authorization code"
// @Success 303 "Fixed console redirect"
// @Failure 400,403,429,503 {object} APIError
// @Router /console/api/v1/oidc/callback [get]
func (b *Browser) callbackOIDC(c *echo.Context) error {
	c.SetCookie(b.oidcCookie("", -1))
	fail := func() error { return c.Redirect(http.StatusSeeOther, "/console/?login_error=oidc") }
	if len(c.Request().URL.RawQuery) > 8192 {
		return fail()
	}
	q, e := url.ParseQuery(c.Request().URL.RawQuery)
	if e != nil || len(q["state"]) != 1 || len(q["code"]) != 1 || q.Has("error") {
		return fail()
	}
	correlation := ""
	count := 0
	for _, cookie := range c.Request().Cookies() {
		if cookie.Name == "picketx_oidc" {
			correlation = cookie.Value
			count++
		}
	}
	if count != 1 {
		return fail()
	}
	proof, e := b.oidc.Complete(c.Request().Context(), q.Get("state"), correlation, q.Get("code"))
	if e != nil {
		return fail()
	}
	p, fingerprint, e := b.service.AuthenticateHuman(proof.PrincipalID)
	if e != nil {
		return fail()
	}
	if b.saveSession(c, p, fingerprint) != nil {
		return fail()
	}
	ctx := c.Request().Context()
	b.sessions.Put(ctx, "oidc_subject", proof.Subject)
	b.sessions.Put(ctx, "oidc_binding", proof.Fingerprint)
	b.sessions.Put(ctx, "oidc_expiry", proof.ExpiresAt.Unix())
	return c.Redirect(http.StatusSeeOther, "/console/")
}
