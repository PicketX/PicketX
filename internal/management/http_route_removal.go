// SPDX-License-Identifier: Apache-2.0
package management

import (
	"errors"
	"github.com/PicketX/PicketX/pkg/configuration"
)

var ErrHTTPRouteNotFound = errors.New("HTTPRoute not found")

func (s *Service) RemoveHTTPRoute(p Principal, expected uint64, reason, uid string) (InventoryChange, error) {
	doc, b, err := s.grantBase(p, expected, reason)
	if err != nil {
		return InventoryChange{}, err
	}
	policy, err := b.RemoveHTTPRoute(uid)
	if errors.Is(err, configuration.ErrHTTPRouteNotFound) {
		return InventoryChange{}, ErrHTTPRouteNotFound
	}
	if err != nil {
		return InventoryChange{}, ErrInvalid
	}
	changed, err := s.Replace(p, expected, reason, string(policy), doc.Facts)
	if err != nil {
		return InventoryChange{}, err
	}
	return InventoryChange{uid, changed.Revision}, nil
}
