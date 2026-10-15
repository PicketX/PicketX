// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"github.com/PicketX/PicketX/pkg/core"
	"net/netip"
	"time"
)

type EntryRef struct{ ResourceName, EndpointName string }
type Protocol string

const (
	TCP Protocol = "tcp"
	UDP Protocol = "udp"
)

type TrafficClass string

const (
	HostLocal        TrafficClass = "host-local"
	PublishedIngress TrafficClass = "published-ingress"
	Transit          TrafficClass = "transit"
)

type ObservationStage string

const OriginalDestination ObservationStage = "original-destination"

type HandlingMode string

const (
	Graceful HandlingMode = "graceful"
	Strict   HandlingMode = "strict"
)

type Family uint8

const (
	IPv4 Family = 4
	IPv6 Family = 6
)

type NetworkDefinition struct {
	Entry     EntryRef
	Protocol  Protocol
	Port      uint16
	Addresses []netip.Prefix
}
type ContextRequest struct {
	ProfileRef          string
	Class               TrafficClass
	Stage               ObservationStage
	AllowedDestinations []netip.Prefix
	Handling            HandlingMode
}
type ExecutionContext struct {
	AgentRef, ProfileRef, ID, NetworkSpaceID string
	Class                                    TrafficClass
	Stage                                    ObservationStage
	Destinations                             []netip.Prefix
	Handling                                 HandlingMode
}
type CapabilityProfile struct {
	Classes             []TrafficClass
	Protocols           []Protocol
	Families            []Family
	Handling            []HandlingMode
	OriginalDestination bool
}

type ResourceSelection struct {
	ResourceName     string
	ResourceSelector *core.LabelSelector
	Endpoints        core.EndpointSelection
}
type AgentSelection struct {
	AgentRef string
	Selector *core.LabelSelector
}
type Binding struct {
	ID        string
	Resources []ResourceSelection
	Agents    []AgentSelection
	Context   ContextRequest
}
type Agent struct {
	Name         string
	Labels       map[string]string
	Offline      bool
	Capabilities *CapabilityProfile
}
type Input struct {
	AuthorityID  string
	Logical      core.Snapshot
	Networks     []NetworkDefinition
	Agents       []Agent
	Contexts     []ExecutionContext
	Bindings     []Binding
	Requirements []string
}
type Diagnostic struct{ Code, BindingID, ResourceName, AgentRef, EndpointName string }
type TargetKey struct {
	AgentRef, ContextID string
	Entry               EntryRef
}
type Match struct {
	Destination netip.Prefix
	Protocol    Protocol
	Port        uint16
}
type Protection struct {
	Key        TargetKey
	Context    ExecutionContext
	BindingIDs []string
	Matches    []Match
	Required   CapabilityProfile
}
type TargetInventory struct {
	Protection     Protection
	Grants, Denies []core.ResolvedContribution
}
type Membership struct {
	Target       TargetKey
	Match        Match
	Contribution core.ResolvedContribution
}
type Projection struct {
	AuthorityID, AgentRef string
	At                    time.Time
	Protected             []Protection
	Allow, Deny           []Membership
	NextChange            time.Time
}

// Withdrawal requests reconsideration of a former desired target. Former is not
// proof of installation, nor a flow deletion list.
type Withdrawal struct {
	Former TargetInventory
	Reason string
}
type Replacement struct {
	Plan        *Plan
	Withdrawals []Withdrawal
}
