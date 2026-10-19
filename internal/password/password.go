// SPDX-License-Identifier: Apache-2.0
// Package password defines the bounded local-human password format.
package password

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"unicode/utf8"
)

const Cost = 12

func Valid(raw string) bool { return len(raw) >= 12 && len(raw) <= 72 && utf8.ValidString(raw) }
func Hash(raw string) (string, error) {
	if !Valid(raw) {
		return "", errors.New("password requires 12–72 UTF-8 bytes")
	}
	h, e := bcrypt.GenerateFromPassword([]byte(raw), Cost)
	return string(h), e
}
func ValidHash(hash string) bool {
	if len(hash) != 60 || (!strings.HasPrefix(hash, "$2a$12$") && !strings.HasPrefix(hash, "$2b$12$")) {
		return false
	}
	for _, c := range hash[7:] {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '/') {
			return false
		}
	}
	cost, e := bcrypt.Cost([]byte(hash))
	return e == nil && cost == Cost
}
