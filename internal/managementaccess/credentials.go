// SPDX-License-Identifier: Apache-2.0
package managementaccess

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PicketX/PicketX/internal/password"
	"github.com/PicketX/PicketX/internal/strictjson"
)

type principalRecord struct {
	ID     string   `json:"id"`
	SHA256 string   `json:"sha256"`
	Roles  []string `json:"roles"`
}

type userRecord struct {
	ID           string   `json:"id"`
	Username     string   `json:"username"`
	PasswordHash string   `json:"password_hash"`
	Roles        []string `json:"roles"`
	Disabled     bool     `json:"disabled"`
	Version      uint64   `json:"-"`
}
type credentialDocument struct {
	Principals *[]principalRecord `json:"principals"`
	Users      []userRecord       `json:"users"`
}

func loadPrincipals(path string) ([]principalRecord, error) {
	doc, e := loadCredentials(path)
	if e != nil {
		return nil, e
	}
	return *doc.Principals, nil
}
func loadCredentials(path string) (credentialDocument, error) {
	var document credentialDocument
	info, e := os.Stat(path)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return document, ErrUnavailable
	}
	f, e := os.Open(path)
	if e != nil {
		return document, ErrUnavailable
	}
	defer f.Close()
	info, e = f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return document, ErrUnavailable
	}
	data, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(data) > 1<<20 {
		return document, ErrUnavailable
	}
	if e = strictjson.Decode(data, &document); e != nil || document.Principals == nil || len(*document.Principals)+len(document.Users) > 128 {
		return credentialDocument{}, ErrUnavailable
	}
	ids, keys := map[string]bool{}, map[string]bool{}
	for _, p := range *document.Principals {
		key, e := hex.DecodeString(p.SHA256)
		canonical := hex.EncodeToString(key)
		if e != nil || len(key) != 32 || !identifier(p.ID, "principal:") || ids[p.ID] || keys[canonical] || !validRoles(p.Roles) {
			return credentialDocument{}, ErrUnavailable
		}
		ids[p.ID] = true
		keys[canonical] = true
	}
	users := map[string]bool{}
	for _, u := range document.Users {
		if !identifier(u.ID, "principal:") || ids[u.ID] || !validUsername(u.Username) || users[u.Username] || !password.ValidHash(u.PasswordHash) || !validRoles(u.Roles) {
			return credentialDocument{}, ErrUnavailable
		}
		ids[u.ID] = true
		users[u.Username] = true
	}
	return document, nil
}
func validRoles(roles []string) bool {
	if len(roles) < 1 || len(roles) > 16 {
		return false
	}
	seen := map[string]bool{}
	for _, role := range roles {
		if (!identifier(role, "role:") && !identifier(role, "group:")) || seen[role] {
			return false
		}
		seen[role] = true
	}
	return true
}
func validUsername(name string) bool {
	if len(name) < 3 || len(name) > 64 {
		return false
	}
	for i, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || i > 0 && (c == '.' || c == '_' || c == '-')) {
			return false
		}
	}
	return true
}
func identifier(value, prefix string) bool {
	if !strings.HasPrefix(value, prefix) || len(value) <= len(prefix) || len(value) > 128 {
		return false
	}
	for _, c := range value[len(prefix):] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func (a *Access) Authenticate(raw string) (Principal, error) {
	if len(raw) < 32 || len(raw) > 256 {
		return Principal{}, ErrUnauthorized
	}
	for _, c := range []byte(raw) {
		if c < 33 || c > 126 {
			return Principal{}, ErrUnauthorized
		}
	}
	doc, e := a.credentials()
	if e != nil {
		return Principal{}, e
	}
	entries := *doc.Principals
	sum := sha256.Sum256([]byte(raw))
	var found *principalRecord
	for i := range entries {
		digest, _ := hex.DecodeString(entries[i].SHA256)
		if subtle.ConstantTimeCompare(sum[:], digest) == 1 {
			found = &entries[i]
		}
	}
	if found == nil {
		return Principal{}, ErrUnauthorized
	}
	return Principal{ID: found.ID, Roles: append([]string(nil), found.Roles...)}, nil
}

// ResolveSession refreshes trusted membership and binds the session to the
// exact credential used at login, not merely to a reusable principal ID.
func (a *Access) ResolveSession(id, fingerprint string) (Principal, error) {
	key, e := hex.DecodeString(fingerprint)
	if e != nil || len(key) != sha256.Size {
		return Principal{}, ErrUnauthorized
	}
	document, e := a.credentials()
	if e != nil {
		return Principal{}, e
	}
	for _, entry := range *document.Principals {
		digest, _ := hex.DecodeString(entry.SHA256)
		if subtle.ConstantTimeCompare(key, digest) == 1 && entry.ID == id {
			return Principal{ID: entry.ID, Roles: append([]string(nil), entry.Roles...)}, nil
		}
	}
	for _, u := range document.Users {
		digest, _ := hex.DecodeString(userFingerprint(u))
		if !u.Disabled && u.ID == id && subtle.ConstantTimeCompare(key, digest) == 1 {
			return Principal{ID: u.ID, Roles: append([]string(nil), u.Roles...)}, nil
		}
	}
	return Principal{}, ErrUnauthorized
}

// ValidateCredentials checks configuration without disclosing input or Token hashes.
func ValidateCredentials(path string) error {
	if _, e := loadPrincipals(path); e != nil {
		return fmt.Errorf("invalid management credential file")
	}
	return nil
}
