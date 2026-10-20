// SPDX-License-Identifier: Apache-2.0
// Package oidclogin verifies external authentication, never authorization roles.
package oidclogin

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/PicketX/PicketX/internal/strictjson"
)

var ErrLogin = errors.New("OIDC sign-in failed")
var ErrUnavailable = errors.New("OIDC sign-in unavailable")

type binding struct {
	Subject     string `json:"subject"`
	PrincipalID string `json:"principal_id"`
	Generation  string `json:"generation"`
}
type bindingDocument struct {
	Bindings *[]binding `json:"bindings"`
}

func readPrivate(path string, target any) error {
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return ErrUnavailable
	}
	f, e := os.Open(path)
	if e != nil {
		return ErrUnavailable
	}
	defer f.Close()
	actual, e := f.Stat()
	if e != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() || actual.Mode().Perm()&0077 != 0 {
		return ErrUnavailable
	}
	raw, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 || strictjson.Decode(raw, target) != nil {
		return ErrUnavailable
	}
	return nil
}
func validSubject(s string) bool {
	if len(s) < 1 || len(s) > 255 {
		return false
	}
	for _, b := range []byte(s) {
		if b < 33 || b > 126 {
			return false
		}
	}
	return true
}
func validPrincipal(s string) bool {
	if !strings.HasPrefix(s, "principal:") || len(s) <= 10 || len(s) > 128 {
		return false
	}
	for _, b := range s[10:] {
		if !(b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '-' || b == '_' || b == '.') {
			return false
		}
	}
	return true
}
func (c *Client) readBindings() ([]binding, error) {
	var doc bindingDocument
	if readPrivate(c.bindings, &doc) != nil || doc.Bindings == nil || len(*doc.Bindings) > 128 {
		return nil, ErrUnavailable
	}
	seen := map[string]bool{}
	for _, b := range *doc.Bindings {
		generation, e := hex.DecodeString(b.Generation)
		if !validSubject(b.Subject) || seen[b.Subject] || !validPrincipal(b.PrincipalID) || e != nil || len(generation) != 16 || hex.EncodeToString(generation) != b.Generation {
			return nil, ErrUnavailable
		}
		seen[b.Subject] = true
	}
	return *doc.Bindings, nil
}

// Resolve rereads the complete authority. A supplied old fingerprint cannot be
// reused after issuer/client/subject/target/generation changes.
func (c *Client) Resolve(subject, fingerprint string) (string, string, error) {
	entries, e := c.readBindings()
	if e != nil {
		return "", "", e
	}
	for _, b := range entries {
		if b.Subject == subject {
			sum := sha256.Sum256([]byte("oidc-binding\x00" + c.issuer + "\x00" + c.oauth.ClientID + "\x00" + b.Subject + "\x00" + b.PrincipalID + "\x00" + b.Generation))
			fp := hex.EncodeToString(sum[:])
			if fingerprint != "" && subtle.ConstantTimeCompare([]byte(fp), []byte(fingerprint)) != 1 {
				return "", "", ErrLogin
			}
			return b.PrincipalID, fp, nil
		}
	}
	return "", "", ErrLogin
}
