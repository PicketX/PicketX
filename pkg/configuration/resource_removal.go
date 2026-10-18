// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"errors"
	"fmt"
	"slices"
)

var ErrResourceNotFound = errors.New("Resource not found")
var ErrResourceInUse = errors.New("Resource is referenced")

// RemoveResource removes only an unused definition, never its references.
// Revocation, expiry and empty dynamic selections do not erase references.
// The caller must compile/Replace the entire result before accepting it.
func (b *Bundle) RemoveResource(uid string) ([]byte, error) {
	index := slices.IndexFunc(b.objects, func(o document) bool { return o.Kind == "Resource" && o.Metadata.UID == uid })
	if index < 0 {
		return nil, ErrResourceNotFound
	}
	for _, o := range b.objects {
		switch s := o.Spec.(type) {
		case contributionSpec:
			for _, scope := range s.Scopes {
				if scope.ResourceRef.UID == uid {
					return nil, ErrResourceInUse
				}
			}
		case bindingSpec:
			for _, selection := range s.Resources {
				if selection.ResourceRef.UID == uid {
					return nil, ErrResourceInUse
				}
			}
		case httpRouteSpec:
			if s.ResourceRef.UID == uid {
				return nil, ErrResourceInUse
			}
		case resourceSpec, emptySpec:
		default:
			return nil, fmt.Errorf("cannot determine references for %q", o.Kind)
		}
	}
	return b.removeDocument(index)
}
