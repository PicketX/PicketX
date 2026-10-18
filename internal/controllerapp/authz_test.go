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
	for _, network := range []string{"tcp", "unix"} {
		t.Run(network, func(t *testing.T) {
			c := fixture(t)
			c.Authz.Enabled = true
			c.Authz.Network = network
			c.Authz.Address = "127.0.0.1:0"
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
				info, e := os.Stat(addr)
				if e != nil || info.Mode().Perm() != 0600 {
					t.Fatal(info, e)
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
