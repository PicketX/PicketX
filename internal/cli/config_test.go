// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const cliPolicy = `apiVersion: picketx.io/v1alpha1
kind: Resource
metadata: {name: gitea, uid: r1}
spec:
 endpoints:
 - name: https
   network: {protocol: TCP, port: 443}
---
apiVersion: picketx.io/v1alpha1
kind: Grant
metadata: {name: temporary, uid: g1}
spec:
 subject: {type: source, cidr: 198.51.100.9}
 scopes: [{resourceRef: {name: gitea, uid: r1}}]
 validity: {mode: bounded, expiresAt: '2026-10-15T10:00:00Z'}
---
apiVersion: picketx.io/v1alpha1
kind: EnforcementBinding
metadata: {name: host, uid: b1}
spec:
 resources: [{resourceRef: {name: gitea, uid: r1}}]
 agents: [{agentRef: {name: a}}]
 enforcement: {profileRef: host, class: host-local, stage: original-destination, allowedDestinations: [192.0.2.100], handling: graceful}
`
const cliFacts = `{"authorityID":"static:example","agents":[{"Name":"a","Capabilities":{"Classes":["host-local"],"Protocols":["tcp"],"Families":[4],"Handling":["graceful"],"OriginalDestination":true}}],"contexts":[{"AgentRef":"a","ProfileRef":"host","ID":"host-v1","NetworkSpaceID":"test-only-operator-facts","Class":"host-local","Stage":"original-destination","Destinations":["192.0.2.100/32"],"Handling":"graceful"}]}`

func files(t *testing.T, p, f string) (string, string) {
	t.Helper()
	d := t.TempDir()
	pp, ff := filepath.Join(d, "policy.yaml"), filepath.Join(d, "facts.json")
	if e := os.WriteFile(pp, []byte(p), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(ff, []byte(f), 0600); e != nil {
		t.Fatal(e)
	}
	return pp, ff
}
func execute(args ...string) (string, error) {
	var out, diag bytes.Buffer
	c := New(&out, &diag)
	c.SetArgs(args)
	e := c.Execute()
	return out.String(), e
}
func TestCheckProducesOnlyValidatedDesiredIntent(t *testing.T) {
	p, f := files(t, cliPolicy, cliFacts)
	out, e := execute("config", "check", "--file", p, "--facts", f, "--at", "2026-10-15T09:00:00Z")
	if e != nil {
		t.Fatal(e)
	}
	var result struct {
		State       string `json:"state"`
		Confirmed   bool   `json:"enforcementConfirmed"`
		Hash        string `json:"policyHash"`
		Objects     []any  `json:"objects"`
		Projections []struct {
			Allow     []any
			Protected []any
		} `json:"projections"`
	}
	if e = json.Unmarshal([]byte(out), &result); e != nil {
		t.Fatal(e)
	}
	if result.State != "validated" || result.Confirmed || len(result.Hash) != 64 || len(result.Objects) != 3 || len(result.Projections) != 1 || len(result.Projections[0].Allow) != 1 {
		t.Fatalf("misleading result: %s", out)
	}
	out, e = execute("config", "check", "--file", p, "--facts", f, "--at", "2026-10-15T10:00:00Z")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal([]byte(out), &result); e != nil {
		t.Fatal(e)
	}
	if len(result.Projections[0].Allow) != 0 || len(result.Projections[0].Protected) != 1 {
		t.Fatalf("deadline renewed or protection disappeared: %s", out)
	}
}
func TestCheckErrorsNeverPrintSuccess(t *testing.T) {
	for name, tc := range map[string]struct{ policy, facts, at string }{
		"invalid policy":       {strings.Replace(cliPolicy, "endpoint", "misspelled", 1), cliFacts, "2026-10-15T09:00:00Z"},
		"invalid timestamp":    {cliPolicy, cliFacts, "tomorrow"},
		"unknown facts":        {cliPolicy, strings.Replace(cliFacts, "\"authorityID\":", "\"unknown\":", 1), ""},
		"unknown nested facts": {cliPolicy, strings.Replace(cliFacts, "\"Protocols\":", "\"Protcols\":", 1), ""},
		"trailing facts":       {cliPolicy, cliFacts + " {}", ""},
		"duplicate facts":      {cliPolicy, strings.Replace(cliFacts, "\"authorityID\":", "\"authorityID\":\"other\",\"authorityID\":", 1), ""},
		"null facts":           {cliPolicy, strings.Replace(cliFacts, "[4]", "null", 1), ""},
		"oversize facts":       {cliPolicy, strings.Repeat(" ", 4<<20+1), ""},
		"oversize policy":      {strings.Repeat(" ", 4<<20+1), cliFacts, ""},
		"deep facts":           {cliPolicy, strings.Repeat("[", 66) + "1" + strings.Repeat("]", 66), ""},
	} {
		t.Run(name, func(t *testing.T) {
			p, f := files(t, tc.policy, tc.facts)
			args := []string{"config", "check", "--file", p, "--facts", f}
			if tc.at != "" {
				args = append(args, "--at", tc.at)
			}
			out, e := execute(args...)
			if e == nil || out != "" {
				t.Fatalf("failure emitted success %q: %v", out, e)
			}
		})
	}
	for _, args := range [][]string{{"config", "check"}, {"config", "check", "--file", "/missing", "--facts", "/missing"}, {"config", "check", "--other"}} {
		out, e := execute(args...)
		if e == nil || out != "" {
			t.Fatalf("expected error without stdout: %v %q", e, out)
		}
	}
}

func TestCheckRejectsCaseAliasedDuplicateFacts(t *testing.T) {
	p, f := files(t, cliPolicy, strings.Replace(cliFacts, `"Name":"a"`, `"Name":"a","name":"a"`, 1))
	out, e := execute("config", "check", "--file", p, "--facts", f)
	if e == nil || out != "" {
		t.Fatal("case-aliased facts keys silently overwrote each other")
	}
}

func TestCheckPreservesCaseSensitiveLabelKeys(t *testing.T) {
	facts := strings.Replace(cliFacts, `"Name":"a"`, `"Name":"a","Labels":{"env":"dev","ENV":"other"}`, 1)
	p, f := files(t, cliPolicy, facts)
	if out, e := execute("config", "check", "--file", p, "--facts", f); e != nil || out == "" {
		t.Fatalf("valid distinct labels rejected: %v", e)
	}
}

func TestCheckEmptyWithdrawalReportsKnownAgents(t *testing.T) {
	p, f := files(t, `apiVersion: picketx.io/v1alpha1
kind: Configuration
metadata: {name: empty, uid: empty-v1}
spec: {empty: true}
`, cliFacts)
	out, e := execute("config", "check", "--file", p, "--facts", f, "--at", "2026-10-15T09:00:00Z")
	if e != nil {
		t.Fatal(e)
	}
	var report struct {
		Projections []struct {
			AgentRef  string
			Protected []any
			Allow     []any
		} `json:"projections"`
	}
	if e = json.Unmarshal([]byte(out), &report); e != nil {
		t.Fatal(e)
	}
	if len(report.Projections) != 1 || report.Projections[0].AgentRef != "a" || len(report.Projections[0].Protected) != 0 || len(report.Projections[0].Allow) != 0 {
		t.Fatalf("missing explicit empty known target: %s", out)
	}
}

func TestCheckRejectsUnicodeFoldFieldAliases(t *testing.T) {
	f := strings.Replace(cliFacts, `"Protocols":[`, `"Protocols":["unsupported"],"Protocolſ":[`, 1)
	p, path := files(t, cliPolicy, f)
	if out, e := execute("config", "check", "--file", p, "--facts", path); e == nil || out != "" {
		t.Fatal("Unicode field alias overwrote an invalid safety constraint")
	}
}
