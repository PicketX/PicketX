// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"os"
	"strings"
	"testing"
)

func TestResourceNetworkEditSoleDocument(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, ending := range []string{"", "...\n# external trailing comment\n"} {
		source := strings.Split(string(raw), "---\n")[0] + ending
		b, err := Decode([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		out, err := b.UpdateResourceNetwork("resource-gitea-v1", []InventoryEndpoint{{Name: "https", Protocol: "TCP", Port: 8443}, {Name: "ssh", Protocol: "UDP", Port: 2222}})
		if err != nil {
			t.Fatal(err)
		}
		next, err := Decode(out)
		if err != nil || len(next.objects) != 1 || next.objects[0].Metadata.UID != "resource-gitea-v1" {
			t.Fatal(err, string(out))
		}
		if ending != "" && !strings.HasSuffix(string(out), "# external trailing comment\n") {
			t.Fatal("lost trailing bytes", string(out))
		}
		if string(b.source) != source || b.objects[0].Spec.(resourceSpec).Endpoints[0].Network.Port != 443 {
			t.Fatal("input mutated")
		}
	}
}
