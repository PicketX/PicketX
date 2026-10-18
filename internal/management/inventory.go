// SPDX-License-Identifier: Apache-2.0
package management

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
)

type InventoryPage struct {
	Authority  string                            `json:"authority"`
	Revision   uint64                            `json:"revision"`
	ObservedAt time.Time                         `json:"observed_at"`
	Resources  []configuration.InventoryResource `json:"resources"`
	Bindings   []configuration.InventoryBinding  `json:"bindings"`
	Contexts   []configuration.InventoryContext  `json:"contexts"`
}
type InventoryChange struct {
	UID      string `json:"uid"`
	Revision uint64 `json:"revision"`
}

func (s *Service) Inventory(p Principal) (InventoryPage, error) {
	doc, err := s.Snapshot(p)
	if err != nil {
		return InventoryPage{}, err
	}
	out := InventoryPage{Authority: doc.Authority, Revision: doc.Revision, ObservedAt: time.Now().UTC(), Resources: []configuration.InventoryResource{}, Bindings: []configuration.InventoryBinding{}, Contexts: []configuration.InventoryContext{}}
	if doc.Policy == "" {
		return out, nil
	}
	b, err := configuration.Decode([]byte(doc.Policy))
	if err != nil {
		return out, ErrUnavailable
	}
	f, err := configuration.DecodeFacts([]byte(doc.Facts))
	if err != nil {
		return out, ErrUnavailable
	}
	inventory := b.Inventory(f)
	out.Resources = inventory.Resources
	out.Bindings = inventory.Bindings
	out.Contexts = inventory.Contexts
	return out, nil
}
func inventoryUID(prefix string) (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", ErrUnavailable
	}
	return prefix + hex.EncodeToString(data[:]), nil
}
func (s *Service) CreateResource(p Principal, expected uint64, reason, name string, endpoints []configuration.InventoryEndpoint) (InventoryChange, error) {
	doc, b, err := s.grantBase(p, expected, reason)
	if err != nil {
		return InventoryChange{}, err
	}
	uid, err := inventoryUID("resource-")
	if err != nil {
		return InventoryChange{}, err
	}
	policy, err := b.AddResource(name, uid, endpoints)
	if err != nil {
		return InventoryChange{}, ErrInvalid
	}
	changed, err := s.Replace(p, expected, reason, string(policy), doc.Facts)
	if err != nil {
		return InventoryChange{}, err
	}
	return InventoryChange{uid, changed.Revision}, nil
}
func (s *Service) CreateBinding(p Principal, expected uint64, reason string, in configuration.BindingInput) (InventoryChange, error) {
	doc, b, err := s.grantBase(p, expected, reason)
	if err != nil {
		return InventoryChange{}, err
	}
	f, err := configuration.DecodeFacts([]byte(doc.Facts))
	if err != nil {
		return InventoryChange{}, ErrUnavailable
	}
	in.UID, err = inventoryUID("binding-")
	if err != nil {
		return InventoryChange{}, err
	}
	policy, err := b.AddBinding(in, f)
	if err != nil {
		return InventoryChange{}, ErrInvalid
	}
	changed, err := s.Replace(p, expected, reason, string(policy), doc.Facts)
	if err != nil {
		return InventoryChange{}, err
	}
	return InventoryChange{in.UID, changed.Revision}, nil
}
