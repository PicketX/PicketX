// SPDX-License-Identifier: Apache-2.0
package runtimeconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func file(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if e := os.WriteFile(p, []byte(s), 0600); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestDefaultsAndPrecedence(t *testing.T) {
	c, e := Load("", nil, nil)
	if e != nil || c.HTTPListen != "127.0.0.1:8080" || c.Agent.Enabled || c.Poll != time.Second {
		t.Fatal(c, e)
	}
	c, e = Load(file(t, "http:\n  listen: 127.0.0.1:8001\npoll: 2s\n"), map[string]string{"PICKETX_HTTP_LISTEN": "127.0.0.1:8002"}, map[string]string{"http.listen": "127.0.0.1:8003"})
	if e != nil || c.HTTPListen != "127.0.0.1:8003" || c.Poll != 2*time.Second {
		t.Fatal(c, e)
	}
	if _, e = Load("", map[string]string{"PICKETX_HTTP_LISTEN": ""}, nil); e == nil {
		t.Fatal("present empty env ignored")
	}
}

func TestBrowserRequiresManagedAuthorityAndExplicitSecureOrigin(t *testing.T) {
	base := map[string]string{"agent.enabled": "true", "agent.authority": "controller:test", "database": "state/db.sqlite", "credentials": "agent.json", "source.mode": "database", "management.credentials": "principals.json", "browser.enabled": "true", "browser.origin": "https://control.example"}
	c, e := Load("", nil, base)
	if e != nil {
		t.Fatal("valid browser config", e)
	}
	if !c.Browser.Enabled || c.Browser.Origin != "https://control.example" {
		t.Fatal(c)
	}
	for _, origin := range []string{"", "http://control.example", "https://control.example/", "https://user@control.example", "https://control.example?x=1", "https://control.example#x", "https://control.example/a"} {
		base["browser.origin"] = origin
		if _, e = Load("", nil, base); e == nil {
			t.Fatal("invalid browser origin accepted", origin)
		}
	}
	for _, origin := range []string{"http://127.0.0.1:8080", "http://[::1]:8080", "http://localhost:5173"} {
		base["browser.origin"] = origin
		if _, e = Load("", nil, base); e != nil {
			t.Fatal(origin, e)
		}
	}
	if _, e = Load("", nil, map[string]string{"browser.enabled": "true", "browser.origin": "https://control.example"}); e == nil {
		t.Fatal("file-mode browser accepted")
	}
	if _, e = Load("", nil, map[string]string{"browser.assets": "dist"}); e == nil {
		t.Fatal("ignored disabled browser setting")
	}
}
func TestRejectInvalidConfig(t *testing.T) {
	for _, s := range []string{"unknown: true", "agent: {enabled: null}", "poll: 0s", "agent: {enabled: true}", "http: {listen: true}", "http: {listen: '127.0.0.1:1', listen: '127.0.0.1:2'}", "---\nhttp: {listen: '127.0.0.1:1'}\n---\npoll: 1s", "agent: {unknown: {}}", "agent: {max_frame: 4194305}", "http: {listen: broken}", "agent: {tls_cert: cert}", "agent: &x {enabled: false}\nsource: *x"} {
		t.Run(s, func(t *testing.T) {
			if _, e := Load(file(t, s), nil, nil); e == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
	if _, e := Load("", map[string]string{"PICKETX_UNKNOWN": "true"}, nil); e == nil {
		t.Fatal("unknown env accepted")
	}
}
func TestEnabledConfig(t *testing.T) {
	c, e := Load(file(t, "agent: {enabled: true, authority: 'controller:test'}\nsource: {policy: policy.yaml, facts: facts.json}\ndatabase: state/picketx.sqlite\ncredentials: credentials.json\n"), nil, nil)
	if e != nil || !c.Agent.Enabled || c.Agent.Authority != "controller:test" || c.Agent.Limits.MaxCredentials != 128 {
		t.Fatal(c, e)
	}
}

func TestDocumentedControllerExample(t *testing.T) {
	root := filepath.Join("..", "..")
	c, e := Load(filepath.Join(root, "examples", "controller", "controller.yaml"), nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	if !c.Agent.Enabled || c.Agent.Authority != "static:example-v1" {
		t.Fatal(c)
	}
	for _, source := range []string{c.Source.Policy, c.Source.Facts} {
		info, e := os.Stat(filepath.Join(root, source))
		if e != nil || !info.Mode().IsRegular() {
			t.Fatal("documented source missing", source, e)
		}
	}
}
