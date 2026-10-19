// SPDX-License-Identifier: Apache-2.0
// Package account defines human-directory contracts, separate from network Grants.
package account

import (
	"errors"
	"time"
)

var ErrInvalid = errors.New("invalid account change")
var ErrNotFound = errors.New("account not found")

type Account struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Disabled bool     `json:"disabled"`
	Version  uint64   `json:"version"`
}

// Change is an internal command. HTTP adapters own strict request DTOs.
type Change struct {
	Username, Password string
	Roles              []string
	Disabled           bool
	Reason             string
}

type Event struct {
	Actor     string    `json:"actor"`
	AccountID string    `json:"accountId"`
	Operation string    `json:"operation"`
	Reason    string    `json:"reason"`
	Revision  uint64    `json:"revision"`
	At        time.Time `json:"at"`
}

type Page struct {
	Mode       string    `json:"mode"`
	Revision   uint64    `json:"revision"`
	Accounts   []Account `json:"accounts"`
	Events     []Event   `json:"events"`
	ObservedAt time.Time `json:"observedAt"`
}
