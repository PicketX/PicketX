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

	"github.com/PicketX/PicketX/internal/strictjson"
)

type principalRecord struct {
	ID     string   `json:"id"`
	SHA256 string   `json:"sha256"`
	Roles  []string `json:"roles"`
}

func loadPrincipals(path string) ([]principalRecord, error) {
	info, e := os.Stat(path)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, ErrUnavailable
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, ErrUnavailable
	}
	defer f.Close()
	info, e = f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, ErrUnavailable
	}
	data, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(data) > 1<<20 {
		return nil, ErrUnavailable
	}
	var document struct {
		Principals *[]principalRecord `json:"principals"`
	}
	if e = strictjson.Decode(data, &document); e != nil || document.Principals == nil || len(*document.Principals) > 128 {
		return nil, ErrUnavailable
	}
	ids, keys := map[string]bool{}, map[string]bool{}
	for _, p := range *document.Principals {
		key, e := hex.DecodeString(p.SHA256)
		canonical := hex.EncodeToString(key)
		if e != nil || len(key) != 32 || !identifier(p.ID, "principal:") || ids[p.ID] || keys[canonical] || len(p.Roles) < 1 || len(p.Roles) > 16 {
			return nil, ErrUnavailable
		}
		ids[p.ID] = true
		keys[canonical] = true
		roles := map[string]bool{}
		for _, role := range p.Roles {
			if (!identifier(role, "role:") && !identifier(role, "group:")) || roles[role] {
				return nil, ErrUnavailable
			}
			roles[role] = true
		}
	}
	return *document.Principals, nil
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
	entries, e := loadPrincipals(a.settings.Credentials)
	if e != nil {
		return Principal{}, e
	}
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
	entries, e := loadPrincipals(a.settings.Credentials)
	if e != nil {
		return Principal{}, e
	}
	for _, entry := range entries {
		digest, _ := hex.DecodeString(entry.SHA256)
		if subtle.ConstantTimeCompare(key, digest) == 1 && entry.ID == id {
			return Principal{ID: entry.ID, Roles: append([]string(nil), entry.Roles...)}, nil
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
