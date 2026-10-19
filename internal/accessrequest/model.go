// SPDX-License-Identifier: Apache-2.0
// Package accessrequest owns workflow values, separate from standard Grant state.
package accessrequest

import (
	"errors"
	"github.com/PicketX/PicketX/pkg/configuration"
	"time"
)

var ErrNotFound = errors.New("access request not found")
var ErrConflict = errors.New("access request no longer pending or target changed")

type Record struct {
	ID                int64                           `json:"id"`
	UID               string                          `json:"uid"`
	Requester         string                          `json:"requester"`
	Source            string                          `json:"source"`
	Resource          configuration.InventoryResource `json:"resource"`
	EndpointNames     []string                        `json:"endpoint_names"`
	ExpiresAt         time.Time                       `json:"expires_at"`
	Reason            string                          `json:"reason"`
	CreatedAt         time.Time                       `json:"created_at"`
	State             string                          `json:"state"`
	Version           uint64                          `json:"version"`
	Approver          string                          `json:"approver,omitempty"`
	DecisionReason    string                          `json:"decision_reason,omitempty"`
	DecidedAt         *time.Time                      `json:"decided_at,omitempty"`
	ApprovedEndpoints []string                        `json:"approved_endpoints,omitempty"`
	ApprovedExpiresAt *time.Time                      `json:"approved_expires_at,omitempty"`
	GrantUID          string                          `json:"grant_uid,omitempty"`
	Revision          uint64                          `json:"revision,omitempty"`
	// Response-only observation of current desired Grant, not approval history.
	GrantState string `json:"grant_state,omitempty"`
}
type Input struct {
	Source, ResourceName, ResourceUID, Reason string
	EndpointNames                             []string
	DurationSeconds                           int
}
type Decision struct {
	Version       uint64   `json:"version"`
	Action        string   `json:"action"`
	EndpointNames []string `json:"endpoint_names,omitempty"`
	ExpiresAt     string   `json:"expires_at,omitempty"`
	Reason        string   `json:"reason"`
}
type Page struct {
	Requests   []Record  `json:"requests"`
	NextAfter  int64     `json:"next_after"`
	HasMore    bool      `json:"has_more"`
	ObservedAt time.Time `json:"observed_at"`
}
