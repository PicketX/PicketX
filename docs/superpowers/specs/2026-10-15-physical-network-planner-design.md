# M1.2: Physical network planning

> English (default) | [简体中文](2026-10-15-physical-network-planner-design.zh-CN.md)

- Date: 2026-10-15 (Asia/Shanghai)
- Status: Approved design ([PR #5](https://github.com/PicketX/PicketX/pull/5)); [plan](../plans/2026-10-15-physical-network-planner.md) approved in PR #6 and executed, locally verified in `pkg/networkplan`; separate M1.3 bridge remains unimplemented
- Repository: PicketX/PicketX, Apache-2.0, existing Go 1.27.0 module
- Authority: [Product](../../PRODUCT_DESIGN.md), [architecture](../../ARCHITECTURE.md), [configuration model](../../CONFIGURATION_MODEL.md), [M1.1 logical Core](2026-10-15-shared-network-core-design.md)
- Baseline: M1.1 [PR #4](https://github.com/PicketX/PicketX/pull/4) merged as `a2143104d8e17a1f6e8552a3ec1d5195d5bbd3cf`; [main CI](https://github.com/PicketX/PicketX/actions/runs/38661859967) succeeds

## 1. Intent and selected approach

The user approved an independent physical planner followed by an M0 bridge. The planner translates logical Resource/Endpoint authorization and explicit deployment Bindings into reproducible per-Agent protection and permission intent. Its success criterion is that independently timed source permissions map to the correct physical traffic space without silently broadening scope or losing provenance.

| Approach | Trade-off |
| --- | --- |
| **Independent planner, then M0 bridge (selected)** | Establish shared, testable physical intent before introducing backend and runtime behavior |
| Fixed Gitea-to-M0 path, then generalization | Earlier packet evidence for one case, followed by extraction of reusable selection and ownership semantics |
| Planner and production Agent together | Earlier integration of lifecycle and application, with a substantially larger validation and failure boundary |

M1.2 delivers the normalized planner in `pkg/networkplan`, standard-library-only, and a narrow owned entry/contribution view from `pkg/core`. M1.3 will implement the separate PicketXD-side M0 conformance bridge after its own focused design. This split preserves the agreed order; M1.2 does not import or promote M0 private fixtures into a public contract.

No product dependencies, HTTP/database/Casbin/Linux imports, goroutines, wall clock, transport, manifest decoder, persistent lifecycle service or UI are added. No GPL Agent implementation is copied into the Apache repository. Internal Go names below describe roles; the implementation plan will fix signatures. This is not a YAML/REST/protobuf schema.

## 2. Normalized inputs and Core reuse

| Input | Meaning and validation |
| --- | --- |
| Logical snapshot | Existing Core Resources, Grant and Deny contributions; existing source/selection/lifetime rules remain authoritative |
| Physical definitions | Exactly one network definition for every logical Endpoint, keyed by exact Resource/Endpoint reference; no orphan or duplicate definitions |
| Agent inventory | Trusted unique Agent references, separate Agent labels and declared capability profiles; offline Agents remain in inventory |
| Execution contexts | Trusted per-Agent network-space identity, traffic class, observation stage and explicit destination boundary; no context is inferred from a name or label |
| Bindings | Unique nonempty Binding IDs, nonempty resource and Agent selection lists, explicit context reference per selected Agent and compatible connection-handling requirements |
| Previous plan | Trusted prior accepted plan from the same authority, used to validate replacement and explain withdrawal; not another source of permission |

The first physical definition requires explicit TCP or UDP and one destination port in 1–65535. There is no protocol default. Port ranges, NAT64, implicit alternate-port mapping, DNS destinations and post-DNAT policy matching are unsupported in this increment; supplied requirements must fail rather than be simplified. M0's ability to test ranges does not automatically create a product field.

Destination prefixes use the same canonical IPv4/IPv6 rules as sources, but remain a separate role. Plain addresses normalize to /32 or /128; explicit CIDRs are masked after rejecting malformed, mapped and zoned addresses. A nonempty destination list denotes a union; nil denotes omission; a supplied non-nil empty list is invalid. Canonicalize each family's union by removing contained duplicates and merging complete sibling prefixes, so equivalent sets compare equally without expanding individual addresses. External null/unknown-field handling remains the future decoder's responsibility and cannot be represented as valid omission by an adapter.

Binding resource items start with exact Resource references and reuse `core.EndpointSelection`: omitted whole-Resource, nonempty names or nonempty labels. Broad Resource selectors are rejected until separately designed. Agent items contain exactly one known reference or nonempty Core label selector; their results are unioned and deduplicated. Selected entries apply to every selected Agent, never positional pairs. A Binding supplies one nonempty context reference, resolved under each selected Agent's own trusted context catalog; context keys are scoped by Agent, and a missing selected-target context fails. Agent selectors with no matches produce no targets plus a diagnostic, not ALL. Explicit missing references fail. Labels are never inherited between Resource, Endpoint and Agent.

Core remains the single authority for Grant/deny validation, resolved logical memberships and lifetimes. Add an owned, deterministic entry/contribution view to the compiled Core, including inactive/future/revoked contributions, and reuse it in the planner. Do not expose Core's internal mutable maps, reconstruct permissions by walking raw Grant scopes, or infer prefix-wide permissions from point queries. Binding endpoint selection reuses existing Core primitives. Preserve all M1.1 behavior and tests.

## 3. Execution scope and address omission

Every selected Agent/Binding pair must resolve to one explicit execution context. The Binding's requested traffic class, observation stage and permitted scope belong to the single authoritative security desired state. Runtime facts and capability profiles may only resolve, narrow or reject that request: they cannot promote host ingress to transit, extend its destination scope or introduce Resources/Grants. Host facts resolve local addresses under the requested host guard; transit requires explicitly authorized scope. A Binding's profile reference and the resolved context identity have separate roles; changed resolved meaning needs a fresh context identity and replacement validation. A context includes:

- Agent identity and an opaque network-space identity supplied by a trusted runtime resolver, not user-chosen strings claiming isolation.
- A stated traffic class: host-local ingress, published-service ingress, or transit. Host installation remains the product default; the normalized context must nevertheless be explicit and resolved. Published container traffic is not automatically host-local traffic.
- The original-destination observation contract for this increment, including the source address observed at that enforcement point. Backend-specific hook/interface predicates belong to the adapter.
- A nonempty canonical destination-boundary union, resolved from explicit configuration/runtime context. A full-family boundary is permitted only when explicitly represented and the traffic-class guard is enforceable; omission never invents one.
- Required capability and connection-handling information. Unknown classes/stages, missing selected-target capability profiles or known unsupported requirements fail candidate validation. This includes unsupported address families, TCP/UDP support and requested graceful/strict handling. An offline target is not removed; a stale advertised profile is not proof of current application, and the adapter rechecks its actual capabilities before apply.

For each selected entry, the physical destination set is the intersection of its explicit addresses with the context boundary. Omitted entry addresses mean the context boundary itself, including its IPv4/IPv6 limits. A selected entry with an empty intersection fails with an Agent/Binding/entry diagnostic; it does not disappear silently. An IPv4-only entry list does not gain IPv6 coverage.

The destination prefixes alone do not establish host-versus-transit isolation. The plan retains the traffic-class/context requirement all the way to application. A backend that cannot enforce it rejects the projection even if the address set is representable. The planner never probes interfaces, opens namespaces, creates routes or discovers container topology.

This distinction is required by actual M0 code: its chain is PREROUTING at priority -110, with explicit destination fixtures; the NAT test uses the original VIP/port and separately verifies translated reply tuples. These tests do not establish production host/container scope. In particular, converting omitted addresses directly into 0.0.0.0/0 and ::/0 in that chain would discard the Binding boundary.

## 4. Plan contents, time and ownership

Compilation produces a fully validated immutable plan. Return no partial plan on any invalid candidate; invalid replacement leaves the installed plan untouched. Copy caller-owned collections and return owned result collections. Equivalent input order/defaults produce the same sorted projections and diagnostics.

A per-Agent projection contains:

1. Protected logical entries and physical destination/protocol/port matches, with execution-context requirements.
2. Every contributing Binding ID. Identical contributions to the same entry/context/physical match may share protection representation without losing ownership.
3. Relevant Grant and Deny provenance, canonical sources, explicit validity and revocation state. Future activations remain available for later reevaluation; cross-family source/destination pairs never become permissions. Grant and Deny ID domains remain distinct as in Core, even when their string values coincide.
4. The complete desired owned-target inventory and deterministic differences against the previous plan, including targets to withdraw. An empty projection is an explicit empty owned desired state, not a failure or a request to flush unrelated objects.

An explicit-time materialization derives active allow/deny membership and the earliest relevant future activation/expiry. Protected entries survive expiry or removal of the last Grant while a Binding still contributes. Removing one Binding preserves other contributing Bindings; removing the last Binding requests withdrawal of that owned protection. A withdrawal request is not confirmation that cleanup succeeded. Differences retain former physical matches, contexts and Binding provenance, not only target IDs; a previous desired plan is not proof of installation, and the adapter uses applied/observed owned inventory for actual cleanup. Entries outside all Bindings remain unmanaged by this projection; neither logical allow nor an empty projection claims permission over unrelated host traffic.

Each contribution keeps its absolute deadline. Never replace it with the source's or Resource's latest deadline. Source-prefix overlap remains a union, active deny wins, and revoking one contribution preserves other matching allows. Revoked, expired and otherwise unused inputs still undergo full validation. Keep future deny activation and deny expiry in scheduling information. Reload/restart and failed replacement never renew a deadline.

Group domain intent by protected entry, context and physical match, retaining contribution membership. This is not a specification of one kernel rule per Grant or of the final nftables set layout. The later backend must aggregate membership without changing addresses, provenance or independent lifetime effects. No capacity or rule-count bound is claimed here.

Graceful and strict withdrawal are distinct explicit requirements. Record the requested behavior and reject incompatible sharing; Core/planner do not perform conntrack deletion, TCP reset or connection observation. Apply confirmation and scoped strict revocation remain Agent responsibilities.

## 5. Physical conflicts

Check physical claims after resolution, regardless of current Grants, expiry, revocation or whether any source currently allows access. Different logical names do not make the same traffic independently enforceable.

For the first version, compare claims within the same trusted Agent/network-space identity using original destination, protocol, port and overlapping destination prefixes. Different context IDs or labels do not prove disjointness. Claims on distinct Agents remain separate targets, with no cross-Agent transaction promise.

- The same logical entry with identical normalized context, traffic match and connection behavior can merge Binding provenance; equivalent address unions are canonicalized before comparison.
- Overlapping claims by different logical entries, including different Endpoints of one Resource, fail the whole candidate. A shared-protection alias model is not introduced implicitly.
- Overlapping claims for the same entry with incompatible contexts or behavior also fail. The initial validator may conservatively reject contexts it cannot prove disjoint; it must report that limitation rather than promise independent isolation.
- Disjoint destination ranges, protocols or ports can coexist. Prefix containment is overlap; testing only equal prefix strings is insufficient.

This supports ordinary host/gateway placements on different Agents and prevents accidental L4 aliasing on one target. More permissive sharing requires its own explicit contract.

## 6. Replacement and lifecycle guard

The first planner uses complete authoritative replacements, not imperative grant/revoke commands. Retain the full normalized entry definitions and resolved contribution memberships, including unbound/inactive entries, for replacement checks; current active projections alone are insufficient. The previous plan is immutable and bound to its provider identity; do not combine Controller and Static state. Runtime must supply/recover the corresponding prior inventory when reconciling an existing installation. A trusted initial bootstrap has no previous plan; this is not permission to adopt unknown existing firewall state.

Within the first increment:

- Adding an entry and changing labels retain the already agreed dynamic whole-Resource/label selection semantics. Named selection stays limited to the named entries. Such edits still require management authorization outside the planner.
- Changes to an existing entry's destination union, protocol or port are retargeting. Default replacement rejects retargeting if an old Grant or Deny contribution ID still resolves to that entry in the candidate. Explicit reissuance with fresh contribution IDs and retained absolute limits is required for this first version; no automatic authorization migration or deadline reset is provided. This includes canonical definition changes even when one current Binding boundary hides the changed region.
- Context identity is immutable: changing network space, traffic class, observation stage, destination boundary or connection behavior under a reused context ID fails. A new explicit context/placement is separately validated, including conflicts and cleanup inventory.
- Missing named references fail. Entry removal/rename is modeled as removal/addition under complete candidate validation; named Grants must be removed or explicitly reissued against existing names. Whole/label selectors retain their deliberate dynamic behavior.
- Durable UID formats, cross-restart tombstones and reuse of deleted names/IDs remain outside this normalized planner. No claim is made that a single fresh snapshot can detect historical deletion/recreation. Production publication/recovery must settle durable identity before enabling those lifecycle operations.

The guard is a conservative deployment rule for this increment, not a replacement for request approval, management RBAC or a general migration engine. If validation fails, report the exact target and conflicting or retained IDs; keep the prior valid plan while its original lifetimes continue. No partial cleanup follows a rejected candidate.

## 7. M0 bridge boundary and later evidence

The bridge is the next increment in PicketXD, depending on a versioned PicketX Core/planner revision. Apache dependencies point toward shared code; they never point into GPL Agent implementation. Current `internal/validation/fixture` stays private.

Before bridge application, declare the exact fixture topology, original-tuple observation contract, supported traffic guard and connection-handling capabilities. A thin converter may accept only projections whose entire context it can prove in that disposable topology. It must reject host/published-service or general gateway requirements the fixture does not prove, rather than strip context fields and claim compatibility.

At an explicit time it translates active contributions and absolute deadlines into private M0 allow/deny/protected fixtures. Future activation requires an actual local reevaluation mechanism in a runtime; native expiry alone cannot activate a future rule. A converter is not an autonomous scheduler. M0's zero expiry means nonexpiring fixture membership and is used only for explicitly permanent contributions, never for omitted/invalid validity.

Strict revocation's before side must come from installed historical owned membership and protection inventory, not from rematerializing the old plan at the new time and discarding expired allows: an expired contribution may still have established graceful connections. If required history is unavailable, report that exact cleanup cannot be proven; do not widen deletion.

M0 currently rejects an empty Protected list. Therefore the converter must not silently turn empty desired ownership into an invalid Apply call, no-op success, global flush or unbounded wildcard. Explicit owned cleanup and its confirmation require separate adapter handling before the bridge claims that lifecycle case.

The real-kernel gate must demonstrate that generated physical intent, logical Core decisions and observed new connections agree, including overlap, independent HTTPS/SSH deadlines, deny, revocation and restart. Strict revocation applies and confirms new policy before deleting only withdrawn owned conntrack flows, and confirms remaining targets. Preserve other Grants, marks and unrelated traffic; never flush conntrack. Source matching uses the actually observed source, and NAT original/reply tuples retain their existing scoped checks. Decision/packet comparisons are restricted to the declared owned protection scope; unrelated traffic remains an explicit unmanaged control case.

M1.2's pure tests do not establish packet compatibility. M1.3 must record actual privileged Linux/toolchain/dependency versions and packet outcomes; skipped or mocked kernel tests are not success. Production host-local, container-published and general gateway/NAT coverage require their own concrete paths and evidence.

## 8. Acceptance and context map

These are approved M1.2 acceptance requirements, not implemented test results:

| Case | Required result |
| --- | --- |
| Explicit TCP/UDP and port; missing protocol/zero port/range request | Valid single-port intent or clear unsupported/invalid error |
| Missing/duplicate/orphan physical definition, invalid inactive contribution | Whole candidate fails; no partial plan |
| Omitted addresses versus non-nil empty/mapped/zoned/DNS input | Scoped boundary versus validation error |
| Boundary 192.0.2.0/24 plus entry 192.0.2.128/25 | Exactly /25; no destination widening |
| IPv4-only list inside dual-stack boundary | IPv4 protection only; no fabricated IPv6/source-family mapping |
| Empty selected-entry intersection or missing context | Diagnostic error, never unrestricted or silently omitted protection |
| Multi-entry/multi-Agent Binding; duplicate Agent matches | Cross-product, deduplicated with all Binding provenance |
| Selector matches no Agent; referenced Agent offline | Empty target diagnostic versus preserved desired target |
| No Binding for an otherwise logically allowed entry | No physical target or claimed application; unrelated traffic stays unmanaged |
| Source host plus wider CIDR; independent HTTPS/SSH limits | Both contributors retained, independent expiry and deny precedence |
| Future allow/deny start, deny expiry, revoked permanent contribution | Correct active materialization and next reevaluation; no revival |
| Same tuple under different logical entries; overlapping prefixes | Conflict even without active Grants; no name-based isolation |
| Identical entry/context via two Bindings; remove one | Shared protection retains the remaining Binding provenance |
| Last Grant expires; last Binding removed | Protection remains versus explicit owned withdrawal inventory |
| Destination/port retarget with retained IDs; context-ID mutation | Replacement rejected; old plan and deadlines unchanged |
| Reordering, caller/result mutation, concurrent plan reads | Stable, owned, read-only results |
| Equivalent normalized Controller/static-style inputs | Equivalent projections without implementing providers/decoders |

Approved responsibility map:

| Area | Proposed files | Responsibility |
| --- | --- | --- |
| Core resolved entry view | `pkg/core/network.go`, `network_types.go`, focused tests | One authoritative resolved contribution view; preserve M1.1 decisions |
| Physical normalization and contexts | `pkg/networkplan/types.go`, `network.go`, `context.go`, tests | Canonical destinations, explicit transport/port, trusted execution boundaries |
| Placement and conflict validation | `pkg/networkplan/binding.go`, `conflict.go`, tests | Cross-product resolution, selection reuse, physical overlap and provenance |
| Immutable plan and replacement | `pkg/networkplan/plan.go`, `transition.go`, tests | Per-Agent intent, time materialization, retarget guards and withdrawal inventory |
| Conformance and documentation | `pkg/networkplan/conformance_test.go`, `example_test.go`, `doc.go`, bilingual READMEs | Literal Gitea cases, public usage and truthful remaining integration scope |

The implementation plan will choose exact type/function signatures and TDD tasks from this map. Existing Go CI already formats `pkg` and runs vet/race/build. New code must keep go.mod/go.sum unchanged. This design PR is documentation only; it does not create these packages, a bridge, a migration service or production Agent.
