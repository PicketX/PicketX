// SPDX-License-Identifier: Apache-2.0
package managementaccess

import "crypto/subtle"

// AuthenticateHuman is for an independently verified identity integration only.
// It never accepts a key principal or resolves provider-supplied roles.
func (a *Access) AuthenticateHuman(id string) (Principal, string, error) {
	doc, err := a.credentials()
	if err != nil {
		return Principal{}, "", err
	}
	for _, u := range doc.Users {
		if u.ID == id && !u.Disabled {
			return Principal{ID: u.ID, Roles: append([]string(nil), u.Roles...)}, userFingerprint(u), nil
		}
	}
	return Principal{}, "", ErrUnauthorized
}
func (a *Access) ResolveHumanSession(id, fingerprint string) (Principal, error) {
	p, current, err := a.AuthenticateHuman(id)
	if err != nil {
		return Principal{}, err
	}
	if subtle.ConstantTimeCompare([]byte(current), []byte(fingerprint)) != 1 {
		return Principal{}, ErrUnauthorized
	}
	return p, nil
}
