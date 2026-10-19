// SPDX-License-Identifier: Apache-2.0
package runtimeconfig

import "testing"

func TestManagementUsersModeExplicitAndSourceBound(t *testing.T) {
	base := map[string]string{"agent.enabled": "true", "agent.authority": "controller:test", "database": "state/db.sqlite", "credentials": "agent.json", "source.mode": "database", "management.credentials": "keys.json", "management.users_mode": "database"}
	c, err := Load("", nil, base)
	if err != nil {
		t.Fatal(err)
	}
	if c.Management.UsersMode != "database" {
		t.Fatal(c.Management)
	}
	base["management.users_mode"] = "merge"
	if _, err = Load("", nil, base); err == nil {
		t.Fatal("unknown user authority")
	}
	if _, err = Load("", nil, map[string]string{"management.users_mode": "database"}); err == nil {
		t.Fatal("DB users without managed source")
	}
	c, err = Load("", nil, nil)
	if err != nil || c.Management.UsersMode != "file" {
		t.Fatal(c, err)
	}
}
