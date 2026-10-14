# M1.1: Shared network authorization Core

> English (default) | [简体中文](2026-10-15-shared-network-core-design.zh-CN.md)

- Date: 2026-10-15 (Asia/Shanghai)
- Status: Approved design; logical Core implementation and local acceptance complete, awaiting implementation PR integration
- Next: [Executed plan and validation record](../plans/2026-10-15-shared-network-core.md)
- Implementation repository: PicketX/PicketX, Apache-2.0
- Authority: [Architecture](../../ARCHITECTURE.md), [configuration boundaries](../../CONFIGURATION_MODEL.md), [product workflow](../../PRODUCT_DESIGN.md)
- Prerequisite: [PicketXD M0 PR](https://github.com/PicketX/PicketXD/pull/1), merged; [final tested source](https://github.com/PicketX/PicketXD/commit/13fdf2d9aacd79be74578ebe9c23af19a6b33d6a) and [real-kernel CI](https://github.com/PicketX/PicketXD/actions/runs/38547987642)

## 1. Outcome and agreed intent

The next deliverable is one reusable Go authorization engine that explains which named service entries an IP/CIDR source may access at an explicit time. Controller workflows and the official Agent will reuse its semantics. A source is not a requester, portal session, role, device or application identity.

The first product workflow remains Gitea: independent HTTPS and SSH entries, browser request/approval, source-scoped access, visible application status and expiry. This increment proves the shared logical permission semantics needed by that workflow; it does not claim a completed browser-to-kernel workflow.

Already agreed: permission union across matching exact hosts and CIDRs; deny precedence; independent Grant deadlines; IPv4/IPv6 parity; fixed absolute expiry as the first lifetime mode; whole-Resource versus named/label-selected entries; no implicit source-session requirement. This design supplies a narrow internal Go contract without publishing the unresolved YAML/REST/protobuf schemas.

## 2. Increment boundary and approach

| Approach | Consequence |
| --- | --- |
| **Shared Core first (recommended)** | Prove domain behavior with deterministic tests before introducing persistence and transport; each later layer consumes the same engine |
| Build approval/API/storage first | A visible workflow arrives earlier, but network semantics may be repeated across business and Agent code before a common contract exists |
| Build the complete manifest/binding/runtime contract first | Addresses more integration at once, but requires the unresolved identity, lifecycle, placement and transport decisions before the first domain increment |

Use the first approach. Add a small public Go package under `pkg/core` in the existing module `github.com/PicketX/PicketX`. Public Go package names and types are initially evolving implementation APIs, not a stable wire schema. The package uses the standard library, has no HTTP/ORM/Casbin/Linux imports, and owns no goroutines, databases, sockets or wall-clock service.

The first increment includes source normalization, the existing label-selector operators, endpoint selection, explicit lifetime/revocation state, validated network Grant contributions, deterministic decisions and explanations. Resource selection starts with an explicit Resource reference. Broad Resource selectors combined with endpoint names, global ALL encoding, Agent placement and manifest decoding remain separate decisions.

The engine answers a **logical named-entry query**. It does not map packet destinations to logical Resources, compute host/gateway scope, compile nftables, or authorize arbitrary traffic. A later trusted planner must resolve the actual destination/protocol/port and Binding scope. A logical allow alone is not a packet allow or confirmation of enforcement.

## 3. Normalized input model

Names below describe Go-domain roles, not final external field names.

| Input | Required meaning |
| --- | --- |
| Resource catalog | Explicitly named Resources with uniquely named Endpoints, separate Resource/Endpoint labels, and supported endpoint type |
| Network Grant | Unique nonempty contribution ID; one canonical source prefix; nonempty scopes; one shared lifetime; explicit revocation state |
| Scope | An existing Resource reference; exactly one endpoint-selection mode; supported actions |
| Endpoint selection | Whole Resource, nonempty names, or nonempty label selector; constructors distinguish omission from invalid emptiness |
| Network restriction | Independent uniquely identified deny contribution, with source, scopes and lifetime; not an access-request eligibility policy |
| Query | Valid source address, existing Resource and Endpoint, supported action, explicit nonzero evaluation time supplied by a trusted caller |

The initial endpoint type is `network`, with action `connect`. An omitted type/action normalizes to these agreed defaults. Explicit empty action lists are invalid. Unsupported types, subjects, actions, conditions or renewal requirements produce validation errors; they are never discarded or weakened.

Contribution identifiers are opaque internal strings used for deduplication and explanations. This does not choose a persisted UID format, rename/delete/recreate semantics, or a protobuf identifier representation. Grant IDs and deny IDs are separate typed contribution domains; duplicate IDs within either domain fail the whole candidate.

Validate every input, including revoked, expired and currently unmatched contributions. Unknown Resource references and missing named Endpoints fail validation. Invalid candidates produce no compiled engine or partial new decision state. A resource-selector result with no matches must not become whole-Resource access; broad Resource selectors are outside this increment's accepted input.

### Source addresses

A plain IPv4/IPv6 address normalizes to /32 or /128. An explicitly supplied CIDR normalizes to its masked prefix; a caller authorizing an observed host must use the host-address path, never invent a broader CIDR. Reject malformed values, hostnames, zoned IPv6 addresses and IPv4-mapped IPv6 input in the canonical source contract. Typed compiled prefixes must be valid and canonical.

IPv4 prefixes match IPv4 sources only; IPv6 prefixes match IPv6 sources only. Matching considers all applicable source prefixes, not only the longest match. Source and destination are separate roles; this increment does not reinterpret source prefixes as destination scope.

### Endpoint and label selection

Whole-Resource selection includes all current entries and future catalog additions. Named selection remains limited to its explicit names. Label selection follows the current catalog and includes only matching entries. Empty names, empty selectors and combined modes are invalid. Resource and Endpoint labels are not inherited.

The selector implements the agreed `matchLabels` equality and `In`, `NotIn`, `Exists`, `DoesNotExist` operators: AND requirements, alternative `In` values, absent-key matching for `NotIn`, and no values for existence operators. `NotIn` is a selection filter; it cannot cancel another Grant's allow. Core constructor input preserves whether values were supplied, so an explicitly supplied empty list on an existence operator is rejected.

## 4. Lifetime and revocation

Use two explicit normalized modes: bounded and permanent. A zero/default lifetime mode is invalid. Bounded contributions require a nonzero absolute end; permanent contributions must not also carry an end. An optional nonzero start limits activation. A bounded end must be strictly after a supplied start.

Validity is half-open: active from start inclusive until end exclusive. At the exact deadline the contribution is inactive. Normalize instants to UTC without changing their meaning. The engine never reads `time.Now`; callers provide evaluation time. Recompile/restart never replaces an absolute end with `now + original duration`.

Revocation removes only the marked contribution from new decisions. A revoked permanent contribution is inactive too. Different sources, scopes or lifetimes remain separate Grants. Existing-connection graceful/strict handling stays in the execution layer; this engine must not promise TCP reset or conntrack deletion from a logical decision.

This normalized lifetime representation does not finalize the external permanent-authorization syntax, condition language or optional Lease schema. Activity renewal and source-session invalidation are excluded.

## 5. Decision, explanation and state ownership

Compile a validated immutable catalog/contribution snapshot. Copy caller-owned maps and slices; later caller mutations must not change an installed engine. Replacing a snapshot compiles a separate engine before the caller swaps it into use. Concurrent decisions are read-only; no global registry or decision cache is required.

For a query at time `t`:

1. Resolve the existing logical Resource/Endpoint and supported action.
2. Collect every non-revoked, currently valid Grant matching the source and scope.
3. Collect every currently valid matching explicit deny contribution.
4. A deny wins. Otherwise any matching Grant allows; no matching Grant denies.

Explain the outcome with sorted unique Grant IDs, sorted deny IDs, the matched Grant source/lifetime information and a reason (`allowed`, `no-active-grant`, or `explicit-deny`). Preserve each contribution's absolute deadline. Never combine HTTPS and SSH into one source-wide permission or set both to the latest deadline.

Return the earliest future relevant activation/expiry instant as `nextChange`, when present, so the consumer knows when to reevaluate. It is **not** a new authorization deadline, renewal, lease or enforcement-delay guarantee. Including a contribution whose expiry changes provenance while another Grant still allows is valid; reevaluation must preserve the remaining allow.

An invalid query returns an error, not a fabricated allow or an arbitrary whole-Resource match. Explanations contain authorization provenance, not proof of who generated packets. Decisions contain no desired/applied Agent status.

## 6. Determinism and conformance

Use explicit clocks and small literal fixtures; no Linux privilege is required. Sort outputs so input ordering and map iteration cannot change explanations. Produce equivalent decisions from equivalent normalized inputs, whether an eventual Controller or Static provider supplied them. No decoder/provider is implemented in this increment.

| Case | Required result |
| --- | --- |
| IPv4/IPv6 host normalization and explicit CIDR | Correct canonical prefix without silently broadening an observed host |
| Malformed/mapped/zoned address or invalid typed prefix | Validation error |
| Exact-host plus wider-prefix Grant | Both contributors retained; revoking either preserves the other's matching access |
| HTTPS until 10:00, SSH until 12:00 | At 10:00 HTTPS denied and SSH allowed |
| Another HTTPS Grant until 11:00 | HTTPS remains allowed after the first deadline; expired ID no longer contributes |
| Start/end boundaries and permanent/revoked states | Start inclusive, end exclusive; no accidental permanence from zero values |
| Matching active deny | Deny overrides all allow contributions; inactive deny has no effect |
| `NotIn` with absent key, then combined `Exists` | Match, then no match; never an explicit deny |
| Whole/named/label-selected entries after catalog addition | Whole expands, named stays fixed, selector follows labels |
| Duplicate contribution/catalog identity and missing named entries | Whole candidate rejected; no partial broadened result |
| Omitted network/connect defaults versus explicit defaults | Same canonical intent and decisions |
| Empty selection/actions and unsupported condition/type/action | Rejected, never treated as omitted/all |
| Reordered equivalent snapshot and caller mutation after compile | Stable explanations; installed engine unchanged |
| Relevant future start/expiry | Correct earliest `nextChange`, without changing Grant deadlines |
| Unknown Resource/Endpoint and zero evaluation time | Query error |
| Gitea HTTPS/SSH and overlapping source grants | Independent entries, lifetimes, revocation and explanation verified together |

Tests compare hand-derived expected IDs, booleans and instants. Add a small exhaustive/reference check over bounded source/time fixtures if it proves aggregation behavior rather than mirroring the implementation. This is not a 100k-source benchmark or a performance claim.

## 7. Delivery slices and context map

| Slice | Proposed files | Deliverable |
| --- | --- | --- |
| Source, selector, selection and lifetime primitives | `pkg/core/source.go`, `selector.go`, `validity.go` and corresponding tests | Validated canonical inputs with the specified family, presence and time boundaries |
| Immutable Grant/restriction engine | `pkg/core/network.go`, `network_test.go` | Permission union, deny precedence, deterministic explanation and next-change information |
| Cross-case conformance and integration boundary | `pkg/core/conformance_test.go`, `doc.go`, CI format scope and bilingual documentation | Gitea logical cases and Controller/Agent consumption guidance |

Direct dependencies are the Go standard library. The existing Echo liveness handler is unchanged. Existing CI already runs Go race tests; implementation will include `pkg` in the formatting check. No GPL Agent code or fixture implementation is copied into the Apache repository; conformance expectations are independently written from the agreed contract.

After this design is reviewed, write the concrete TDD implementation plan and continue with the already selected inline execution method. Its final review covers the whole branch once. The original design PR contained documentation only; the logical implementation is now recorded in the plan's execution record.

## 8. Remaining M1 sequence

After the shared logical Core passes review and tests, define the normalized physical endpoint/binding planner and its Core-to-M0 conformance bridge. That stage must resolve host/gateway scope, destination matching, ownership conflicts and catalog-edit/lifecycle behavior rather than infer them from logical names.

Then implement the first persistence/approval slice with versioned business migrations, trusted actor/source handling and reliable publication; add Agent transport/apply status and the Gitea portal/CLI workflow incrementally. Use Casbin for the selected management RBAC boundary, not as a replacement for network Grant union. OIDC/SCS, UI, gRPC and local AuthZ endpoints follow their focused implementation designs.

Do not require a generic workflow engine, tenancy platform, arbitrary ABAC language or event-stream framework to deliver this increment.
