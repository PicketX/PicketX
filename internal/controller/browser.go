// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// Browser owns one process-lifetime in-memory SCS manager. Restart intentionally
// invalidates sessions; it is not a multi-replica identity provider.
type Browser struct {
	service  *management.Service
	sessions *scs.SessionManager
	origin   *url.URL
	assets   string
	// Only existing authenticated sessions acquire an exact-token lock. Entries
	// exist only while held/waiting; unknown cookies cannot occupy shared locks.
	lockMu sync.Mutex
	locks  map[[sha256.Size]byte]*browserSessionLock
}

type browserSessionLock struct {
	gate chan struct{}
	refs int
}

func (b *Browser) lockSession(ctx context.Context, key [sha256.Size]byte) (func(), error) {
	b.lockMu.Lock()
	if b.locks == nil {
		b.locks = make(map[[sha256.Size]byte]*browserSessionLock)
	}
	entry := b.locks[key]
	if entry == nil {
		entry = &browserSessionLock{gate: make(chan struct{}, 1)}
		b.locks[key] = entry
	}
	entry.refs++
	b.lockMu.Unlock()
	drop := func() {
		b.lockMu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(b.locks, key)
		}
		b.lockMu.Unlock()
	}
	select {
	case entry.gate <- struct{}{}:
		return func() { <-entry.gate; drop() }, nil
	case <-ctx.Done():
		drop()
		return nil, ctx.Err()
	}
}

func NewBrowser(service *management.Service, origin, assets string) (*Browser, error) {
	if service == nil {
		return nil, fmt.Errorf("browser requires management service")
	}
	if e := management.ValidateBrowserOrigin(origin); e != nil {
		return nil, e
	}
	if assets != "" {
		r, e := os.OpenRoot(assets)
		if e != nil {
			return nil, fmt.Errorf("open browser assets: %w", e)
		}
		f, e := r.Open("index.html")
		if e == nil {
			info, x := f.Stat()
			if x != nil || !info.Mode().IsRegular() {
				e = fmt.Errorf("not regular")
			}
			f.Close()
		}
		r.Close()
		if e != nil {
			return nil, fmt.Errorf("browser index is required")
		}
	}
	s := scs.New()
	s.Lifetime = 8 * time.Hour
	s.IdleTimeout = 30 * time.Minute
	s.Cookie.Name = "picketx_session"
	s.Cookie.Path = "/console/"
	s.Cookie.SameSite = http.SameSiteStrictMode
	s.Cookie.Persist = false
	s.HashTokenInStore = true
	u, _ := url.Parse(origin)
	s.Cookie.Secure = u.Scheme == "https"
	s.ErrorFunc = func(w http.ResponseWriter, _ *http.Request, _ error) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		_, _ = io.WriteString(w, `{"code":"session_unavailable"}`)
	}
	return &Browser{service: service, sessions: s, origin: u, assets: assets}, nil
}
func (b *Browser) wrap(next http.Handler) http.Handler {
	session := b.sessions.LoadAndSave(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/console" && !strings.HasPrefix(r.URL.Path, "/console/") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if r.Host != b.origin.Host {
			http.Error(w, "untrusted host", 403)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && (len(r.Header.Values("Origin")) != 1 || r.Header.Get("Origin") != b.origin.String()) {
			http.Error(w, "untrusted origin", 403)
			return
		}
		if len(r.Header.Values("Authorization")) != 0 {
			http.Error(w, "ambiguous authentication", 400)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/console/api/") {
			if cookie, err := r.Cookie(b.sessions.Cookie.Name); err == nil && cookie.Value != "" {
				// Precheck only: never attach this loaded context to the request.
				// LoadAndSave must load again under the lock after a queued logout.
				ctx, err := b.sessions.Load(r.Context(), cookie.Value)
				if err != nil {
					b.sessions.ErrorFunc(w, r, err)
					return
				}
				if b.sessions.GetString(ctx, "id") != "" {
					release, err := b.lockSession(r.Context(), sha256.Sum256([]byte(cookie.Value)))
					if err != nil {
						http.Error(w, "request canceled", http.StatusRequestTimeout)
						return
					}
					defer release()
				}
			}
			session.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (b *Browser) principal(c *echo.Context) (management.Principal, error) {
	ctx := c.Request().Context()
	p, e := b.service.ResolveSession(b.sessions.GetString(ctx, "id"), b.sessions.GetString(ctx, "credential"))
	if errors.Is(e, management.ErrUnauthorized) {
		_ = b.sessions.Destroy(ctx)
	}
	return p, e
}
func (b *Browser) register(e *echo.Echo, limits ManagementLimits) {
	h := &managementHTTP{service: b.service, validate: validator.New(), resolve: b.principal}
	g := e.Group("/console/api/v1")
	g.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: float64(limits.RequestsPerSecond), Burst: limits.Burst}),
		IdentifierExtractor: func(c *echo.Context) (string, error) {
			host, _, e := net.SplitHostPort(c.Request().RemoteAddr)
			return host, e
		},
		ErrorHandler: func(c *echo.Context, _ error) error { return apiError(c, 400, "invalid_peer") }, DenyHandler: func(c *echo.Context, _ string, _ error) error { return apiError(c, 429, "rate_limited") },
	}))
	g.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{CookieName: "picketx_csrf", CookiePath: "/console/", CookieSecure: b.origin.Scheme == "https", CookieHTTPOnly: true, CookieSameSite: http.SameSiteStrictMode, ErrorHandler: func(c *echo.Context, _ error) error { return apiError(c, 403, "csrf_rejected") }}))
	g.GET("/session", b.session)
	g.POST("/login", func(c *echo.Context) error { return b.login(c, h.validate) })
	g.POST("/logout", b.logout)
	g.GET("/me", h.browserMe)
	g.GET("/publication", h.browserPublication)
	g.PUT("/publication", h.browserReplace)
	g.GET("/audit", h.browserAudit)
	if b.assets != "" {
		e.GET("/console", func(c *echo.Context) error { return c.Redirect(308, "/console/") })
		e.GET("/console/*", b.asset)
		e.HEAD("/console/*", b.asset)
	}
}
func (b *Browser) asset(c *echo.Context) error {
	name := strings.TrimPrefix(c.Request().URL.Path, "/console/")
	if name == "" {
		name = "index.html"
	}
	if strings.HasPrefix(name, "api/") || path.Clean(name) != name || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
		return c.NoContent(404)
	}
	ext := path.Ext(name)
	switch ext {
	case ".html", ".js", ".css", ".svg", ".png", ".ico", ".woff2":
	default:
		return c.NoContent(404)
	}
	root, err := os.OpenRoot(b.assets)
	if err != nil {
		return c.NoContent(503)
	}
	defer root.Close()
	f, err := root.Open(name)
	if err != nil {
		return c.NoContent(404)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return c.NoContent(404)
	}
	http.ServeContent(c.Response(), c.Request(), name, info.ModTime(), f)
	return nil
}
