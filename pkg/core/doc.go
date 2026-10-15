// SPDX-License-Identifier: Apache-2.0
// Package core evaluates logical network authorization from an immutable snapshot.
//
// A network source is an IP/CIDR, not a requester, portal session, device identity
// or application account. All active matching Grants contribute permission;
// an active matching Deny wins. Each contribution retains its absolute validity.
// NotIn is a label selection filter, not a deny rule.
//
// Callers supply a nonzero clock and a trusted Resource/Endpoint identity. An
// integration must bind that identity to the actual routed service or packet
// destination. Logical allow is neither physical packet permission nor confirmed
// Agent application. Destination/Binding planning, Linux enforcement, providers,
// approval, persistence, renewal and wire decoding are outside this package.
//
// Compile validates the entire candidate, including inactive or revoked data,
// and builds owned entry memberships. Caller changes cannot alter the engine.
// Compile a replacement separately, then swap it under the caller's own lifecycle
// synchronization. Decide performs read-only evaluation and returns owned slices.
// Core has no wall clock, goroutines, I/O or global decision cache.
//
// Omission is explicit: a completely zero EndpointSelection selects all current
// entries in the referenced Resource; nil Actions defaults to connect. Supplied
// empty names, selectors or Actions fail validation. Existence operators require
// nil Values, not a supplied empty slice. Lifetime mode must explicitly be bounded
// or permanent. Instants normalize to UTC; activation is inclusive and expiry is
// exclusive. NextChange only schedules reevaluation, never renews authorization.
//
// Only source/network/connect and disabled renewal are supported. Unsupported
// subjects, endpoint types, actions, conditions or renewal requirements are errors.
// These evolving Go inputs are not a stable external manifest, JSON or wire schema.
package core
