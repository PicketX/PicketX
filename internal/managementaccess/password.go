// SPDX-License-Identifier: Apache-2.0
package managementaccess

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/PicketX/PicketX/internal/password"
	"golang.org/x/crypto/bcrypt"
	"strconv"
)

// Valid cost12 encoding used only to equalize missing/disabled-user work. It is
// never an account hash and a matching comparison cannot authenticate a user.
const dummyPasswordHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEe.OYsR/b.njxQnsWCpiX.4HmpAmj53Eyy"

func userFingerprint(u userRecord) string {
	prefix := "password\x00"
	if u.Version > 0 {
		prefix = "database-password\x00" + strconv.FormatUint(u.Version, 10) + "\x00"
	}
	sum := sha256.Sum256([]byte(prefix + u.ID + "\x00" + u.Username + "\x00" + u.PasswordHash))
	return hex.EncodeToString(sum[:])
}
func (a *Access) PasswordLoginEnabled() (bool, error) {
	doc, e := a.credentials()
	if e != nil {
		return false, e
	}
	for _, u := range doc.Users {
		if !u.Disabled {
			return true, nil
		}
	}
	return false, nil
}
func (a *Access) AuthenticatePassword(username, raw string) (Principal, string, error) {
	if !validUsername(username) || !password.Valid(raw) {
		return Principal{}, "", ErrUnauthorized
	}
	doc, e := a.credentials()
	if e != nil {
		return Principal{}, "", e
	}
	select {
	case a.passwordSlots <- struct{}{}:
		defer func() { <-a.passwordSlots }()
	default:
		return Principal{}, "", ErrUnavailable
	}
	var found *userRecord
	hash := dummyPasswordHash
	for i := range doc.Users {
		if doc.Users[i].Username == username && !doc.Users[i].Disabled {
			found = &doc.Users[i]
			hash = found.PasswordHash
		}
	}
	e = bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw))
	if e != nil || found == nil {
		return Principal{}, "", ErrUnauthorized
	}
	fingerprint := userFingerprint(*found)
	// Re-read after expensive verification: rotation/removal during the check
	// cannot create a usable session, and returned roles are current.
	p, e := a.ResolveSession(found.ID, fingerprint)
	if e != nil {
		return Principal{}, "", e
	}
	return p, fingerprint, nil
}
