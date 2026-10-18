// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"errors"
	"slices"
)

var ErrHTTPRouteNotFound = errors.New("HTTPRoute not found")

// RemoveHTTPRoute removes only this request-time integration. It does not revoke
// source Grants or alter network Bindings. Callers must Replace the full candidate.
func (b *Bundle) RemoveHTTPRoute(uid string) ([]byte, error) {
	index := slices.IndexFunc(b.objects, func(o document) bool { return o.Kind == "HTTPRoute" && o.Metadata.UID == uid })
	if index < 0 {
		return nil, ErrHTTPRouteNotFound
	}
	return b.removeDocument(index)
}
