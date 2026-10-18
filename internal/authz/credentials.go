// SPDX-License-Identifier: Apache-2.0
package authz

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"github.com/PicketX/PicketX/internal/strictjson"
	"github.com/PicketX/PicketX/pkg/httpauth"
	"io"
	"os"
)

type caller struct {
	ID     string   `json:"id"`
	SHA256 string   `json:"sha256"`
	Routes []string `json:"routes"`
}

func load(path string) ([]caller, error) {
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
	var doc struct {
		Callers *[]caller `json:"callers"`
	}
	if e = strictjson.Decode(data, &doc); e != nil || doc.Callers == nil || len(*doc.Callers) > 128 {
		return nil, ErrUnavailable
	}
	ids, keys := map[string]bool{}, map[string]bool{}
	for _, c := range *doc.Callers {
		key, e := hex.DecodeString(c.SHA256)
		canonical := hex.EncodeToString(key)
		if !httpauth.ValidID(c.ID) || e != nil || len(key) != 32 || ids[c.ID] || keys[canonical] || len(c.Routes) < 1 || len(c.Routes) > 128 {
			return nil, ErrUnavailable
		}
		ids[c.ID] = true
		keys[canonical] = true
		routes := map[string]bool{}
		for _, r := range c.Routes {
			if !httpauth.ValidID(r) || routes[r] {
				return nil, ErrUnavailable
			}
			routes[r] = true
		}
	}
	return *doc.Callers, nil
}
func authenticate(path, token string) (caller, error) {
	if len(token) < 32 || len(token) > 256 {
		return caller{}, ErrUnauthorized
	}
	for _, b := range []byte(token) {
		if b < 33 || b > 126 {
			return caller{}, ErrUnauthorized
		}
	}
	entries, e := load(path)
	if e != nil {
		return caller{}, e
	}
	sum := sha256.Sum256([]byte(token))
	var found *caller
	for i := range entries {
		digest, _ := hex.DecodeString(entries[i].SHA256)
		if subtle.ConstantTimeCompare(digest, sum[:]) == 1 {
			found = &entries[i]
		}
	}
	if found == nil {
		return caller{}, ErrUnauthorized
	}
	return *found, nil
}
