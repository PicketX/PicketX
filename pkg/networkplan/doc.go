// SPDX-License-Identifier: Apache-2.0

// Package networkplan compiles Core authorization and explicit deployment
// Bindings into immutable per-Agent physical intent. It uses only the standard
// library and pkg/core; it performs no I/O, clock reads, scheduling or apply.
//
// # Authority and scope
//
// Input is one complete authoritative desired state. AuthorityID distinguishes
// Controller and static publishers; Replace cannot combine their plans. Trusted
// Agent inventory, capability profiles and resolved execution contexts are
// supplied separately from logical permissions. Runtime facts may narrow or
// reject the requested class, stage, handling and destination boundary; broader
// facts fail rather than being clipped under a reused context identity.
// Context/network-space IDs must come from a trusted resolver, not arbitrary
// labels claiming isolation. Declared capabilities do not prove current apply.
//
// Every logical entry needs one explicit TCP/UDP single-port network definition.
// Nil entry Addresses means the resolved context boundary; a supplied non-nil
// empty list is invalid. Context boundaries are always explicit and nonempty.
// ParseDestinations accepts plain hosts/CIDRs and masks explicit CIDRs; typed
// prefixes must already be canonical. CanonicalDestinations merges equivalent
// prefix unions without enumerating addresses. Source and destination roles
// stay separate and cross-family memberships are never synthesized.
//
// Resource selection starts with exact references and reuses Core endpoint
// selection. Broad Resource selectors, DNS, port ranges, NAT64, alternate-port
// mapping and post-DNAT matching are unsupported. Nonempty Requirements fails;
// future wire decoders must also reject unknown fields instead of dropping them.
// Binding entry and Agent selections form a deduplicated cross-product. Missing
// explicit references fail; unmatched selectors diagnose empty scope. Offline
// Agents remain desired targets. Known unselected Agents return empty desired
// inventory; unknown Agents fail. Unbound entries are unmanaged, not implicitly
// permitted. Overlapping claims by different entries fail even without Grants.
// Distinct context IDs do not prove isolation: same-space overlapping contexts
// for an entry are conservatively rejected. Identical targets merge Binding IDs.
//
// # Ownership and explicit time
//
// Compile and Replace validate the whole candidate before returning a plan.
// Plans copy input collections and all reads return independently owned data;
// concurrent reads and independent result mutation are supported. Inventory
// includes future, expired and revoked relevant contributions. Projection uses
// an explicit nonzero time to materialize active Allow/Deny domains. An adapter
// evaluating a flow must apply active deny precedence over the allow union.
// Every contribution retains its independent absolute validity; loading or
// replacing a plan never renews a deadline. NextChange is the earliest relevant
// future activation/expiry, including deny boundaries. The caller must arrange
// actual reevaluation; native expiry alone cannot activate a future rule.
// Protected targets remain when their last Grant expires or is removed.
//
// # Replacements and runtime boundary
//
// Compile is a trusted initial bootstrap, not adoption of unknown installed
// rules. For later complete candidates use Replace with the corresponding prior
// accepted plan. Changes to full entry definitions reject retained Grant/Deny
// IDs in their separate domains, including inactive/unbound membership and
// changes hidden by a current context intersection. A fresh-ID reissuance must
// be authorized by the publisher with original absolute limits retained: there
// is no approval-lineage mapping here to prove that relation automatically.
// Reused context IDs cannot change meaning. Dynamic whole/label selection stays
// dynamic; named references must remain valid or be explicitly removed/reissued.
// Durable UIDs, tombstones and cross-restart identity recovery are not supplied.
//
// Withdrawals preserve full former desired targets; they are not evidence of
// installation or flow deletion lists. Membership-only changes appear in the
// new Inventory without withdrawing protection. A later adapter must compare
// historical installed/observed owned state against confirmed new policy,
// preserve other permissions and unrelated traffic, and apply/confirm before
// scoped strict conntrack cleanup. Rematerializing an old plan at the new time
// loses expired allows that may still own established graceful flows. Missing
// installed history cannot justify wider deletion. Empty desired ownership
// requires explicit owned cleanup and confirmation, never a global flush or
// no-op success; M0 currently rejects empty Protected fixtures.
//
// The separate PicketXD M1.3 bridge must prove its traffic-class guards and
// original-tuple observation, recheck capabilities and implement scheduling and
// cleanup. This package's pure Go tests establish no packet, host/container,
// production gateway or real-kernel compatibility.
package networkplan
