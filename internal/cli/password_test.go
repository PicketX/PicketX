// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"bytes"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"testing"
)

func TestPasswordHashStdinOnlyAndBounded(t *testing.T) {
	for _, ending := range []string{"", "\n", "\r\n"} {
		var out, diag bytes.Buffer
		cmd := New(&out, &diag)
		cmd.SetArgs([]string{"password-hash"})
		cmd.SetIn(strings.NewReader("a long test password!" + ending))
		if e := cmd.Execute(); e != nil {
			t.Fatal("hash command missing", e)
		}
		hash := strings.TrimSuffix(out.String(), "\n")
		if e := bcrypt.CompareHashAndPassword([]byte(hash), []byte("a long test password!")); e != nil {
			t.Fatal(e)
		}
		if cost, _ := bcrypt.Cost([]byte(hash)); cost != 12 {
			t.Fatal(cost)
		}
		if diag.Len() != 0 {
			t.Fatal("unexpected diagnostics", diag.String())
		}
	}
	for name, input := range map[string]string{"short": "short", "long": strings.Repeat("a", 73), "unicode": strings.Repeat("界", 25), "multiline": "first password\nsecond password", "empty": ""} {
		t.Run(name, func(t *testing.T) {
			var out, diag bytes.Buffer
			c := New(&out, &diag)
			c.SetArgs([]string{"password-hash"})
			c.SetIn(strings.NewReader(input))
			e := c.Execute()
			if e == nil || out.Len() != 0 || strings.Contains(e.Error(), input) && input != "" {
				t.Fatal("invalid secret accepted or echoed")
			}
		})
	}
	var out, diag bytes.Buffer
	c := New(&out, &diag)
	c.SetArgs([]string{"password-hash", "argument-secret"})
	c.SetIn(strings.NewReader("a long test password!"))
	if e := c.Execute(); e == nil || out.Len() != 0 {
		t.Fatal("password argument accepted")
	}
}
