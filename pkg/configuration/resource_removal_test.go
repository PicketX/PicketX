// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRemoveResourceSourceAndUnknownReferences(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Split(string(raw), "---\n")[0] + "...\n# retain operator note\n"
	b, err := Decode([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	out, err := b.RemoveResource("resource-gitea-v1")
	if err != nil {
		t.Fatal(err)
	}
	next, err := Decode(out)
	if err != nil || len(next.objects) != 1 || next.objects[0].Kind != "Configuration" || !bytes.HasSuffix(out, []byte("# retain operator note\n")) {
		t.Fatal("empty/source", err, string(out))
	}
	// Future decoded kinds cannot silently bypass reference classification.
	b.objects = append(b.objects, document{Kind: "FutureReference", Spec: struct{}{}})
	if _, err = b.RemoveResource("resource-gitea-v1"); err == nil {
		t.Fatal("unknown reference kind accepted")
	}
	if string(b.source) != source {
		t.Fatal("input mutated")
	}
}
