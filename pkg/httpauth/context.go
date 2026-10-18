// SPDX-License-Identifier: Apache-2.0
// Package httpauth evaluates source-backed HTTP intent, not application identity
// or evidence of kernel enforcement. Trusted integrations own routing/forwarding.
package httpauth

import (
	"fmt"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Query struct {
	Route  string     `json:"route"`
	Source netip.Addr `json:"source"`
	Host   string     `json:"host"`
	Method string     `json:"method"`
	URI    string     `json:"uri"`
	At     time.Time  `json:"-"`
}

var label = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
var id = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func ValidID(s string) bool { return id.MatchString(s) }
func validMethod(s string) bool {
	switch s {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return true
	}
	return false
}
func validHost(s string) bool {
	if s == "" || len(s) > 253 || strings.ToLower(s) != s {
		return false
	}
	u, e := url.Parse("http://" + s)
	if e != nil || u.Host != s || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return false
	}
	if p := u.Port(); p != "" {
		v, e := strconv.Atoi(p)
		if e != nil || v < 1 || v > 65535 || strconv.Itoa(v) != p {
			return false
		}
	} else if strings.HasSuffix(s, ":") {
		return false
	}
	h := u.Hostname()
	if a, e := netip.ParseAddr(h); e == nil {
		return a.Zone() == "" && !a.Is4In6() && a.String() == h
	}
	for _, l := range strings.Split(h, ".") {
		if !label.MatchString(l) {
			return false
		}
	}
	return true
}
func requestPath(uri string) (string, error) {
	if len(uri) == 0 || len(uri) > 8192 || strings.ContainsAny(uri, "#\r\n\x00") {
		return "", fmt.Errorf("invalid request URI")
	}
	p, _, _ := strings.Cut(uri, "?")
	if !strings.HasPrefix(p, "/") || strings.ContainsAny(p, "%\\") || strings.Contains(p, "//") || path.Clean(p) != strings.TrimSuffix(p, "/") && p != "/" {
		return "", fmt.Errorf("noncanonical path")
	}
	for _, c := range uri {
		if c < 33 || c > 126 {
			return "", fmt.Errorf("non-ASCII/control URI")
		}
	}
	return p, nil
}
func Validate(q Query) error {
	if !ValidID(q.Route) || !q.Source.IsValid() || q.Source.Is4In6() || q.Source.Zone() != "" || !validHost(q.Host) || !validMethod(q.Method) || q.At.IsZero() {
		return fmt.Errorf("invalid authorization context")
	}
	_, e := requestPath(q.URI)
	return e
}
