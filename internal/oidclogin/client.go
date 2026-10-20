// SPDX-License-Identifier: Apache-2.0
package oidclogin

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type providerConfig struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Bindings     string `json:"bindings"`
}
type transaction struct {
	correlation, nonce, verifier string
	expires                      time.Time
}
type Identity struct {
	Subject, PrincipalID, Fingerprint string
	ExpiresAt                         time.Time
}
type Client struct {
	issuer, bindings string
	oauth            oauth2.Config
	verifier         *oidc.IDTokenVerifier
	http             *http.Client
	mu               sync.Mutex
	pending          map[string]transaction
	now              func() time.Time
}

// Provider HTTP is bounded and never follows redirects carrying client secrets.
type boundedTransport struct{ base http.RoundTripper }

func (t boundedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, e := t.base.RoundTrip(r)
	if e != nil {
		return nil, ErrUnavailable
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 {
		return nil, ErrUnavailable
	}
	response.Body = io.NopCloser(bytes.NewReader(raw))
	return response, nil
}
func endpoint(value string) bool {
	u, e := url.Parse(value)
	if e != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	ip, e := netip.ParseAddr(u.Hostname())
	return u.Scheme == "http" && e == nil && ip.IsLoopback() && !ip.Is4In6() && ip.Zone() == ""
}
func New(ctx context.Context, configPath, origin string) (*Client, error) {
	var cfg providerConfig
	if !filepath.IsAbs(configPath) || readPrivate(configPath, &cfg) != nil || !endpoint(cfg.Issuer) || strings.Contains(cfg.Issuer, "?") || len(cfg.ClientID) < 1 || len(cfg.ClientID) > 256 || len(cfg.ClientSecret) < 1 || len(cfg.ClientSecret) > 4096 || !filepath.IsAbs(cfg.Bindings) {
		return nil, ErrUnavailable
	}
	h := &http.Client{Timeout: 5 * time.Second, Transport: boundedTransport{http.DefaultTransport}, CheckRedirect: func(*http.Request, []*http.Request) error { return ErrUnavailable }}
	provider, e := oidc.NewProvider(oidc.ClientContext(ctx, h), cfg.Issuer)
	if e != nil {
		return nil, ErrUnavailable
	}
	var discovery struct {
		JWKS string `json:"jwks_uri"`
	}
	if provider.Claims(&discovery) != nil || !endpoint(discovery.JWKS) || !endpoint(provider.Endpoint().AuthURL) || !endpoint(provider.Endpoint().TokenURL) {
		return nil, ErrUnavailable
	}
	c := &Client{issuer: cfg.Issuer, bindings: cfg.Bindings, http: h, now: time.Now, pending: map[string]transaction{}, oauth: oauth2.Config{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, Endpoint: provider.Endpoint(), RedirectURL: origin + "/console/api/v1/oidc/callback", Scopes: []string{oidc.ScopeOpenID}}, verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID, SupportedSigningAlgs: []string{"RS256", "ES256", "PS256", "EdDSA"}})}
	if _, e = c.readBindings(); e != nil {
		return nil, e
	}
	return c, nil
}
func random() (string, error) {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", ErrUnavailable
	}
	return hex.EncodeToString(b[:]), nil
}
func (c *Client) Start() (string, string, string, error) {
	state, e := random()
	if e != nil {
		return "", "", "", e
	}
	nonce, e := random()
	if e != nil {
		return "", "", "", e
	}
	correlation, e := random()
	if e != nil {
		return "", "", "", e
	}
	verifier := oauth2.GenerateVerifier()
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for k, p := range c.pending {
		if !now.Before(p.expires) {
			delete(c.pending, k)
		}
	}
	if len(c.pending) >= 128 {
		return "", "", "", ErrUnavailable
	}
	c.pending[state] = transaction{correlation: correlation, nonce: nonce, verifier: verifier, expires: now.Add(5 * time.Minute)}
	return c.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), state, correlation, nil
}
func (c *Client) Complete(ctx context.Context, state, correlation, code string) (Identity, error) {
	c.mu.Lock()
	p, ok := c.pending[state]
	delete(c.pending, state)
	now := c.now()
	c.mu.Unlock()
	if !ok || !now.Before(p.expires) || len(code) < 1 || len(code) > 4096 || subtle.ConstantTimeCompare([]byte(p.correlation), []byte(correlation)) != 1 {
		return Identity{}, ErrLogin
	}
	ctx, cancel := context.WithTimeout(oidc.ClientContext(ctx, c.http), 5*time.Second)
	defer cancel()
	token, e := c.oauth.Exchange(ctx, code, oauth2.VerifierOption(p.verifier))
	if e != nil {
		return Identity{}, ErrLogin
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || len(raw) > 1<<20 {
		return Identity{}, ErrLogin
	}
	verified, e := c.verifier.Verify(ctx, raw)
	if e != nil || verified.Issuer != c.issuer || subtle.ConstantTimeCompare([]byte(verified.Nonce), []byte(p.nonce)) != 1 || !validSubject(verified.Subject) || !now.Before(verified.Expiry) {
		return Identity{}, ErrLogin
	}
	var claims struct {
		IssuedAt        int64  `json:"iat"`
		AuthorizedParty string `json:"azp"`
	}
	if verified.Claims(&claims) != nil || claims.IssuedAt <= 0 || claims.IssuedAt > now.Add(time.Minute).Unix() || (len(verified.Audience) > 1 && claims.AuthorizedParty != c.oauth.ClientID) || (claims.AuthorizedParty != "" && claims.AuthorizedParty != c.oauth.ClientID) {
		return Identity{}, ErrLogin
	}
	id, fp, e := c.Resolve(verified.Subject, "")
	if e != nil {
		return Identity{}, e
	}
	return Identity{Subject: verified.Subject, PrincipalID: id, Fingerprint: fp, ExpiresAt: verified.Expiry}, nil
}
