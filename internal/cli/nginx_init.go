// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PicketX/PicketX/pkg/httpauth"
	"github.com/spf13/cobra"
)

type nginxInitOptions struct{ directory, tokenFile, routeID, address string }

func nginxInitCommand() *cobra.Command {
	var o nginxInitOptions
	c := &cobra.Command{Use: "nginx-init", Short: "Generate NEW private Nginx authorization includes (no reload or Grant)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := initializeNginx(o); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Private includes: %s\nInclude location.conf inside the protected location; server.conf inside its server.\nInspect privately and run nginx -t before manual reload. No service changed; no permission granted.\n", o.directory)
			return err
		}}
	c.Flags().StringVar(&o.directory, "directory", "", "New absolute private directory; existing non-symlink parent")
	c.Flags().StringVar(&o.tokenFile, "token-file", "", "Absolute private ingress-token file from controller-init (64 lowercase hex)")
	c.Flags().StringVar(&o.routeID, "route-id", "", "Fixed HTTP route UID, also permitted by caller registry")
	c.Flags().StringVar(&o.address, "authz-address", "", "Literal loopback IPv4/IPv6 decision IP:nonzero-port")
	for _, name := range []string{"directory", "token-file", "route-id", "authz-address"} {
		_ = c.MarkFlagRequired(name)
	}
	return c
}

func initializeNginx(o nginxInitOptions) error {
	if !filepath.IsAbs(o.directory) || !filepath.IsAbs(o.tokenFile) {
		return fmt.Errorf("directory and token-file require absolute paths")
	}
	for _, parent := range []string{filepath.Dir(filepath.Clean(o.directory)), filepath.Dir(filepath.Clean(o.tokenFile))} {
		resolved, err := filepath.EvalSymlinks(parent)
		if err != nil || resolved != parent {
			return fmt.Errorf("parents must exist without symlinks")
		}
	}
	if !httpauth.ValidID(o.routeID) {
		return fmt.Errorf("invalid fixed route UID")
	}
	host, port, err := net.SplitHostPort(o.address)
	if err != nil {
		return fmt.Errorf("callback requires loopback IP and nonzero port")
	}
	ip, err := netip.ParseAddr(host)
	n, pe := strconv.Atoi(port)
	if err != nil || !ip.IsLoopback() || ip.Is4In6() || ip.Zone() != "" || pe != nil || n < 1 || n > 65535 {
		return fmt.Errorf("callback requires loopback IP and nonzero port")
	}
	address := net.JoinHostPort(ip.String(), strconv.Itoa(n))
	// The trusted operator exclusively controls both parents. Reject symlinks and
	// verify the opened file is the same regular private file observed before open.
	info, err := os.Lstat(o.tokenFile)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("token-file must be a private regular non-symlink file")
	}
	f, err := os.Open(o.tokenFile)
	if err != nil {
		return fmt.Errorf("cannot open private token-file")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() || opened.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("token-file changed or is not private")
	}
	data, err := io.ReadAll(io.LimitReader(f, 66))
	if err != nil {
		return fmt.Errorf("cannot read private token-file")
	}
	token := strings.TrimSuffix(string(data), "\n")
	if len(token) != 64 {
		return fmt.Errorf("token-file requires controller-init 64 lowercase hex token with optional single LF")
	}
	for _, b := range []byte(token) {
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f') {
			return fmt.Errorf("token-file requires controller-init 64 lowercase hex token with optional single LF")
		}
	}
	sum := sha256.Sum256([]byte(o.routeID))
	id := hex.EncodeToString(sum[:])
	method := "$picketx_method_" + id
	callback := "/_picketx_auth_" + id
	location := fmt.Sprintf("# Private PicketX protected-location include. No Grant is created.\nsatisfy all;\nset %s $request_method;\nauth_request %s;\n", method, callback)
	server := fmt.Sprintf(`# Private PicketX server-context include. Contains a secret; never log nginx -T.
location = %s {
    internal;
    auth_request off;
    proxy_pass http://%s/authz/v1/nginx;
    proxy_method GET;
    proxy_pass_request_body off;
    proxy_pass_request_headers off;
    proxy_set_header Content-Length "";
    proxy_set_header Host localhost;
    proxy_set_header Authorization "Bearer %s";
    proxy_set_header X-PicketX-Route "%s";
    proxy_set_header X-PicketX-Source $remote_addr;
    proxy_set_header X-PicketX-Host $http_host;
    proxy_set_header X-PicketX-Method %s;
    proxy_set_header X-PicketX-URI $request_uri;
    proxy_cache off;
    proxy_intercept_errors off;
    proxy_connect_timeout 1s;
    proxy_read_timeout 2s;
    proxy_send_timeout 2s;
}
`, callback, address, token, o.routeID, method)
	// Exclusive creation; on any I/O failure retain partial private output for
	// operator inspection, never remove or overwrite existing installation state.
	if err := os.Mkdir(o.directory, 0700); err != nil {
		return fmt.Errorf("cannot create NEW private include directory")
	}
	root, err := os.OpenRoot(o.directory)
	if err != nil {
		return fmt.Errorf("cannot open new include directory")
	}
	defer root.Close()
	for _, item := range []struct{ name, body string }{{"location.conf", location}, {"server.conf", server}} {
		file, err := root.OpenFile(item.name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("cannot create private include; inspect partial directory")
		}
		_, writeErr := io.WriteString(file, item.body)
		syncErr := file.Sync()
		closeErr := file.Close()
		if writeErr != nil || syncErr != nil || closeErr != nil {
			return fmt.Errorf("cannot persist private include; inspect partial directory")
		}
	}
	return nil
}
