// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuthorizationRuntimeTCPUnixAndShutdown(t *testing.T) {
	for _, network := range []string{"tcp", "tcp6", "unix"} {
		t.Run(network, func(t *testing.T) {
			c := fixture(t)
			original, e := os.ReadFile(c.Source.Policy)
			if e != nil {
				t.Fatal(e)
			}
			// Existing static examples intentionally have historical deadlines. This
			// runtime acceptance fixture gives both independent Grants a future bound.
			policy := strings.ReplaceAll(string(original), "2026-10-15T10:00:00Z", time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano))
			policy = strings.ReplaceAll(policy, "2026-10-15T12:00:00Z", time.Now().UTC().Add(2*time.Hour).Format(time.RFC3339Nano))
			policy = strings.ReplaceAll(policy, "endpointNames: [ssh]", "endpointNames: [https]")
			write(t, c.Source.Policy, append([]byte(policy), []byte(`
---
apiVersion: picketx.io/v1alpha1
kind: HTTPRoute
metadata: {name: gitea-web, uid: route-v1}
spec: {resourceRef: {name: gitea, uid: resource-gitea-v1}, endpointName: https, host: app.example, methods: [GET], paths: [/allowed]}
`)...))
			c.Authz.Enabled = true
			c.Authz.Network = network
			if network == "tcp6" {
				c.Authz.Network = "tcp"
			}
			c.Authz.Address = "127.0.0.1:0"
			if network == "tcp6" {
				c.Authz.Address = "[::1]:0"
			}
			if network == "unix" {
				dir := filepath.Join(t.TempDir(), "private")
				if e := os.Mkdir(dir, 0700); e != nil {
					t.Fatal(e)
				}
				c.Authz.Address = filepath.Join(dir, "auth.sock")
			}
			c.Authz.Credentials = filepath.Join(t.TempDir(), "callers.json")
			sum := sha256.Sum256([]byte(adminToken))
			data, _ := json.Marshal(map[string]any{"callers": []map[string]any{{"id": "nginx", "sha256": hex.EncodeToString(sum[:]), "routes": []string{"route-v1"}}}})
			write(t, c.Authz.Credentials, data)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			addrs := make(chan string, 3)
			go func() {
				done <- Run(ctx, c, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{Listen: func(n, a string) (net.Listener, error) {
					l, e := net.Listen(n, a)
					if e == nil {
						addrs <- l.Addr().String()
					}
					return l, e
				}})
			}()
			var addr string
			for i := 0; i < 3; i++ {
				select {
				case addr = <-addrs:
				case e := <-done:
					t.Fatal("startup", e)
				case <-time.After(3 * time.Second):
					t.Fatal("missing authz listener")
				}
			}
			tr := &http.Transport{Proxy: nil}
			url := "http://" + addr + "/authz/v1/decisions"
			if network == "unix" {
				url = "http://localhost/authz/v1/decisions"
				tr.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", addr)
				}
			}
			client := &http.Client{Transport: tr, Timeout: time.Second}
			defer tr.CloseIdleConnections()
			r, e := client.Post(url, "application/json", strings.NewReader(`{}`))
			if e != nil {
				t.Fatal(e)
			}
			r.Body.Close()
			if r.StatusCode != 401 {
				t.Fatal(r.StatusCode)
			}
			if network == "unix" {
				// Address discovery happens inside Listen, before runtime chmod.
				// A real response proves Serve has started after socket setup.
				info, e := os.Stat(addr)
				if e != nil || info.Mode().Perm() != 0600 {
					t.Fatal(info, e)
				}
			}
			source := "198.51.100.9"
			if network == "tcp6" {
				source = "2001:db8:1::9"
			}
			req, e := http.NewRequest("POST", url, strings.NewReader(`{"route":"route-v1","source":"`+source+`","host":"app.example","method":"GET","uri":"/allowed"}`))
			if e != nil {
				t.Fatal(e)
			}
			req.Header.Set("Authorization", "Bearer "+adminToken)
			req.Header.Set("Content-Type", "application/json")
			r, e = client.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			raw, e := io.ReadAll(r.Body)
			r.Body.Close()
			if e != nil || r.StatusCode != 200 || !strings.Contains(string(raw), `"allowed":true`) {
				t.Fatal("real scoped decision", r.StatusCode, string(raw), e)
			}
			r, e = client.Get(strings.TrimSuffix(url, "/authz/v1/decisions") + "/api/v1/publication")
			if e != nil {
				t.Fatal(e)
			}
			r.Body.Close()
			if r.StatusCode != 404 {
				t.Fatal("admin API on authorization listener", r.StatusCode)
			}
			cancel()
			select {
			case e := <-done:
				if e != nil {
					t.Fatal(e)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("shutdown stalled")
			}
			if network == "unix" {
				if _, e = os.Lstat(addr); !os.IsNotExist(e) {
					t.Fatal("owned socket not cleaned", e)
				}
			}
		})
	}
}
func TestAuthorizationUnixNeverOverwritesExistingObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing")
	write(t, path, []byte("user data"))
	if l, e := listenAuthorization("unix", path, net.Listen); e == nil {
		l.Close()
		t.Fatal("overwrote object")
	}
	data, e := os.ReadFile(path)
	if e != nil || string(data) != "user data" {
		t.Fatal(string(data), e)
	}
}
func TestHTTPDecisionAuditUsesManagementRBAC(t *testing.T) {
	c := managedConfig(t)
	p := start(t, c)
	ready(t, p, 200)
	code, _, body := apiCall(t, p, "GET", "/api/v1/http-decisions?limit=1", nil, "", adminToken)
	if code != 200 || !strings.Contains(string(body), `"events":[]`) {
		t.Fatal(code, string(body))
	}
	sum := sha256.Sum256([]byte(adminToken))
	data, _ := json.Marshal(map[string]any{"principals": []map[string]any{{"id": "principal:admin", "sha256": hex.EncodeToString(sum[:]), "roles": []string{"role:blocked"}}}})
	write(t, c.Management.Credentials, data)
	code, _, body = apiCall(t, p, "GET", "/api/v1/http-decisions", nil, "", adminToken)
	if code != 403 {
		t.Fatal("audit bypassed current RBAC", code, string(body))
	}
}
