// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PicketX/PicketX/internal/runtimeconfig"
)

func initCommand(t *testing.T, dir, assets string, extra ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	c := New(&out, &out)
	args := []string{"controller-init", "--directory", dir, "--assets", assets, "--origin", "http://127.0.0.1:8080"}
	c.SetArgs(append(args, extra...))
	err := c.Execute()
	return out.String(), err
}

func initAssets(t *testing.T) string {
	t.Helper()
	p := t.TempDir()
	if err := os.WriteFile(filepath.Join(p, "index.html"), []byte("<html>built web</html>"), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Catches accidental default file authority, shared keys, broad scopes or echoed keys.
func TestControllerInitPrivateIndependentAudiences(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "instance with spaces")
	assets := initAssets(t)
	out, err := initCommand(t, dir, assets)
	if err != nil {
		t.Fatal(err)
	}
	c, err := runtimeconfig.Load(filepath.Join(dir, "controller.yaml"), nil, nil)
	if err != nil {
		t.Fatal("generated runtime rejected", err)
	}
	if c.Source.Mode != "database" || c.Source.Policy != "" || c.Source.Facts != "" || c.Management.UsersMode != "database" || !c.Browser.Enabled || c.Browser.Assets != assets || c.HTTPListen != "127.0.0.1:8080" || c.Agent.Listen != "127.0.0.1:9090" || c.Authz.Address != "127.0.0.1:9081" {
		t.Fatal("unsafe generated runtime", c)
	}
	if !filepath.IsAbs(c.Database) || !strings.HasPrefix(c.Database, dir+string(os.PathSeparator)) || c.Agent.Authority == "" {
		t.Fatal("missing private authority")
	}
	keys := map[string]bool{}
	for _, pair := range [][3]string{{"bootstrap-key", "principals.json", "principals"}, {"agent-token", "agents.json", "credentials"}, {"ingress-token", "callers.json", "callers"}} {
		raw, err := os.ReadFile(filepath.Join(dir, pair[0]))
		if err != nil {
			t.Fatal(err)
		}
		key := strings.TrimSuffix(string(raw), "\n")
		decoded, err := hex.DecodeString(key)
		if err != nil || len(decoded) != 32 || keys[key] || strings.Contains(out, key) {
			t.Fatal("bad, reused or exposed key")
		}
		keys[key] = true
		b, err := os.ReadFile(filepath.Join(dir, pair[1]))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(b, []byte(key)) {
			t.Fatal("raw key in registry")
		}
		var document map[string][]map[string]any
		if err := json.Unmarshal(b, &document); err != nil {
			t.Fatal(err)
		}
		rows := document[pair[2]]
		if len(rows) != 1 {
			t.Fatal("wrong credential count")
		}
		sum := sha256.Sum256([]byte(key))
		if rows[0]["sha256"] != hex.EncodeToString(sum[:]) {
			t.Fatal("digest mismatch")
		}
		switch pair[0] {
		case "bootstrap-key":
			if rows[0]["id"] != "principal:bootstrap" || rows[0]["roles"].([]any)[0] != "role:admin" {
				t.Fatal(rows)
			}
		case "agent-token":
			if rows[0]["agent_ref"] != "node-a" {
				t.Fatal(rows)
			}
		case "ingress-token":
			if rows[0]["routes"].([]any)[0] != "route-web" || len(rows[0]["routes"].([]any)) != 1 {
				t.Fatal(rows)
			}
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 7 {
		t.Fatal("unexpected seeded files", len(entries))
	}
	for _, ent := range entries {
		info, _ := ent.Info()
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
			t.Fatal(ent.Name(), info.Mode())
		}
	}
	info, _ := os.Stat(dir)
	if info.Mode().Perm() != 0700 {
		t.Fatal(info.Mode())
	}
	before, _ := os.ReadFile(filepath.Join(dir, "bootstrap-key"))
	if _, err = initCommand(t, dir, assets); err == nil {
		t.Fatal("rerun overwrote initialized instance")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "bootstrap-key"))
	if !bytes.Equal(before, after) {
		t.Fatal("rerun changed key")
	}
	other := filepath.Join(t.TempDir(), "other")
	if _, err = initCommand(t, other, assets); err != nil {
		t.Fatal(err)
	}
	b, _ := runtimeconfig.Load(filepath.Join(other, "controller.yaml"), nil, nil)
	if b.Agent.Authority == c.Agent.Authority {
		t.Fatal("authority reused")
	}
}

// Catches partial output for invalid parameters or path traversal via a symlink parent.
func TestControllerInitRejectsBeforeWriting(t *testing.T) {
	assets := initAssets(t)
	for _, args := range [][]string{{"--origin", "http://public.example"}, {"--origin", "https://example.test/path"}, {"--listen", "0.0.0.0:8080"}, {"--agent-listen", "[::]:9090"}, {"--authz-listen", "127.0.0.1:0"}, {"--agent-ref", "bad/ref"}, {"--route-id", "bad route"}, {"--assets", "/does-not-exist"}, {"extra"}} {
		dir := filepath.Join(t.TempDir(), "no-output")
		out, err := initCommand(t, dir, assets, args...)
		if err == nil {
			t.Fatalf("accepted invalid %v", args)
		}
		if out != "" {
			t.Fatal("printed success on failure", out)
		}
		if _, err = os.Lstat(dir); !os.IsNotExist(err) {
			t.Fatal("partial invalid init")
		}
	}
	parent := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(parent, link); err != nil {
		t.Fatal(err)
	}
	if _, err := initCommand(t, filepath.Join(link, "no-output"), assets); err == nil {
		t.Fatal("symlink parent accepted")
	}
	if files, _ := os.ReadDir(parent); len(files) != 0 {
		t.Fatal("wrote through parent symlink")
	}
	for _, kind := range []string{"dir", "file", "link"} {
		target := filepath.Join(t.TempDir(), "existing")
		switch kind {
		case "dir":
			_ = os.Mkdir(target, 0700)
		case "file":
			_ = os.WriteFile(target, []byte("keep"), 0600)
		case "link":
			_ = os.Symlink(parent, target)
		}
		if _, err := initCommand(t, target, assets); err == nil {
			t.Fatal("existing target accepted", kind)
		}
	}
}
