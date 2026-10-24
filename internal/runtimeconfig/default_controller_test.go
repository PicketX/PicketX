// SPDX-License-Identifier: Apache-2.0
package runtimeconfig

import "testing"

func TestDefaultControllerConfiguration(t *testing.T) {
	c, err := LoadDefault(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !c.InitialSetup || !c.Agent.Enabled || !c.Authz.Enabled || !c.Browser.Enabled || c.Source.Mode != "database" || c.Management.UsersMode != "database" {
		t.Fatal("incomplete default Controller")
	}
	if c.Database != "var/controller/controller.sqlite" || c.Browser.Origin != "" {
		t.Fatal("unexpected default paths/origin")
	}
	c, err = LoadDefault(map[string]string{"PICKETX_BROWSER_ORIGIN": "http://127.0.0.1:5173", "PICKETX_HTTP_LISTEN": "127.0.0.1:8081"}, map[string]string{"http.listen": "127.0.0.1:8082"})
	if err != nil || c.HTTPListen != "127.0.0.1:8082" || c.Browser.Origin != "http://127.0.0.1:5173" {
		t.Fatal("default loading changed precedence", err)
	}
	if _, err = LoadDefault(map[string]string{"PICKETX_DATABASE": ""}, nil); err == nil {
		t.Fatal("explicit empty value ignored")
	}
	c, err = Load("", nil, nil)
	if err != nil || c.InitialSetup || c.Agent.Enabled {
		t.Fatal("configured/file-mode defaults changed", err)
	}
}
