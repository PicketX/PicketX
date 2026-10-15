// SPDX-License-Identifier: Apache-2.0
package networkplan

import "net/netip"

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
