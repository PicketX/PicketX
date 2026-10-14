// SPDX-License-Identifier: Apache-2.0
package core

import (
	"net/netip"
	"time"
)

type Resource struct {
	Name      string
	Labels    map[string]string
	Endpoints []Endpoint
}
type Scope struct {
	ResourceName string
	Endpoints    EndpointSelection
	Actions      []string
}
type Grant struct {
	ID          string
	Source      netip.Prefix
	Scopes      []Scope
	Validity    Validity
	Revoked     bool
	SubjectType string
	Conditions  []string
	Renewal     string
}
type Deny Grant
type Snapshot struct {
	Resources []Resource
	Grants    []Grant
	Denies    []Deny
}
type Query struct {
	Source                             netip.Addr
	ResourceName, EndpointName, Action string
	At                                 time.Time
}
type DecisionReason string

const (
	ReasonAllowed       DecisionReason = "allowed"
	ReasonNoActiveGrant DecisionReason = "no-active-grant"
	ReasonExplicitDeny  DecisionReason = "explicit-deny"
)

type Contribution struct {
	GrantID  string
	Source   netip.Prefix
	Validity Validity
}
type Decision struct {
	Allowed           bool
	Reason            DecisionReason
	GrantIDs, DenyIDs []string
	Contributions     []Contribution
	NextChange        time.Time
}
