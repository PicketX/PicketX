// SPDX-License-Identifier: Apache-2.0
// Package controllerclient reads authenticated northbound diagnostic snapshots.
package controllerclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Origin, KeyFile, CAFile, Agent string
	Timeout                        time.Duration
}

var errInput = errors.New("invalid diagnostic options or private management-key file")
var errRead = errors.New("Controller diagnostic read failed; check address, trust, credentials and availability")
var errData = errors.New("Controller returned invalid or inconsistent diagnostic data; retry a fresh read")

type client struct {
	origin, key string
	http        *http.Client
}

func newClient(o Options) (*client, error) {
	u, e := url.Parse(o.Origin)
	if e != nil || u == nil || u.Host == "" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.RawPath != "" || (u.Scheme != "http" && u.Scheme != "https") || o.Timeout <= 0 || o.Timeout > 30*time.Second || (o.Agent != "" && !validID(o.Agent)) {
		return nil, errInput
	}
	if u.Scheme == "http" {
		ip, e := netip.ParseAddr(u.Hostname())
		if e != nil || !ip.IsLoopback() || ip.Is4In6() || ip.Zone() != "" {
			return nil, errInput
		}
	}
	if strings.Contains(u.Host, "%") {
		return nil, errInput
	}
	raw, e := readFile(o.KeyFile, 257, true)
	if e != nil {
		return nil, errInput
	}
	key := strings.TrimSuffix(string(raw), "\n")
	if len(key) < 32 || len(key) > 256 {
		return nil, errInput
	}
	for _, b := range []byte(key) {
		if b < 33 || b > 126 {
			return nil, errInput
		}
	}
	var roots *x509.CertPool
	if o.CAFile != "" {
		ca, e := readFile(o.CAFile, 4<<20, false)
		if e != nil {
			return nil, errInput
		}
		roots, e = x509.SystemCertPool()
		if e != nil {
			return nil, errInput
		}
		if !roots.AppendCertsFromPEM(ca) {
			return nil, errInput
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil // Diagnostic credentials do not follow ambient proxy settings.
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}
	transport.ResponseHeaderTimeout = o.Timeout
	transport.TLSHandshakeTimeout = o.Timeout
	u.Path = ""
	u.RawPath = ""
	return &client{origin: u.String(), key: key, http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func readFile(path string, limit int64, private bool) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, errInput
	}
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() || (private && info.Mode().Perm()&0077 != 0) {
		return nil, errInput
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, errInput
	}
	defer f.Close()
	actual, e := f.Stat()
	if e != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() || (private && actual.Mode().Perm()&0077 != 0) {
		return nil, errInput
	}
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil || int64(len(b)) > limit {
		return nil, errInput
	}
	return b, nil
}

func (c *client) get(ctx context.Context, path string, auth bool, target any, limit int64) (int, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, c.origin+path, nil)
	if e != nil {
		return 0, errRead
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	req.Header.Set("Accept", "application/json")
	r, e := c.http.Do(req)
	if e != nil {
		return 0, errRead
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return r.StatusCode, nil
	}
	if target == nil {
		return r.StatusCode, nil
	}
	media, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil || media != "application/json" {
		return 0, errData
	}
	raw, e := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if e != nil || int64(len(raw)) > limit {
		return 0, errData
	}
	if decode(raw, target) != nil {
		return 0, errData
	}
	return r.StatusCode, nil
}

// Server reports deliberately allow report:null. Reject ambiguity before normal
// typed decoding, without exposing any key, body fragment or decoder diagnostic.
func decode(raw []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	if walk(d, 0, false) != nil {
		return errData
	}
	if _, e := d.Token(); e != io.EOF {
		return errData
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil {
		return errData
	}
	return nil
}

// Exact wire spellings prevent encoding/json's case-folded aliases from
// overwriting trusted fields. Typed decoding below also checks field placement.
var wireFields = map[string]bool{
	"authority": true, "revision": true, "policy": true, "facts": true,
	"observed_at": true, "nodes": true, "agent_ref": true, "desired_revision": true,
	"desired_sha256": true, "connection": true, "freshness": true,
	"fresh_for_millis": true, "matches_desired": true, "report": true,
	"sha256": true, "phase": true, "received_at": true,
}

func walk(d *json.Decoder, depth int, nullable bool) error {
	if depth > 32 {
		return errData
	}
	t, e := d.Token()
	if e != nil {
		return errData
	}
	if t == nil && !nullable {
		return errData
	}
	if delimiter, ok := t.(json.Delim); ok {
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, e := d.Token()
				s, ok := key.(string)
				if e != nil || !ok || seen[s] || !wireFields[s] {
					return errData
				}
				seen[s] = true
				if walk(d, depth+1, s == "report") != nil {
					return errData
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim('}') {
				return errData
			}
		case '[':
			for d.More() {
				if walk(d, depth+1, false) != nil {
					return errData
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim(']') {
				return errData
			}
		default:
			return errData
		}
	}
	return nil
}
