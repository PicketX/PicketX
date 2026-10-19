// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/pkg/httpauth"
	"github.com/spf13/cobra"
)

type controllerInitOptions struct {
	directory, assets, origin, http, agent, authz, agentRef, routeID string
}

func controllerInitCommand() *cobra.Command {
	var o controllerInitOptions
	c := &cobra.Command{Use: "controller-init", Short: "Create a NEW private database-authoritative Controller instance (never overwrite)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := initializeController(o); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Initialized private Controller directory: %s\nRecovery key: %s\nKeep all key files private. No service started; no network access granted.\n", o.directory, filepath.Join(o.directory, "bootstrap-key"))
			return err
		}}
	f := c.Flags()
	f.StringVar(&o.directory, "directory", "", "New absolute private directory; parent must exist without symlinks")
	f.StringVar(&o.assets, "assets", "", "Absolute deployed Web directory containing index.html")
	f.StringVar(&o.origin, "origin", "", "Exact public HTTPS origin (HTTP only for loopback)")
	f.StringVar(&o.http, "listen", "127.0.0.1:8080", "Loopback HTTP IP:port behind existing ingress")
	f.StringVar(&o.agent, "agent-listen", "127.0.0.1:9090", "Loopback Agent IP:port; remote access requires explicit later TLS/tunnel configuration")
	f.StringVar(&o.authz, "authz-listen", "127.0.0.1:9081", "Loopback decision IP:port")
	f.StringVar(&o.agentRef, "agent-ref", "node-a", "Initial Agent ref (letters/digits/dot/underscore/hyphen)")
	f.StringVar(&o.routeID, "route-id", "route-web", "Only HTTP route UID the initial ingress key can query")
	for _, flag := range []string{"directory", "assets", "origin"} {
		_ = c.MarkFlagRequired(flag)
	}
	return c
}

func (o controllerInitOptions) validate() error {
	if !filepath.IsAbs(o.directory) || !filepath.IsAbs(o.assets) {
		return fmt.Errorf("directory and assets must be absolute paths")
	}
	parent := filepath.Dir(filepath.Clean(o.directory))
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return fmt.Errorf("directory parent must exist without symlinks")
	}
	assets, err := filepath.EvalSymlinks(o.assets)
	if err != nil || assets != filepath.Clean(o.assets) {
		return fmt.Errorf("assets must exist without symlinks")
	}
	info, err := os.Lstat(filepath.Join(assets, "index.html"))
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("assets require a regular index.html")
	}
	if err := management.ValidateBrowserOrigin(o.origin); err != nil {
		return err
	}
	if !httpauth.ValidID(o.agentRef) || !httpauth.ValidID(o.routeID) {
		return fmt.Errorf("invalid Agent ref or route UID")
	}
	seen := map[string]bool{}
	for _, address := range []string{o.http, o.agent, o.authz} {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("listeners require loopback IP and nonzero port")
		}
		ip, err := netip.ParseAddr(host)
		n, pe := strconv.Atoi(port)
		if err != nil || !ip.IsLoopback() || ip.Is4In6() || ip.Zone() != "" || pe != nil || n < 1 || n > 65535 {
			return fmt.Errorf("listeners require loopback IP and nonzero port")
		}
		key := net.JoinHostPort(ip.String(), strconv.Itoa(n))
		if seen[key] {
			return fmt.Errorf("listeners must be distinct")
		}
		seen[key] = true
	}
	return nil
}

// initializeController writes only inside a newly-created private directory.
// The parent must be trusted and remain under the operator's control. On failure
// leave the directory for inspection: never recursively remove partially written state.
func initializeController(o controllerInitOptions) error {
	if err := o.validate(); err != nil {
		return err
	}
	type keyPair struct{ raw, digest string }
	var keys [3]keyPair
	for i := range keys {
		var b [32]byte
		if _, err := rand.Read(b[:]); err != nil {
			return fmt.Errorf("generate private credential: %w", err)
		}
		keys[i].raw = hex.EncodeToString(b[:])
		digest := sha256.Sum256([]byte(keys[i].raw))
		keys[i].digest = hex.EncodeToString(digest[:])
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return fmt.Errorf("generate authority: %w", err)
	}
	dir := filepath.Clean(o.directory)
	path := func(name string) string { return filepath.Join(dir, name) }
	documents := map[string]any{
		"principals.json": map[string]any{"principals": []any{map[string]any{"id": "principal:bootstrap", "sha256": keys[0].digest, "roles": []string{"role:admin"}}}},
		"agents.json":     map[string]any{"credentials": []any{map[string]any{"id": "initial-agent", "agent_ref": o.agentRef, "sha256": keys[1].digest}}},
		"callers.json":    map[string]any{"callers": []any{map[string]any{"id": "initial-ingress", "routes": []string{o.routeID}, "sha256": keys[2].digest}}},
		"controller.yaml": map[string]any{
			"http":        map[string]any{"listen": o.http},
			"agent":       map[string]any{"enabled": true, "listen": o.agent, "authority": "controller:" + hex.EncodeToString(id[:])},
			"authz":       map[string]any{"enabled": true, "network": "tcp", "address": o.authz, "credentials": path("callers.json")},
			"source":      map[string]any{"mode": "database"},
			"management":  map[string]any{"credentials": path("principals.json"), "users_mode": "database"},
			"browser":     map[string]any{"enabled": true, "origin": o.origin, "assets": filepath.Clean(o.assets)},
			"credentials": path("agents.json"), "database": path("state/controller.sqlite"),
		},
	}
	files := map[string][]byte{}
	for name, doc := range documents {
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		files[name] = append(b, '\n')
	}
	for i, name := range []string{"bootstrap-key", "agent-token", "ingress-token"} {
		files[name] = []byte(keys[i].raw + "\n")
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return fmt.Errorf("create new private directory (existing paths are never reused): %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	for name, data := range files {
		f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("create private instance file: %w", err)
		}
		_, writeErr := f.Write(data)
		if writeErr == nil {
			writeErr = f.Sync()
		}
		closeErr := f.Close()
		if writeErr != nil {
			return fmt.Errorf("write private instance file: %w", writeErr)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
