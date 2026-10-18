// SPDX-License-Identifier: Apache-2.0
package runtimeconfig

import "testing"

func TestAuthorizationListenerExplicitAndLocal(t *testing.T) {
	base := map[string]string{"authz.enabled": "true", "authz.network": "tcp", "authz.address": "[::1]:0", "authz.credentials": "callers.json", "agent.enabled": "true", "agent.authority": "test", "database": "db", "credentials": "agents.json", "source.policy": "p", "source.facts": "f"}
	c, e := Load("", nil, base)
	if e != nil || !c.Authz.Enabled {
		t.Fatal(c, e)
	}
	for _, mut := range []map[string]string{{"authz.address": "0.0.0.0:9091"}, {"authz.address": "localhost:9091"}, {"authz.credentials": ""}, {"authz.enabled": "false"}, {"agent.enabled": "false"}, {"authz.network": "udp"}, {"authz.network": "unix", "authz.address": "relative.sock"}} {
		v := map[string]string{}
		for k, x := range base {
			v[k] = x
		}
		for k, x := range mut {
			v[k] = x
		}
		if _, e = Load("", nil, v); e == nil {
			t.Fatal("invalid listener accepted", mut)
		}
	}
}
