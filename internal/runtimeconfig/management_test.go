// SPDX-License-Identifier: Apache-2.0
package runtimeconfig

import "testing"

func TestDatabaseAuthorityIsExplicitAndExclusive(t *testing.T) {
	base := map[string]string{"agent.enabled": "true", "agent.authority": "authority:test", "database": "state/db.sqlite", "credentials": "agents.json", "source.mode": "database", "management.credentials": "principals.json"}
	c, e := Load("", nil, base)
	if e != nil {
		t.Fatal(e)
	}
	if c.Source.Mode != "database" || c.Management.Credentials != "principals.json" || c.Management.Reload <= 0 {
		t.Fatal(c)
	}
	for _, mutation := range []map[string]string{{"source.mode": "both"}, {"source.policy": "policy.yaml"}, {"source.facts": "facts.json"}, {"management.credentials": ""}, {"agent.enabled": "false"}, {"management.reload": "0s"}, {"management.requests_per_second": "0"}} {
		values := map[string]string{}
		for k, v := range base {
			values[k] = v
		}
		for k, v := range mutation {
			values[k] = v
		}
		if _, e := Load("", nil, values); e == nil {
			t.Fatal("invalid mode settings accepted", mutation)
		}
	}
	if _, e = Load("", nil, map[string]string{"management.credentials": "principals.json"}); e == nil {
		t.Fatal("management enabled implicitly in file mode")
	}
}
