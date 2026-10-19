// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ingressKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func nginxCommand(dir, key string, extra ...string) (string, error) {
	var out bytes.Buffer
	c := New(&out, &out)
	c.SetArgs(append([]string{"nginx-init", "--directory", dir, "--token-file", key, "--route-id", "route-web", "--authz-address", "127.0.0.1:9081"}, extra...))
	err := c.Execute()
	return out.String(), err
}

// Catch incorrect command wiring, source-key rewriting, disclosure and overwrite.
func TestNginxInitPrivateOutputs(t *testing.T) {
	parent := t.TempDir()
	key := filepath.Join(parent, "ingress-token")
	if err := os.WriteFile(key, []byte(ingressKey+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(parent, "private includes with spaces")
	out, err := nginxCommand(dir, key)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, ingressKey) {
		t.Fatal("secret in output")
	}
	info, err := os.Stat(dir)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("private directory: %v %v", info, err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 2 {
		t.Fatalf("expected two includes, got %d", len(files))
	}
	for _, name := range []string{"location.conf", "server.conf"} {
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
			t.Fatalf("private include %s: %v %v", name, info, err)
		}
	}
	server, _ := os.ReadFile(filepath.Join(dir, "server.conf"))
	if !bytes.Contains(server, []byte(ingressKey)) {
		t.Fatal("dedicated credential not rendered")
	}
	if _, err := nginxCommand(dir, key); err == nil {
		t.Fatal("existing target overwritten")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "server.conf"))
	if !bytes.Equal(server, after) {
		t.Fatal("existing config changed")
	}
	raw, _ := os.ReadFile(key)
	if string(raw) != ingressKey+"\n" {
		t.Fatal("source credential changed")
	}
	for _, address := range []string{"[::1]:9081", "127.0.0.2:443"} {
		d := filepath.Join(parent, strings.NewReplacer(":", "_", "[", "", "]", "").Replace(address))
		if _, err := nginxCommand(d, key, "--authz-address", address, "--route-id", "another-route"); err != nil {
			t.Fatal(err)
		}
	}
}

// Every invalid input must fail before creating output or disclosing a token.
func TestNginxInitRejectsUnsafeInputs(t *testing.T) {
	for _, address := range []string{"localhost:9081", "0.0.0.0:9081", "[::]:9081", "192.0.2.1:9081", "[::ffff:127.0.0.1]:9081", "[::1%lo]:9081", "127.0.0.1:0", "127.0.0.1:65536", "127.0.0.1:9081;foo"} {
		t.Run(address, func(t *testing.T) {
			p := t.TempDir()
			k := filepath.Join(p, "key")
			_ = os.WriteFile(k, []byte(ingressKey), 0600)
			d := filepath.Join(p, "new")
			if _, err := nginxCommand(d, k, "--authz-address", address); err == nil {
				t.Fatal("unsafe callback accepted")
			}
			if _, err := os.Lstat(d); !os.IsNotExist(err) {
				t.Fatal("output created")
			}
		})
	}
	for _, token := range []string{"", strings.Repeat("a", 63), strings.Repeat("a", 66), strings.Repeat("A", 64), ingressKey + "\n\n", ingressKey + " ", strings.Repeat("$", 64), strings.Repeat("\"", 64), strings.Repeat("a", 1024)} {
		t.Run("token", func(t *testing.T) {
			p := t.TempDir()
			k := filepath.Join(p, "key")
			_ = os.WriteFile(k, []byte(token), 0600)
			d := filepath.Join(p, "new")
			out, err := nginxCommand(d, k)
			if err == nil {
				t.Fatal("unsafe token accepted")
			}
			if len(token) > 0 && strings.Contains(out+err.Error(), token) {
				t.Fatal("secret disclosed")
			}
			if _, err := os.Lstat(d); !os.IsNotExist(err) {
				t.Fatal("output created")
			}
		})
	}
	for _, scenario := range []string{"public-key", "symlink-key", "symlink-parent", "symlink-target", "directory-key", "relative-key", "relative-output", "route-injection"} {
		t.Run(scenario, func(t *testing.T) {
			p := t.TempDir()
			k := filepath.Join(p, "key")
			_ = os.WriteFile(k, []byte(ingressKey), 0600)
			d := filepath.Join(p, "new")
			var args []string
			switch scenario {
			case "public-key":
				_ = os.Chmod(k, 0644)
			case "symlink-key":
				link := filepath.Join(p, "link")
				_ = os.Symlink(k, link)
				k = link
			case "symlink-parent":
				link := filepath.Join(p, "link")
				_ = os.Symlink(p, link)
				d = filepath.Join(link, "new")
			case "symlink-target":
				_ = os.Symlink(k, d)
			case "directory-key":
				k = p
			case "relative-key":
				k = "key"
			case "relative-output":
				d = "new"
			case "route-injection":
				args = []string{"--route-id", "route;return 200"}
			}
			if _, err := nginxCommand(d, k, args...); err == nil {
				t.Fatal("unsafe input accepted")
			}
			raw, _ := os.ReadFile(filepath.Join(p, "key"))
			if string(raw) != ingressKey {
				t.Fatal("source changed")
			}
			if scenario != "symlink-target" {
				if _, err := os.Lstat(d); !os.IsNotExist(err) {
					t.Fatal("output created")
				}
			}
		})
	}
}
