# M1.3: Core/planner to M0 conformance bridge

> English (default) | [简体中文](2026-10-15-core-m0-bridge-design.zh-CN.md)

- Date: 2026-10-15 (Asia/Shanghai)
- Status: Proposed written design; bridge implementation has not started
- Shared baseline: PicketX/PicketX [PR #7](https://github.com/PicketX/PicketX/pull/7), merged as `e39f96e2787154a8af8e7d996b2f65aa10b6c3eb`; [tested revision](https://github.com/PicketX/PicketX/commit/64b3fd7a7adf6d37905d76f6c57637a69cefb3f1) and [Go/Web CI](https://github.com/PicketX/PicketX/actions/runs/38701569147)
- Agent baseline: PicketX/PicketXD `685ec06afbc6da825eb3a1e3acb54fb539cf9da3`, tree `0fd5e3ea2cd51473d5b961c907ff086d704e2a31`; [M0 final kernel evidence](https://github.com/PicketX/PicketXD/actions/runs/38547987642)
- Authority: [M1.2 design, especially section 7](2026-10-15-physical-network-planner-design.md), [executed M1.2 plan](../plans/2026-10-15-physical-network-planner.md), [product](../../PRODUCT_DESIGN.md), [architecture](../../ARCHITECTURE.md), [configuration model](../../CONFIGURATION_MODEL.md)

## 1. Outcome and selected boundary

The agreed next increment connects the implemented shared logical/physical intent to the existing real-kernel validation infrastructure. Success means that Core decisions, generated protected/source membership and observed new connections agree inside a declared, owned fixture scope, including replacement, independent deadlines and scoped strict revocation. This remains validation infrastructure rather than a production Agent or a browser-to-kernel product workflow.

| Approach | Result and trade-off |
| --- | --- |
| **Isolated conformance bridge (selected)** | Reuse M0 namespaces/backends; prove conversion and lifecycle with one explicit fixture contract before introducing production reconciliation |
| Gitea-only conversion embedded in test bodies | Smaller first diff, but duplicates context/provenance/history handling across lifecycle tests |
| Direct production Agent integration | Adds loading, durable recovery, transport and service lifecycle before the shared-to-kernel boundary is demonstrated |

Implement in the GPL-3.0-or-later PicketXD repository under `internal/validation`. Depend on the Apache PicketX module at immutable revision `e39f96e2787154a8af8e7d996b2f65aa10b6c3eb`; implementation resolves and records the corresponding Go pseudo-version/checksum. Do not commit a floating branch dependency or machine-local replace directive. Keep Go 1.27.0 and the existing pinned Netlink modules unless implementation demonstrates a concrete necessary change. Import only `pkg/core` and `pkg/networkplan`; importing those packages must not bring Controller/Web frameworks into the Agent binary dependency graph. The module graph may still record the shared module's existing requirements; package dependency inspection must distinguish that from linked imports.

Do not copy GPL code into PicketX, promote `internal/validation/fixture` into a public schema, add a Controller client/static manifest decoder, or implement production readiness, persistent approval lineage, durable UIDs, a general scheduler/service, sessions, mTLS or UI. The existing M1.2 mixed-Agent-selector diagnostic Minor stays recorded in PR #7; this bridge uses explicit fixture Agent references and does not silently expand its scope to fix that issue.

## 2. A contract tied to actual topology

The current `kerneltest.Topology` has separate Client, Enforcement and Server namespaces. Both protected destinations and NAT VIPs are assigned to Server, not Enforcement. IPv4/IPv6 forwarding is enabled in Enforcement. The nftables admission chain is PREROUTING at priority -110; the fixture DNAT chain is later at destination-NAT priority. These facts establish one isolated transit test path, not production host-local/container coverage.

| Role | IPv4 | IPv6 |
| --- | --- | --- |
| Primary observed source | `192.0.2.10/32` | `2001:db8:1:0:1::10/128` |
| Independent source control | `192.0.2.11/32` | `2001:db8:1:0:1::11/128` |
| Wider source contribution | `192.0.2.0/26` | `2001:db8:1:0:1::/80` |
| Ordinary destination | `192.0.2.100/32` | `2001:db8:1:0:2::100/128` |
| NAT original VIP | `192.0.2.101/32` | `2001:db8:1:0:2::101/128` |

A fixture scope is created by the trusted topology harness, bound to the actual Enforcement namespace handle/identity, an ephemeral fixture epoch, one Agent reference, one context identity/profile, `transit`, `original-destination`, explicit destination boundary and one `graceful` or `strict` handling mode. One serialized coordinator owns that scope/table/mark; concurrent writers and two authorities sharing it are unsupported. Separate scopes/namespaces represent separate Agents, without a global transaction promise.

Before advertising capabilities, preflight and setup must establish forwarding, fixture links/routes/addresses, the chosen observation stage, and concrete source/destination reachability. Verify protected destinations are not Enforcement-local addresses, bind Netlink handles to the same namespace, and record observed original source/tuple. A label, context string, destination prefix or successful Go conversion alone is no proof of transit isolation. Revalidate fixture identity and required capabilities before every application.

The bridge accepts only that fixture's proven context meaning and physical matches inside its explicit boundary. Reject `host-local`, `published-ingress`, post-DNAT matching, unknown/mismatched Agent/context/network-space identity, unsupported protocol/family/handling, or a wider physical destination set before kernel mutation. Do not strip fields to make a projection fit. Ordinary and NAT fixtures have distinct explicit context identities/boundaries. A running fixture's context meaning and handling mode do not change; mode/placement migration across contexts remains outside this first bridge's accepted capability envelope.

Extend fixture listeners to TCP/443 and TCP/22, preserving existing listeners. The Gitea-shaped case has logical `gitea/https` and `gitea/ssh` with the same physical ports 443 and 22, not an implicit mapping to 8000/8001. Echo payloads validate L4 reachability, not TLS, SSH authentication or Gitea authorization. A separate UDP entry uses UDP/8002. NAT uses explicitly declared original TCP/8000 and TCP/8001 translated by the existing fixture to Server/9000; conversion keeps original VIP/port and does not add a product port-mapping field.

## 3. Conversion, ownership and explicit time

Separate three responsibilities in the private bridge:

1. Candidate validation: construct the first shared plan with `networkplan.Compile`; use `networkplan.Replace` against the fixture's accepted plan for subsequent complete inputs. Check the same authority and exact fixture facts. Validate the entire candidate before any apply/cleanup action. Raw Grant scopes are never reinterpreted in the bridge.
2. Pure materialization: consume the owned `Projection`/`Inventory` at a supplied instant, validate context/capability compatibility and convert supported physical matches and active memberships to private M0 fixture data plus a provenance sidecar.
3. Fixture coordinator: serialize live-clock materialization, apply/confirmation, history and optional strict cleanup. It owns side effects; the converter owns no namespace handles, clock reads, timers or backend calls.

The accepted physical tuple catalog is finite: ordinary fixtures support TCP/443, TCP/22 and UDP/8002 at their declared destinations; NAT fixtures support original TCP/8000 and TCP/8001 at the declared VIPs. Verify the corresponding unfiltered listener/forwarding paths before policy tests. Reject selected ports/tuples outside that proven catalog even if M0 can represent them. Existing extra listeners remain unmanaged controls, not implicit supported product mappings.

Protocol mapping is explicit: `tcp` → 6, `udp` → 17; `PortFirst == PortLast` equals the planner's single port. Retain canonical source/destination prefixes and same-family pairing. Every protected match survives zero active Grants. Active Grant and Deny memberships become separate lists, with deny precedence preserved. Only explicit permanent contributions yield zero M0 expiry. Bounded contributions retain their original absolute ExpiresAt; expired/future/revoked members cannot be installed as active allows or denies.

The sidecar preserves authority, logical entry, context, every Binding ID and separate Grant/Deny ID domains, source/match, validity and revocation state. Equivalent physical rules may deduplicate only with all provenance retained and independent deadline effects preserved; do not union permissions by assigning their latest expiry. No production nftables layout or capacity bound is specified. All prepared/results/history objects own their collections.

Inventory and planned withdrawals are desired state, not evidence of installation. The backend must provide a receipt for the actual membership emitted into an acknowledged batch, and separately identify membership omitted by its expired/submillisecond filtering. Preserve original absolute deadlines and join accepted physical rules back to sidecar provenance. A missing/failed receipt is never fabricated from desired Inventory. A permission observed absent after its legitimate native expiry may still have been installed earlier: successful batch receipt plus owned confirmation is recorded as historical application, rather than erasing that membership.

## 4. Applying, confirming and retaining history

Use the existing owned `inet picketx` table and `nftbackend.TestMark` only inside the bound disposable namespace. Preserve all unrelated tables/marks/flows. A namespace epoch prevents supplying history from another fixture instance even if an Agent/context name repeats.

| Stage | Required behavior |
| --- | --- |
| Prepare | Validate complete replacement/context; no partial output or kernel mutation on failure |
| Materialize | Under the fixture coordinator's serialization, obtain fresh explicit time and active memberships; pass absolute deadlines to the backend |
| Apply | Commit owned admission and, for NAT, its known translation fixture in the same transaction; current M0 rebuilds the owned table |
| Confirm | Require successful batch ACK, owned-table/chain/set readback matching the emitted physical scope/membership, and NAT setup confirmation when applicable; unexpected extra allow state or failed observation is an error |
| Record | Advance the accepted plan and confirmed applied generation before cleanup, and append its actually emitted permissions/protection/provenance to fixture-owned historical records |
| Cleanup | In strict mode, delete only withdrawn historical owned flows against the confirmed new policy, then require confirmation of zero remaining targets |

The current M0 NAT test invokes a separate DNAT flush after Apply. The bridge must instead append the declared fixture translation to the same pending owned-table transaction before its commit, so admission cannot become active with translation missing. Keep the legacy helper for existing M0 tests; a private append-only fixture helper must not flush independently or add arbitrary product policy. Failed preparation must discard its queued batch before any retry, rather than replay stale operations. A failed combined batch preserves the previous admission and translation together.

Readback must account for native expiry, kernel timeout rounding and nonsemantic counters; exact remaining TTL byte equality is not required. A batch receipt is not alone a general concurrent-reconciliation protocol. This confirmation contract relies on exclusive test ownership and must still detect an incorrect physical scope, missing required objects or unexpected extra permission state. Readback failure blocks destructive cleanup.

Keep a cumulative, deduplicated history of confirmed emitted allow membership and protection across all fixture applications. Preserve old absolute deadlines and provenance even after expiry or a later apply omits that permission. Never seed installed history from future/revoked/unbound desired contributions that were never emitted. In this finite test harness retain the necessary historical records until the fixture is destroyed; do not garbage-collect by deadline or silently replace history with the last current projection. Long-running durable history retention/reclamation is a later runtime concern.

Strict cleanup's before side uses this confirmed historical ownership, not an old plan rematerialized at the new time, nor only the last currently active snapshot. Its after side is the confirmed current policy evaluated at the cleanup time, with active deny precedence and all surviving contributions. Deletion retains M0's exact original/reply tuples, zone, family and nonzero mark filter; complete both family dumps before deleting, and confirm remaining targets afterward. Preserve overlapping active Grants, other entries, unrelated/unmarked flows and other marks. No conntrack flush or source-wide deletion is allowed.

A rejected candidate changes no confirmed state. A known kernel transaction rejection retains previous policy; no deletion follows. An ACK/transport/confirmation ambiguity is not proof of rollback: fault the fixture, report unknown application status and perform no revocation or successful handoff. Preserve its evidence, then let harness teardown destroy only its disposable namespaces. This bridge does not recover unknown production state.

Once new policy is confirmed, a later cleanup failure must not roll back it or renew deadlines. Retain historical records, mark cleanup incomplete, preserve the selected/deleted/remaining/error evidence, and allow an explicit retry against the same confirmed ownership. A retry still recomputes time-sensitive membership and confirms the current policy before deletion; it cannot substitute a new wider filter. Missing required history fails exact-cleanup validation.

## 5. Future boundaries, deadlines and empty ownership

The pure converter cannot activate future permissions. Add a cancellable **test-only** boundary driver: use the plan's NextChange, wait with a standard-library timer, then rematerialize/apply at a fresh explicit time without Controller involvement. Check already-passed boundaries immediately after a slow step; do not sleep until a stale or fabricated later deadline. Each wake must read the latest accepted plan under coordinator serialization. Confirmed plan changes replace/cancel the earlier generation’s timer; a stale event may never apply a captured old plan or reopen a revoked permission. Cancellation stops timers and prevents a later apply. This does not introduce a production scheduling service.

At backend dispatch, calculate remaining native timeout from original absolute expiry using a fresh clock inside the backend's serialized application path; do not use a stale preparation timestamp after queue/lock delay, a configured original TTL, or zero for an expired bounded member. Return which rules were actually emitted. Clock/boundary crossing during slow preparation/apply requires a fresh materialization/confirmation step; an expired permission must not be replayed with a fresh full lifetime. Keep the source absolute timestamp unchanged in every receipt/history/replay.

Real packet checks use an explicit **1 second observation window** after an absolute boundary, and record actual first observed denial/activation times. This is a test bound, not a permission extension or a nanosecond enforcement guarantee. Distinguish native-expiry tests with no reapply from boundary-driver tests that must actively install a future allow/deny or clean up strict flows. Late-apply and near-deadline replay tests must detect renewed access; timing misses fail visibly instead of changing the deadline to make a case pass.

Current M0 rejects empty Protected in both nftables Apply and conntrack Revoke. Handle an empty owned desired state explicitly:

- Never call the nonempty Apply path, invent a wildcard/sentinel protected entry, report no-op success or globally flush.
- Delete only the exact owned `inet picketx` table in the bound namespace, await ACK and read back its absence. Confirm absence even for an initially empty trusted fixture. Unrelated tables and foreign conntrack marks remain intact.
- Graceful withdrawal performs no conntrack deletion. Strict withdrawal uses a separate explicit private empty-after cleanup path, validating the nonempty historical before scope and exact ownership mark while accepting the explicitly empty after intent. Do not weaken general fixture.Validate or treat an arbitrary zero snapshot as trusted cleanup authority.
- Confirm zero remaining historical withdrawn owned flows in strict mode. A failure is incomplete cleanup, never success; keep its history/evidence for an explicit retry.

A known but unselected Agent has a valid empty Projection. A removed Agent instead produces an unknown-Agent error from the new shared plan: do not reinterpret that or any arbitrary query error as empty intent. Derive this specific cleanup case only from the validated same-authority complete replacement, explicit absence of the bound Agent in candidate inventory, its former targets in replacement Withdrawals, and the coordinator’s retained fixture identity/history. For retries after an already confirmed empty transition, use that retained explicit empty-state record and pending-cleanup history when there are no newly withdrawn targets. Process the withdrawal/cleanup even though the Agent is absent from the new AgentRefs. All other query/validation errors fail before cleanup.

After unbinding, those entries become unmanaged. Removal of old marked conntrack state does not promise that the service remains blocked or that every old TCP socket is permanently terminated once protection itself is gone. Packet assertions that old ACK/PSH cannot reacquire owned authorization apply while the tuple remains protected; empty-ownership assertions instead verify owned-state absence and preservation of unmanaged controls.

## 6. Conformance cases and evidence

Retain all 17 required M0 scenarios and their existing evidence gate. Add bridge scenarios with distinct required names; the implementation plan fixes exact test names and signatures. Each required kernel case exercises IPv4 and IPv6 or reports its explicitly applicable protocol. Unit tests may use injected effects to verify ordering/failures, but cannot establish packet compatibility.

Verification: `unit` means pure/injected boundary tests; `kernel` means actual privileged packets/state; `both` requires both kinds. Injected failures alone never satisfy the kernel part.

| Case | Required proof  Verification |
| --- | --- | --- |
| Context rejection | Host/published/post-DNAT/wrong namespace/profile/capability/wider scope/unproven tuple rejects before any backend action  `unit` |
| Conversion and provenance | Explicit protocol/single ports, same-family prefixes, all Binding IDs and typed Grant/Deny IDs survive; permanent alone maps to zero expiry  `unit` |
| Invalid or inactive input | Full shared validation; future/revoked/expired permissions never gain active rules; backend submillisecond omissions are present in receipts  `both` |
| Gitea-shaped independent limits | TCP/443 and TCP/22, exact source and wider CIDR retain independently timed permissions; packet results agree with Core for owned tuples  `kernel` |
| Deny and revocation | Active deny wins; removing one contribution preserves matching wider/other grants; unrelated entry/source control continues  `kernel` |
| UDP | Generated UDP/8002 policy accepts authorized packets; withdrawn owned state cannot bypass current permission within protected scope  `kernel` |
| Native expiry | New connections deny after absolute expiry without Controller/reapply, within the recorded observation window; graceful established TCP survives until a later explicit deny/cleanup  `kernel` |
| Future boundaries | Driver activates future allow and deny and removes expired deny; logs boundary/apply/probe times; stale generations cannot replay old policy and cancellation causes no later apply  `both` |
| Replay/late application | Near-expiry backend reconstruction and repeated apply keep original ends; lock/preparation delay does not restart full TTL or turn expiry into permanence  `both` |
| Historical strict cleanup | Prior emitted allow remains in historical before after expiry, an intervening confirmed apply and failed cleanup; retry selects and removes the actual existing withdrawn flow  `both` |
| Strict preservation | Overlapping active grant, other entry, unrelated/unmarked traffic and other ownership mark survive; new withdrawn connections stay denied in protected scope  `kernel` |
| Failed apply/confirmation | No revoker call or false state advancement; real invalid batch preserves previous reachability; ambiguous apply is reported unknown  `both` |
| Interrupted/partial cleanup | No destructive action after incomplete initial dumps; errors/remaining counts persist; confirmed policy remains and explicit retry can finish  `both` |
| Binding ownership | Removing one of two Bindings preserves protection and remaining provenance; last Grant removal keeps protection  `both` |
| Empty owned desired | Last Binding/Agent removal takes explicit scoped table cleanup; strict historical marked flows reach Remaining=0; graceful path skips deletion; unmanaged controls remain usable  `both` |
| NAT original/reply tuples | Admission and translation commit atomically; generated original VIP TCP/8000 withdrawal preserves independently allowed TCP/8001; translated reply Server/9000 is observed and exact filters preserve independent mappings  `kernel` |

The kernel runner must compile and execute the bridge integration binary in the existing disposable-namespace path and parse its events into required evidence. Remove stale bridge results before repeat runs. Missing/duplicate/failed/skipped required bridge cases fail the gate; privilege/setup failure under PICKETX_KERNEL_TESTS=1 fails rather than skips. Keep unit and privileged jobs separate and preserve failure artifacts even when no scenario can run.

Record actual Linux kernel, Go and module versions, immutable shared revision, fixture epoch/context/namespace identity, original tuples, logical expected results, generated physical membership, absolute start/end, apply/confirmation/driver/probe timestamps, and selected/deleted/remaining counts. Old M0 passes do not count as bridge evidence. Pure tests, converter output or a green nonprivileged job do not establish this increment's kernel acceptance. Do not hardcode the old measured kernel as the new result.

## 7. Responsibility map and handoff

All following implementation paths are in **PicketX/PicketXD**, unless stated otherwise. Exact private APIs/TDD tasks belong to the implementation plan after written-design review.

| Area | Files/packages | Responsibility |
| --- | --- | --- |
| Shared dependency | `go.mod`, `go.sum` | Immutable PicketX revision; package dependency/license direction checks |
| Pure bridge | `internal/validation/planbridge` | Scope contract, conversion/provenance, full-candidate/shared replacement reuse; no reimplementation of Core policy |
| Fixture coordinator | `internal/validation/planbridge` | Serialized apply/confirm, actual emitted receipts, cumulative history, test boundary driver and empty-owned lifecycle |
| Backend additions | `internal/validation/nftbackend`, `ctbackend` | Fresh dispatch-time membership, receipts/readback, combined admission/NAT batching, exact owned removal and explicit empty-after strict cleanup; keep existing M0 APIs/tests working |
| Topology and integration | `internal/validation/kerneltest`, bridge integration tests | Bound transit proof, 443/22 listeners, UDP/NAT controls and real connection lifecycle |
| Evidence gate | `internal/validation/report`, `cmd/m0-report`, `scripts/test-kernel.sh`, workflow | Execute new binary, require bridge cases, preserve actual versions/times/failure artifacts |
| Documentation | Agent bilingual READMEs and `docs/validation`; central PicketX design/plan links | Report observed results and remaining runtime envelope without production claims |

The scope is one conformance subsystem with narrowly necessary backend/test additions. No parallel implementation or per-task reviewer is selected: preserve native inline execution with one final fresh whole-branch review when the approved implementation plan is executed. Do not start product code or add the dependency during this design PR. After written design approval, create the bilingual implementation plan and review it before implementation.
