// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/pkg/core"
	"github.com/PicketX/PicketX/pkg/networkplan"
)

func facts() Facts {
	return Facts{AuthorityID: "static:example-v1", Agents: []networkplan.Agent{{Name: "agent-a", Labels: map[string]string{"env": "dev"}, Capabilities: &networkplan.CapabilityProfile{Classes: []networkplan.TrafficClass{networkplan.HostLocal}, Protocols: []networkplan.Protocol{networkplan.TCP, networkplan.UDP}, Families: []networkplan.Family{networkplan.IPv4, networkplan.IPv6}, Handling: []networkplan.HandlingMode{networkplan.Graceful, networkplan.Strict}, OriginalDestination: true}}}, Contexts: []networkplan.ExecutionContext{{AgentRef: "agent-a", ProfileRef: "host", ID: "host-v1", NetworkSpaceID: "live-runtime-owned-space", Class: networkplan.HostLocal, Stage: networkplan.OriginalDestination, Destinations: []netip.Prefix{netip.MustParsePrefix("192.0.2.100/32"), netip.MustParsePrefix("2001:db8::100/128")}, Handling: networkplan.Graceful}}}
}
func instant(s string) time.Time {
	return time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC).Add(mustDuration(s))
}
func mustDuration(s string) time.Duration {
	d, e := time.ParseDuration(s)
	if e != nil {
		panic(e)
	}
	return d
}
func candidate(t *testing.T, s string) *Candidate {
	t.Helper()
	b, e := Decode([]byte(s))
	if e != nil {
		t.Fatal(e)
	}
	c, e := Compile(b, facts())
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestCompileIndependentDualStackDeadlinesAndDeny(t *testing.T) {
	ssh := strings.ReplaceAll(grantYAML, "browser", "ssh6")
	ssh = strings.Replace(ssh, "198.51.100.9", "2001:db8:1::9", 1)
	ssh = strings.Replace(ssh, "[https]", "[ssh]", 1)
	ssh = strings.Replace(ssh, "10:00:00", "12:00:00", 1)
	deny := strings.ReplaceAll(grantYAML, "browser", "block")
	deny = strings.Replace(deny, "kind: Grant", "kind: NetworkDeny", 1)
	deny = strings.Replace(deny, "mode: bounded, expiresAt:", "mode: bounded, notBefore: '2026-10-15T09:30:00Z', expiresAt:", 1)
	c := candidate(t, policy()+"---\n"+ssh+"---\n"+deny)
	p, e := c.Plan().Projection("agent-a", instant("9h"))
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Protected) != 2 || len(p.Allow) != 2 || len(p.Deny) != 0 || !p.NextChange.Equal(instant("9h30m")) {
		t.Fatalf("initial projection %#v", p)
	}
	atDeny, _ := c.Plan().Projection("agent-a", instant("9h30m"))
	if len(atDeny.Deny) != 1 {
		t.Fatal("future deny not activated")
	}
	p, _ = c.Plan().Projection("agent-a", instant("10h"))
	if len(p.Allow) != 1 || p.Allow[0].Target.Entry.EndpointName != "ssh" || p.Allow[0].Contribution.Source.Addr().Is4() {
		t.Fatalf("independent expiry/family lost: %#v", p)
	}
	engine, e := core.Compile(c.Input().Logical)
	if e != nil {
		t.Fatal(e)
	}
	d, e := engine.Decide(core.Query{Source: netip.MustParseAddr("198.51.100.9"), ResourceName: "resource-gitea-v1", EndpointName: "https", At: instant("9h30m")})
	if e != nil || d.Allowed || d.Reason != core.ReasonExplicitDeny {
		t.Fatalf("deny precedence %#v %v", d, e)
	}
	p, _ = c.Plan().Projection("agent-a", instant("12h"))
	if len(p.Allow) != 0 || len(p.Protected) != 2 {
		t.Fatal("expiry removed protection or renewed access")
	}
}
func TestCompileRejectsCompleteInvalidCandidate(t *testing.T) {
	tests := map[string]string{
		"empty actions":                   strings.Replace(policy(), "endpointNames: [https]", "endpointNames: [https]\n      actions: []", 1),
		"empty endpoint names":            strings.Replace(policy(), "[https]", "[]", 1),
		"empty endpoint selector":         strings.Replace(policy(), "endpointNames: [https]", "endpointSelector: {}", 1),
		"both endpoint restrictions":      strings.Replace(policy(), "endpointNames: [https]", "endpointNames: [https]\n      endpointSelector: {matchLabels: {access: dev}}", 1),
		"selector existence empty values": strings.Replace(policy(), "endpointNames: [https]", "endpointSelector: {matchExpressions: [{key: access, operator: Exists, values: []}]}", 1),
		"missing resource":                strings.Replace(policy(), "resourceRef: {name: gitea, uid: resource-gitea-v1}", "resourceRef: {name: missing, uid: resource-gitea-v1}", 1),
		"recreated resource":              strings.Replace(policy(), "uid: resource-gitea-v1}", "uid: resource-gitea-v2}", 1),
		"missing pinned uid":              strings.Replace(policy(), "resourceRef: {name: gitea, uid: resource-gitea-v1}", "resourceRef: {name: gitea}", 1),
		"unknown endpoint":                strings.Replace(policy(), "[https]", "[absent]", 1),
		"empty destinations":              strings.Replace(policy(), "network: {protocol: TCP, port: 443}", "network: {protocol: TCP, port: 443, addresses: []}", 1),
		"bad source":                      strings.Replace(policy(), "198.51.100.9", "example.com", 1),
		"bad deadline":                    strings.Replace(policy(), "2026-10-15T10:00:00Z", "two hours", 1),
		"empty notBefore":                 strings.Replace(policy(), "mode: bounded, expiresAt:", "mode: bounded, notBefore: '', expiresAt:", 1),
		"implicit permanent":              strings.Replace(policy(), "mode: bounded, expiresAt: '2026-10-15T10:00:00Z'", "mode: bounded", 1),
		"permanent with deadline":         strings.Replace(policy(), "mode: bounded", "mode: permanent", 1),
		"empty endpoints":                 strings.Replace(resourceYAML, "  endpoints:\n    - name: https\n      network: {protocol: TCP, port: 443}\n    - name: ssh\n      network: {protocol: TCP, port: 22}", "  endpoints: []", 1),
		"unknown Agent":                   strings.Replace(policy(), "name: agent-a", "name: unknown", 1),
		"empty Agent":                     strings.Replace(policy(), "agentRef: {name: agent-a}", "agentRef: {}", 1),
		"empty Agent selector":            strings.Replace(policy(), "agentRef: {name: agent-a}", "agentSelector: {}", 1),
		"both Agent selections":           strings.Replace(policy(), "agentRef: {name: agent-a}", "agentRef: {name: agent-a}\n      agentSelector: {matchLabels: {env: dev}}", 1),
	}
	for n, s := range tests {
		t.Run(n, func(t *testing.T) {
			b, e := Decode([]byte(s))
			if e != nil {
				return
			}
			c, e := Compile(b, facts())
			if e == nil || c != nil {
				t.Fatalf("accepted %s", n)
			}
		})
	}
}
func TestReplacePinnedIdentityRetargetingAndEmptyWithdrawal(t *testing.T) {
	old := candidate(t, policy())
	for name, s := range map[string]string{
		"retarget":       strings.Replace(policy(), "port: 443", "port: 8443", 1),
		"uid kind reuse": resourceYAML + "---\n" + strings.Replace(grantYAML, "kind: Grant", "kind: NetworkDeny", 1) + "---\n" + bindingYAML,
	} {
		t.Run(name, func(t *testing.T) {
			b, e := Decode([]byte(s))
			if e != nil {
				t.Fatal(e)
			}
			c, w, e := Replace(old, b, facts())
			if e == nil || c != nil || w != nil {
				t.Fatal("invalid replacement produced state/actions")
			}
		})
	}
	badFacts := facts()
	badFacts.AuthorityID = "other"
	b, _ := Decode([]byte(policy()))
	if _, _, e := Replace(old, b, badFacts); e == nil {
		t.Fatal("mixed authorities")
	}
	empty, _ := Decode([]byte(emptyYAML))
	next, w, e := Replace(old, empty, facts())
	if e != nil {
		t.Fatal(e)
	}
	p, _ := next.Plan().Projection("agent-a", instant("9h"))
	if len(p.Protected) != 0 || len(w) != 2 {
		t.Fatalf("not explicit withdrawal: %#v %#v", p, w)
	}
	// Merely removing the Grant keeps Binding-owned protection.
	noGrant, _ := Decode([]byte(resourceYAML + "---\n" + bindingYAML))
	retained, _, e := Replace(old, noGrant, facts())
	if e != nil {
		t.Fatal(e)
	}
	p, _ = retained.Plan().Projection("agent-a", instant("9h"))
	if len(p.Protected) != 2 || len(p.Allow) != 0 {
		t.Fatal("last Grant removal lost protection")
	}
}
func TestCandidateOwnsInputsAndAbsoluteReplay(t *testing.T) {
	data := []byte(policy())
	b, e := Decode(data)
	if e != nil {
		t.Fatal(e)
	}
	for i := range data {
		data[i] = 'x'
	}
	f := facts()
	c, e := Compile(b, f)
	if e != nil {
		t.Fatal(e)
	}
	f.Agents[0].Labels["env"] = "attacker"
	f.Contexts[0].Destinations[0] = netip.MustParsePrefix("0.0.0.0/0")
	input := c.Input()
	input.Logical.Grants[0].Validity.ExpiresAt = instant("1000h")
	input.Bindings[0].Resources[0].ResourceName = "other"
	input.Agents[0].Labels["env"] = "changed"
	ids := c.Objects()
	ids[0].UID = "changed"
	if c.Hash() == "" || c.Input().Agents[0].Labels["env"] != "dev" || c.Objects()[0].UID == "changed" {
		t.Fatal("aliased accepted state")
	}
	next, _, e := Replace(c, b, facts())
	if e != nil {
		t.Fatal(e)
	}
	p, _ := next.Plan().Projection("agent-a", instant("10h"))
	if len(p.Allow) != 0 {
		t.Fatal("replay renewed deadline")
	}
	if _, e = Compile(nil, facts()); e == nil {
		t.Fatal("nil bundle accepted")
	}
	if _, _, e = Replace(nil, b, facts()); e == nil {
		t.Fatal("nil previous accepted")
	}
}

func TestSelectionDefaultsAndNoMatchNeverBroaden(t *testing.T) {
	whole := strings.Replace(grantYAML, "      endpointNames: [https]\n", "", 1)
	c := candidate(t, resourceYAML+"---\n"+whole+"---\n"+bindingYAML)
	p, _ := c.Plan().Projection("agent-a", instant("9h"))
	if len(p.Allow) != 2 {
		t.Fatalf("omitted selection should cover two entries: %#v", p)
	}
	noMatch := strings.Replace(grantYAML, "endpointNames: [https]", "endpointSelector: {matchLabels: {interface: absent}}", 1)
	c = candidate(t, resourceYAML+"---\n"+noMatch+"---\n"+bindingYAML)
	p, _ = c.Plan().Projection("agent-a", instant("9h"))
	if len(p.Allow) != 0 || len(p.Protected) != 2 {
		t.Fatal("empty selector match broadened")
	}
	agentSelector := strings.Replace(bindingYAML, "agentRef: {name: agent-a}", "agentSelector: {matchLabels: {env: dev}}", 1)
	c = candidate(t, resourceYAML+"---\n"+whole+"---\n"+agentSelector)
	p, _ = c.Plan().Projection("agent-a", instant("9h"))
	if len(p.Allow) != 2 {
		t.Fatal("Agent label selection not resolved")
	}
	permanent := strings.Replace(grantYAML, "mode: bounded, expiresAt: '2026-10-15T10:00:00Z'", "mode: permanent", 1)
	c = candidate(t, resourceYAML+"---\n"+permanent+"---\n"+bindingYAML)
	p, _ = c.Plan().Projection("agent-a", instant("1000h"))
	if len(p.Allow) != 1 {
		t.Fatal("explicit permanent rejected")
	}
	revoked := strings.Replace(permanent, "spec:\n", "spec:\n  revoked: true\n", 1)
	c = candidate(t, resourceYAML+"---\n"+revoked+"---\n"+bindingYAML)
	p, _ = c.Plan().Projection("agent-a", instant("9h"))
	if len(p.Allow) != 0 {
		t.Fatal("revoked permanent active")
	}
}
