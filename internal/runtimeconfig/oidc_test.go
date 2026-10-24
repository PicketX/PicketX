// SPDX-License-Identifier: Apache-2.0
package runtimeconfig

import "testing"

func TestOIDCRequiresExplicitBrowserAndPrivateConfigPath(t *testing.T) {
	base := map[string]string{"agent.enabled": "true", "agent.authority": "authority:test", "database": "db.sqlite", "credentials": "agents.json", "source.mode": "database", "management.credentials": "keys.json", "browser.enabled": "true", "browser.origin": "http://127.0.0.1:8080", "browser.oidc_config": "/private/oidc.json"}
	c, e := Load("", nil, base)
	if e != nil {
		t.Fatalf("OIDC config not supported: %v", e)
	}
	if c.Browser.OIDCConfig != "/private/oidc.json" {
		t.Fatal("OIDC config lost")
	}
	for _, m := range []map[string]string{{"browser.enabled": "false", "browser.origin": ""}, {"browser.oidc_config": "relative.json"}, {"browser.origin": ""}} {
		values := map[string]string{}
		for k, v := range base {
			values[k] = v
		}
		for k, v := range m {
			values[k] = v
		}
		if _, e := Load("", nil, values); e == nil {
			t.Fatal("invalid OIDC settings accepted", m)
		}
	}
}
