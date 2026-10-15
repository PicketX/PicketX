// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"fmt"
	"strings"
	"testing"
)

const resourceYAML = `apiVersion: picketx.io/v1alpha1
kind: Resource
metadata: {name: gitea, uid: resource-gitea-v1}
spec:
  endpoints:
    - name: https
      network: {protocol: TCP, port: 443}
    - name: ssh
      network: {protocol: TCP, port: 22}
`
const grantYAML = `apiVersion: picketx.io/v1alpha1
kind: Grant
metadata: {name: browser, uid: grant-browser-v1}
spec:
  subject: {type: source, cidr: 198.51.100.9}
  scopes:
    - resourceRef: {name: gitea, uid: resource-gitea-v1}
      endpointNames: [https]
  validity: {mode: bounded, expiresAt: '2026-10-15T10:00:00Z'}
`
const bindingYAML = `apiVersion: picketx.io/v1alpha1
kind: EnforcementBinding
metadata: {name: gitea-host, uid: binding-gitea-v1}
spec:
  resources:
    - resourceRef: {name: gitea, uid: resource-gitea-v1}
  agents:
    - agentRef: {name: agent-a}
  enforcement:
    profileRef: host
    class: host-local
    stage: original-destination
    allowedDestinations: [192.0.2.100, '2001:db8::100']
    handling: graceful
`
const emptyYAML = `apiVersion: picketx.io/v1alpha1
kind: Configuration
metadata: {name: empty, uid: empty-v1}
spec: {empty: true}
`

func policy() string { return resourceYAML + "---\n" + grantYAML + "---\n" + bindingYAML }

func TestDecodeHashOrderAndDefaults(t *testing.T) {
	a, e := Decode([]byte(policy()))
	if e != nil {
		t.Fatal(e)
	}
	explicit := strings.Replace(resourceYAML, "- name: https", "- name: https\n      type: network", 1)
	explicitGrant := strings.Replace(grantYAML, "endpointNames: [https]", "endpointNames: [https]\n      actions: [connect]", 1)
	b, e := Decode([]byte(bindingYAML + "---\n" + explicitGrant + "---\n" + explicit))
	if e != nil {
		t.Fatal(e)
	}
	if a.hash == "" || a.hash != b.hash {
		t.Fatalf("canonical hashes differ: %s %s", a.hash, b.hash)
	}
}
func TestDecodeRejectsAmbiguousOrUnsupportedSyntax(t *testing.T) {
	tests := map[string]string{
		"empty": "", "null": "null", "empty map": "{}", "sequence": "[]",
		"unknown version":        strings.Replace(resourceYAML, "v1alpha1", "v99", 1),
		"unknown kind":           strings.Replace(resourceYAML, "Resource", "AccessPolicy", 1),
		"unknown header":         resourceYAML + "status: {applied: true}\n",
		"unknown nested":         strings.Replace(resourceYAML, "port: 443", "port: 443, protcol: TCP", 1),
		"unknown metadata":       strings.Replace(resourceYAML, "uid: resource-gitea-v1", "uid: resource-gitea-v1, namespace: default", 1),
		"null restriction":       strings.Replace(grantYAML, "endpointNames: [https]", "endpointNames: null", 1),
		"null optional":          strings.Replace(resourceYAML, "name: https", "name: https\n      type: null", 1),
		"duplicate":              strings.Replace(resourceYAML, "port: 443", "port: 443, port: 22", 1),
		"numeric name":           strings.Replace(resourceYAML, "name: https", "name: 443", 1),
		"string port":            strings.Replace(resourceYAML, "port: 443", "port: '443'", 1),
		"float port":             strings.Replace(resourceYAML, "port: 443", "port: 443.0", 1),
		"alias":                  strings.Replace(resourceYAML, "name: https", "name: &a https", 1),
		"custom tag":             strings.Replace(resourceYAML, "name: https", "name: !thing https", 1),
		"merge":                  strings.Replace(resourceYAML, "name: https", "<<: {name: https}", 1),
		"missing uid":            strings.Replace(resourceYAML, ", uid: resource-gitea-v1", "", 1),
		"invalid uid":            strings.Replace(resourceYAML, "resource-gitea-v1", "bad uid", 1),
		"duplicate name":         resourceYAML + "---\n" + strings.ReplaceAll(resourceYAML, "resource-gitea-v1", "resource-gitea-v2"),
		"duplicate uid":          resourceYAML + "---\n" + strings.ReplaceAll(resourceYAML, "name: gitea", "name: other"),
		"empty input withdrawal": emptyYAML + "---\n" + resourceYAML,
		"false withdrawal":       strings.Replace(emptyYAML, "empty: true", "empty: false", 1),
	}
	for n, s := range tests {
		t.Run(n, func(t *testing.T) {
			if b, e := Decode([]byte(s)); e == nil || b != nil {
				t.Fatalf("accepted %s: %#v", n, b)
			}
		})
	}
}
func TestDecodeLimitsAndNoPartialBundle(t *testing.T) {
	for name, s := range map[string]string{
		"bytes": strings.Repeat(" ", MaxBytes+1),
		"depth": strings.Repeat("[", 65) + "x" + strings.Repeat("]", 65),

		"trailing invalid": resourceYAML + "---\n[",
	} {
		t.Run(name, func(t *testing.T) {
			b, e := Decode([]byte(s))
			if e == nil || b != nil {
				t.Fatal("accepted invalid complete candidate")
			}
		})
	}
}

func TestDecodeRejectsAlternateFieldCasing(t *testing.T) {
	for _, s := range []string{
		strings.Replace(resourceYAML, "port: 443", "port: 443, Port: 22", 1),
		strings.Replace(grantYAML, "endpointNames: [https]", "EndpointNames: [https]", 1),
		strings.Replace(resourceYAML, "uid: resource-gitea-v1", "UID: resource-gitea-v1", 1),
	} {
		if b, e := Decode([]byte(s)); e == nil || b != nil {
			t.Fatal("accepted non-contract field casing")
		}
	}
}

func TestDecodeObjectCountBound(t *testing.T) {
	var input strings.Builder
	for i := 0; i < MaxObjects+1; i++ {
		r := strings.ReplaceAll(resourceYAML, "gitea", fmt.Sprintf("gitea-%d", i))
		input.WriteString(r)
		input.WriteString("---\n")
	}
	_, err := Decode([]byte(input.String()))
	if err == nil || !strings.Contains(err.Error(), "too many configuration objects") {
		t.Fatalf("count guard did not run: %v", err)
	}
}
