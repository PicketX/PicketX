# M1.3 Core/Planner to M0 Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Preserve native inline execution and one fresh whole-branch review at the end.

> English (default) | [简体中文](2026-10-15-core-m0-bridge.zh-CN.md)

**Goal:** Connect shared plans to disposable PicketXD fixtures and prove deadline, replacement and precise-revocation behavior with real dual-stack packets.

**Architecture:** A pure converter handles explicit time and provenance. A serialized coordinator owns confirmation, history and the test boundary driver; backend additions provide actual-emission receipts, atomic admission/DNAT and exact owned cleanup.

**Tech Stack:** Go 1.27.0; existing pinned Go Netlink modules; standard-library clocks/timers; `github.com/PicketX/PicketX/pkg/core` and `pkg/networkplan`.

**Spec:** [M1.3 approved design](../specs/2026-10-15-core-m0-bridge-design.md), [PR #8](https://github.com/PicketX/PicketX/pull/8), merged as `9d53b13e2d8a805b2a7b2f3a8ad957fcf84ea12d`.

**Status:** Historical planning baseline, approved through continued implementation authorization. M1.3 was delivered in the focused [compiler increment](https://github.com/PicketX/PicketXD/pull/2) and [confirmed lifecycle increment](https://github.com/PicketX/PicketXD/pull/3), both now merged. The Agent repository's executed plans and validation records describe the final APIs, 19 additional bridge scenarios and official dependency fixes; unchecked steps below are the original plan, not a current backlog. [M2B](https://github.com/PicketX/PicketXD/pull/4) is also merged, with all 43 required kernel scenarios passing. Production enforcement, L7 and management remain subsequent work.

## Global Constraints

- Shared dependency: immutable `e39f96e2787154a8af8e7d996b2f65aa10b6c3eb`; resolve its pseudo-version/checksum, no floating branch or committed local `replace`.
- Go 1.27.0; GPL-3.0-or-later PicketXD may import Apache Core/planner, never the reverse. Import only `pkg/core` and `pkg/networkplan`; inspect linked package dependencies independently of module graph.
- Only proven `transit`, `original-destination`, one fixed graceful/strict mode and one bound namespace/epoch/authority per coordinator; never widen fixture scope.
- Ordinary catalog: TCP/443, TCP/22, UDP/8002. NAT catalog: original TCP/8000 and TCP/8001 -> Server/9000; separate explicit context. Equal IPv4/IPv6 coverage.
- Only owned `inet picketx` and nonzero `nftbackend.TestMark`; no global conntrack flush, sentinel protection or relaxed general fixture validation.
- Absolute deadlines survive conversion/replay; only explicit permanent means zero expiry. Boundary observation window: **1 second**, never an authorization extension.
- Full candidate -> live-time materialization -> apply/ACK/readback -> record -> optional precise cleanup. Unknown application blocks cleanup and future applies; cleanup failure retains confirmed policy and historical evidence.
- Retain all 17 M0 required cases; add the 16 bridge kernel cases listed below. Opted-in privilege/setup failures fail, never skip. Unit checks are not kernel evidence.
- This increment is private conformance infrastructure: no production provider/transport/LKG/schema/UI or TLS/SSH authentication claim; retain the M1.2 selector-diagnostic Minor.

## Review Focus

| Input/failure | Expected behavior | Owning unit test |
| --- | --- | --- |
| Namespace reuse with same Agent/context name | Reject foreign epoch before effects | `TestBridgeCoordinatorIdentityAndOwnership` |
| Rule expires between ACK and readback | Retain actual emitted history, tolerate only legitimate membership expiry | `TestBridgeCoordinatorHistoryAcrossExpiry` |
| Append preparation fails before flush | Discard pending batch before retry | `TestConfirmedApplyBatchIsolation` |
| Removed Agent vs arbitrary lookup error | Explicit validated empty proof only | `TestBridgeCoordinatorEmptyAndRemovedAgent` |
| Slow application crosses another boundary | Immediate fresh refresh, stale timer cannot replay | `TestBridgeDriverStaleCancellationAndDelay` |

## Context, file and dependency map

All implementation paths below are relative to **PicketX/PicketXD**. Central plans/roadmap and links remain in **PicketX/PicketX**. The map is reviewed together with this plan.

| Task | Files to create/modify | Responsibility |
| --- | --- | --- |
| 1 | `go.mod`, `go.sum`; create `internal/validation/planbridge/{types,prepare,materialize}.go` and corresponding `_test.go` | Immutable shared dependency, scope and pure conversion |
| 2 | modify `internal/validation/kerneltest/{topology_linux,nat_linux}.go`; create `bridge_linux.go`, `bridge_integration_test.go` | Actual namespace identity, tuple proof, listener/connection cleanup, append-only NAT |
| 3 | modify `internal/validation/nftbackend/backend_linux.go`; create `confirmed_linux.go`, `readback_linux.go`, `confirmed_linux_test.go` | Dispatch-time expiry, receipts, atomic fixture batch, confirmation, owned-table removal |
| 4 | modify `internal/validation/ctbackend/{backend_linux,select_linux}.go`; create `empty_linux_test.go` | Explicit empty-after precise revocation |
| 5 | create `internal/validation/planbridge/{coordinator,history,driver}.go` and corresponding `_test.go` | Serialized apply state machine, cumulative history, explicit retry and cancelable driver |
| 6 | create `internal/validation/planbridge/{conformance_integration_test,helpers_integration_test}.go` | Literal Gitea/UDP/NAT and lifecycle packet cases |
| 7 | modify `internal/validation/report/{report.go,report_test.go}`, `cmd/m0-report/main.go`, `scripts/test-kernel.sh`, `.github/workflows/validation.yml`, paired Agent READMEs; create paired `docs/validation/m13-core-bridge.md` | Required evidence and truthful delivery report |

Dependencies: `planbridge` imports Core/planner, fixture and receipt/result types; `kerneltest` never imports `planbridge`. External integration package `planbridge_test` binds topology, nftbackend and ctbackend into Effects. Backend code never imports Core or Controller. Existing M0 `scenario.ApplyStrict`, `Backend.Apply` and `Backend.Revoke` continue to work. Reference patterns: `fixture.Validate`, `ctbackend.exactFilter`, `kerneltest.WithTopology`, existing integration build tags and `report.CheckRequired`.

Risk assessment: no public wire API, database migration or production runtime configuration changes. Private backend APIs grow; existing M0 tests must remain green. Real privileged Linux is required for acceptance. Native execution order: **1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7**; one focused commit and verification per task, one independent whole-branch review after all tasks.

### Task 1: Pure scope, full-candidate validation and materialization

**Files:** Task 1 paths above; `types_test.go`, `prepare_test.go`, `materialize_test.go`.

**Interfaces consumed:** existing `networkplan.Compile(Input)`, `Replace(*Plan, Input)`, `Inventory(string)`, `Projection(string,time.Time)` and Core validity.

**Interfaces produced:**

```go
type NamespaceID struct { Device, Inode uint64 }
type Scope struct {
    AuthorityID, AgentRef, Epoch string
    Namespace NamespaceID
    Context networkplan.ExecutionContext
    Catalog []networkplan.Match
}
type Origin struct {
    AuthorityID string
    Target networkplan.TargetKey
    BindingIDs []string
    Domain string // "grant" or "deny"
    Contribution core.ResolvedContribution
    Match networkplan.Match
}
type Member struct { Rule fixture.Rule; Origins []Origin }
type Materialized struct {
    Snapshot fixture.Snapshot
    Allow, Deny []Member
    Inventory []networkplan.TargetInventory
    At, NextChange time.Time
}
type Candidate struct { /* private immutable plan, withdrawals, scope and explicit empty proof */ }
func Prepare(previous *networkplan.Plan, input networkplan.Input, scope Scope) (*Candidate, error)
func Materialize(candidate *Candidate, at time.Time) (Materialized, error)
```

- [ ] **1. Write failing tests**; build literal fixtures in this package. The following are assertion excerpts with case-local variables. Fix t0=`2026-10-15T10:00:00Z`, HTTPS end=t0+2s, SSH end=t0+4s and future start=t0+1s; pure unit cases use fake clocks:

```go
// TestBridgePrepareRejectsUnsupportedScope: err != nil; candidate == nil; backendCalls == 0 for host-local, published-ingress, post-DNAT, mismatched Agent/context/profile/namespace, unsupported family/protocol/mode, wider destination, TCP/444 and NAT TCP/443.
if err == nil || candidate != nil || backendCalls != 0 { t.Fatal(candidate, err, backendCalls) }
// TestBridgePrepareUsesFullReplacement: authority switch and retarget with retained Grant/Deny IDs fail even when unbound; previous plan remains unchanged.
if err == nil || !reflect.DeepEqual(previousInventory, afterInventory) { t.Fatal(err, afterInventory) }
// TestBridgeMaterializeProvenanceAndDeadlines: TCP/443 and TCP/22 map to protocol 6, UDP/8002 to 17; all Binding IDs and separate grant/deny origins survive; exact end is inactive; only permanent has zero expiry.
if https.Protocol != 6 || https.PortFirst != 443 || https.PortLast != 443 || ssh.PortFirst != 22 || udp.Protocol != 17 || udp.PortFirst != 8002 { t.Fatal(got) }
if !bounded.ExpiresAt.Equal(end) || !permanent.ExpiresAt.IsZero() || len(bindingIDs) != 2 || grantOrigin.Domain != "grant" || denyOrigin.Domain != "deny" { t.Fatal(got) }
// TestBridgeMaterializeOwnsData: mutating input/scope/returned origins, slices and inventory leaves a fresh materialization unchanged; concurrent reads pass race.
if !reflect.DeepEqual(before, fresh) { t.Fatal(fresh) }
// TestBridgeMaterializeInactiveAndEmpty: future/revoked/expired members absent; zero active Grants retains Protected; known unselected Agent is explicitly empty; arbitrary query error fails.
if len(got.Snapshot.Allow) != 0 || len(got.Snapshot.Deny) != 0 || len(got.Snapshot.Protected) == 0 { t.Fatal(got) }
```

- [ ] **2. Verify red.** Run `go test ./internal/validation/planbridge -run "TestBridge(Prepare|Materialize)" -count=1`. Expect missing API/behavior to fail; for integration cases that already pass, record the baseline and add a literal counterexample instead of breaking working code artificially.

- [ ] **3. Implement.** Prepare uses Compile once and Replace thereafter; owns copies of scope/catalog and retains explicit removed-Agent proof from validated input plus withdrawals. Materialize uses projection and complete inventory at the caller time, preserves inactive inventory for provenance but emits active members only. Domain-qualified origins keep Grant/Deny IDs separate. Do not merge independent deadlines; simple one-contribution membership is sufficient. Validate every selected physical tuple and context against exact Scope; scope metadata alone never proves topology. Empty materialization carries an explicit candidate proof, not an arbitrary zero snapshot.

- [ ] **4. Verify green.** Run `go test ./internal/validation/planbridge -run "TestBridge(Prepare|Materialize)" -count=1`; require PASS for every named test, no unexpected skip. Keep prior tasks and M0 regression tests passing.

- [ ] **5. Commit.** Stage only the task files and commit as `feat(validation): materialize shared network plans`. Preserve failure/green evidence in the final validation record.

### Task 2: Proven topology and atomic NAT append seam

**Files:** Task 2 paths above; preserve existing 8000/8001/8002/9000 listeners.

**Interfaces consumed:** existing WithTopology, explicit namespace handles, fixture links/routes and packet helpers.

**Interfaces produced:**

```go
// In kerneltest; no planbridge import, so integration tests can join both packages.
type BridgeIdentity struct { Device, Inode uint64; Epoch string }
func (x *Topology) BridgeIdentity() (BridgeIdentity, error)
func (x *Topology) VerifyBridge(ctx context.Context, id BridgeIdentity, nat bool) error
func AppendDNAT(c *nftables.Conn, table *nftables.Table) error
```

- [ ] **1. Write failing tests**; build literal fixtures in this package. The following are assertion excerpts with case-local variables. Fix t0=`2026-10-15T10:00:00Z`, HTTPS end=t0+2s, SSH end=t0+4s and future start=t0+1s; pure unit cases use fake clocks:

```go
// TestBridgeTopologyIdentity: same live namespace yields stable device/inode and epoch; new topology has a new epoch; closed/replaced handle rejects.
if id.Device == 0 || id.Inode == 0 || id.Epoch == "" || id != again || id.Epoch == other.Epoch { t.Fatal(id, again, other) }
// TestBridgeTopologyTupleCatalog: unfiltered IPv4 and IPv6 reach TCP/443, TCP/22 and UDP/8002; NAT 8000/8001 reaches Server/9000; protected destinations are not Enforcement-local.
if err != nil || ipv4Probes != 3 || ipv6Probes != 3 || destinationIsEnforcementLocal { t.Fatal(err) }
```

- [ ] **2. Verify red.** Run `PICKETX_KERNEL_TESTS=1 go test -tags=integration ./internal/validation/kerneltest -run "TestBridgeTopology" -count=1`. Expect missing API/behavior to fail; for integration cases that already pass, record the baseline and add a literal counterexample instead of breaking working code artificially.

- [ ] **3. Implement.** BridgeIdentity records fstat device/inode and a per-topology random epoch. VerifyBridge checks handle identity, forwarding, route/address facts and ownership before each apply. Prove exact unfiltered catalog on fixture initialization; per-apply verification checks identity/capabilities without treating policy denial as broken reachability. Add TCP/443 and TCP/22 echo listeners and close accepted TCP connections as well as listeners so repeat tests release namespace resources. AppendDNAT queues only the existing fixed VIP/8000-8001 -> Server/9000 fixture translation and never flushes. Keep legacy DNAT as append then Flush. No scope-derived arbitrary NAT policy.

- [ ] **4. Verify green.** Run `PICKETX_KERNEL_TESTS=1 go test -tags=integration ./internal/validation/kerneltest -run "TestBridgeTopology" -count=1`; require PASS for every named test, no unexpected skip. Keep prior tasks and M0 regression tests passing.

- [ ] **5. Commit.** Stage only the task files and commit as `test(validation): prove bridge namespace and tuple scope`. Preserve failure/green evidence in the final validation record.

### Task 3: Confirmed actual emission and combined owned transaction

**Files:** Task 3 paths above; extend existing nftbackend integration tests for atomic rejected batch.

**Interfaces consumed:** fixture.Validate, existing nftables membership construction, Task 2 AppendDNAT through trusted integration assembly.

**Interfaces produced:**

```go
// In nftbackend; preserve New, Apply and Close for existing M0 callers.
type ApplyState string
const (
    ApplyRejected ApplyState = "rejected"
    ApplyUnknown ApplyState = "unknown"
    ApplyConfirmed ApplyState = "confirmed"
)
type Omission struct { Domain string; Rule fixture.Rule; Reason string }
type Receipt struct {
    State ApplyState
    DispatchedAt, ConfirmedAt time.Time
    Emitted fixture.Snapshot
    Omitted []Omission
    OwnedAbsent bool
}
type ConfirmOptions struct {
    Now func() time.Time
    AppendFixture func(*nftables.Conn, *nftables.Table) error
}
func (b *Backend) ApplyConfirmed(ctx context.Context, s fixture.Snapshot, options ConfirmOptions) (Receipt, error)
func (b *Backend) RemoveOwnedConfirmed(ctx context.Context, now func() time.Time) (Receipt, error)
```

- [ ] **1. Write failing tests**; build literal fixtures in this package. The following are assertion excerpts with case-local variables. Fix t0=`2026-10-15T10:00:00Z`, HTTPS end=t0+2s, SSH end=t0+4s and future start=t0+1s; pure unit cases use fake clocks:

```go
// TestConfirmedApplyFreshClockAndOmissions: clock read after backend lock; expired and <1ms remainder omitted with reason; bounded emitted rule retains original end; no zero-time conversion.
if len(receipt.Emitted.Allow) != 1 || len(receipt.Omitted) != 2 || !receipt.Emitted.Allow[0].ExpiresAt.Equal(end) { t.Fatal(receipt) }
// TestConfirmedApplyReadback: wrong family/prefix/port, missing required chain/set, extra allow/deny/member or wrong NAT fails; expected native-expired member may disappear; no required structural object may disappear.
if err == nil || receipt.State == ApplyConfirmed { t.Fatal(receipt, err) }
// TestConfirmedApplyBatchIsolation: append preparation error discards pending operations; retry does not replay them; known rejected transaction returns rejected; ambiguous send/readback returns unknown.
if residualQueuedOperations != 0 || retryContainsFailedAppend { t.Fatal(residualQueuedOperations) }
// TestRemoveOwnedConfirmed: delete only inet picketx, confirm absence even initially empty, preserve foreign tables; missing ACK/readback never confirms.
if receipt.State != ApplyConfirmed || !receipt.OwnedAbsent || !foreignTableExists { t.Fatal(receipt) }
```

- [ ] **2. Verify red.** Run `go test ./internal/validation/nftbackend -run "Test(Confirmed|RemoveOwned)" -count=1`. Expect missing API/behavior to fail; for integration cases that already pass, record the baseline and add a literal counterexample instead of breaking working code artificially.

- [ ] **3. Implement.** ApplyConfirmed validates nonempty input and owns copied results. After acquiring its lock and completing potentially blocking setup, use fresh Now to rebuild absolute-deadline membership immediately before queue/flush. If preparation crosses an expiry, discard and rebuild; if a post-send boundary is crossed the coordinator refreshes. Use one transaction for owned table, admission and optional validated fixed-fixture DNAT append. Track actual emitted rules and expired/submillisecond omissions; success requires ACK plus normalized semantic readback of every owned rule/object/member and NAT expression. Native expiry may remove only expected expired membership, not missing structural state or extra permissions; forbid wider/longer-than-expected timeouts while allowing kernel rounding. Readback observes current state independently, not merely the queued receipt. Rejected requires proof of whole-transaction refusal; uncertain transport/ACK/readback is unknown. Drop all pending messages on preflush errors. RemoveOwnedConfirmed handles only the exact owned table and verifies absence, including initial empty state. The optional append callback is harness-owned, never accepted from a security manifest.

- [ ] **4. Verify green.** Run `go test ./internal/validation/nftbackend -run "Test(Confirmed|RemoveOwned)" -count=1`; require PASS for every named test, no unexpected skip. Keep prior tasks and M0 regression tests passing.

- [ ] **5. Commit.** Stage only the task files and commit as `feat(validation): confirm emitted atomic nftables batches`. Preserve failure/green evidence in the final validation record.

### Task 4: Explicit empty-after strict revocation

**Files:** Task 4 paths above; preserve standard fixture validation.

**Interfaces consumed:** existing Result, exactFilter and SelectWithdrawn(original/reply, zone, family, mark).

**Interfaces produced:**

```go
// In ctbackend; Revoke remains unchanged for existing M0 callers.
func (b *Backend) RevokeEmpty(ctx context.Context, before fixture.Snapshot, mark uint32, now time.Time) (Result, error)
```

- [ ] **1. Write failing tests**; build literal fixtures in this package. The following are assertion excerpts with case-local variables. Fix t0=`2026-10-15T10:00:00Z`, HTTPS end=t0+2s, SSH end=t0+4s and future start=t0+1s; pure unit cases use fake clocks:

```go
// TestRevokeEmptyValidatesOwnership: empty history or zero mark fails before dump/delete; fixture.Validate(empty) still fails; complete both family dumps before any deletion.
if err == nil || dumpCalls != 0 || deleteCalls != 0 { t.Fatal(err, dumpCalls, deleteCalls) }
// TestRevokeEmptyPreciseAndPartial: match original/reply, zone, family, mark; preserve foreign/unmarked/nonhistorical flows; interrupted initial dump deletes zero; partial deletion returns selected/deleted/remaining plus error.
if err == nil || result.Selected != 2 || result.Deleted != 1 || result.Remaining != 1 { t.Fatal(result, err) }
```

- [ ] **2. Verify red.** Run `go test ./internal/validation/ctbackend -run "TestRevokeEmpty" -count=1`. Expect missing API/behavior to fail; for integration cases that already pass, record the baseline and add a literal counterexample instead of breaking working code artificially.

- [ ] **3. Implement.** Refactor an internal shared deletion worker without relaxing Revoke. RevokeEmpty validates nonempty historical before, canonical historical source/endpoint membership and nonzero ownership mark; explicitly constructs private empty after. Both family dumps finish before deletion; every result, including failures, retains counters with Remaining=-1 until confirmed. The coordinator calls this only after confirmed owned-table absence plus validated empty intent. No broad source deletion or conntrack flush.

- [ ] **4. Verify green.** Run `go test ./internal/validation/ctbackend -run "TestRevokeEmpty" -count=1`; require PASS for every named test, no unexpected skip. Keep prior tasks and M0 regression tests passing.

- [ ] **5. Commit.** Stage only the task files and commit as `feat(validation): revoke explicit empty owned state`. Preserve failure/green evidence in the final validation record.

### Task 5: Serial coordinator, cumulative history and boundary driver

**Files:** Task 5 paths above; split files by state transition, history and driver responsibilities.

**Interfaces consumed:** Tasks 1/3/4; Effects implementation remains in integration tests and binds the same namespace handles, clock and fixed append helper.

**Interfaces produced:**

```go
// In planbridge; effects are bound by the trusted topology adapter, not by input manifests.
type Timer interface { C() <-chan time.Time; Stop() bool }
type Clock interface { Now() time.Time; NewTimer(time.Duration) Timer }
type Effects interface {
    Verify(context.Context, Scope) error
    Apply(context.Context, fixture.Snapshot) (nftbackend.Receipt, error)
    Remove(context.Context) (nftbackend.Receipt, error)
    Revoke(context.Context, fixture.Snapshot, fixture.Snapshot, uint32, time.Time) (ctbackend.Result, error)
    RevokeEmpty(context.Context, fixture.Snapshot, uint32, time.Time) (ctbackend.Result, error)
}
type AppliedMember struct { Rule fixture.Rule; Origins []Origin }
type State struct {
    Generation uint64
    Status string // "initial", "confirmed", "cleanup-pending", "unknown"
    Receipt nftbackend.Receipt
    History []AppliedMember
    Cleanup ctbackend.Result
    NextChange time.Time
}
type Coordinator struct { /* serialized accepted plan, scope, cumulative history, state */ }
func NewCoordinator(scope Scope, effects Effects, clock Clock) (*Coordinator, error)
func (c *Coordinator) Replace(ctx context.Context, input networkplan.Input) (State, error)
func (c *Coordinator) Refresh(ctx context.Context) (State, error)
func (c *Coordinator) RetryCleanup(ctx context.Context) (State, error)
func (c *Coordinator) State() State
func (c *Coordinator) Run(ctx context.Context) error
```

- [ ] **1. Write failing tests**; build literal fixtures in this package. The following are assertion excerpts with case-local variables. Fix t0=`2026-10-15T10:00:00Z`, HTTPS end=t0+2s, SSH end=t0+4s and future start=t0+1s; pure unit cases use fake clocks:

```go
// TestBridgeCoordinatorOrdering: Verify -> Apply/Remove -> confirmation -> record -> Revoke; rejected candidate/apply invokes no revoker and preserves accepted generation.
if !reflect.DeepEqual(events, []string{"verify", "apply", "confirm", "record", "revoke"}) { t.Fatal(events) }
// TestBridgeCoordinatorHistoryAcrossExpiry: emitted old allow survives expiry and later confirmed refresh; never-emitted future/revoked/omitted membership never enters history; native expiry during confirmation does not erase ACKed emission.
if len(state.History) != 1 || !state.History[0].Rule.ExpiresAt.Equal(end) { t.Fatal(state) }
// TestBridgeCoordinatorCleanupRetry: cleanup failure advances confirmed policy without rollback, keeps history and counters; explicit retry rematerializes and confirms fresh after then achieves Remaining == 0.
if failed.Status != "cleanup-pending" || failed.Generation != 2 || retried.Status != "confirmed" || retried.Cleanup.Remaining != 0 { t.Fatal(failed, retried) }
// TestBridgeCoordinatorUnknownIsTerminal: missing/ambiguous receipt or failed confirmation marks unknown; no cleanup or future application; preserve last known history; namespace disposal is harness responsibility.
if state.Status != "unknown" || cleanupCalls != 0 || effectsAfterFault != 0 { t.Fatal(state) }
// TestBridgeCoordinatorEmptyAndRemovedAgent: last Binding removal and validated complete Agent removal use exact owned-table removal; arbitrary unknown-Agent error cannot authorize it; empty-state retry works without new withdrawals; graceful calls no revoker.
if removeCalls != 1 || nonemptyApplyCalls != 0 || !state.Receipt.OwnedAbsent || state.Cleanup.Remaining != 0 { t.Fatal(state) }
// TestBridgeCoordinatorIdentityAndOwnership: wrong epoch or namespace fails before effects; receipt cannot add extra emitted rules or omit omission accounting; mutated State does not mutate coordinator.
if err == nil || effectCalls != 0 || !reflect.DeepEqual(before, fresh) { t.Fatal(err, fresh) }
// TestBridgeDriverFutureAndExpiry: fake clock installs future allow and deny and removes expired deny at their exact boundaries using current generation.
if !allowAppliedAt.Equal(start) || !denyAppliedAt.Equal(denyStart) || !denyRemovedAt.Equal(denyEnd) { t.Fatal(events) }
// TestBridgeDriverStaleCancellationAndDelay: stale wakeup after Replace never replays old plan; elapsed boundary during a blocked operation triggers immediate refresh; cancelled Run returns after joining in-flight work, with no later apply.
if oldGenerationReplayed || appliesAfterCancel != 0 || !refreshedImmediately { t.Fatal(events) }
```

- [ ] **2. Verify red.** Run `go test -race ./internal/validation/planbridge -run "TestBridge(Coordinator|Driver)" -count=1`. Expect missing API/behavior to fail; for integration cases that already pass, record the baseline and add a literal counterexample instead of breaking working code artificially.

- [ ] **3. Implement.** NewCoordinator rejects nil Effects/Clock and invalid scope. Replace validates and applies under one serial gate; Refresh/RetryCleanup use the accepted immutable plan. A generation advances only after matching confirmed receipt/readback, before strict cleanup. Join receipt.Emitted to all origins and accumulate historical allow/protection without discarding expiry or deduplicating away origins. Validate that emitted plus omitted rules account exactly for submitted membership; tolerate deadline-based omission only. Compute strict before from cumulative confirmed emission, after from fresh current materialization. If prepare/apply/confirm crosses NextChange, rematerialize, apply and confirm at fresh time before any strict deletion, so after corresponds to the actual confirmed current policy. Empty removal requires validated full replacement/withdrawals or a retained confirmed empty state for retry; initial trusted known-empty state still confirms table absence. Unknown receipt/confirmation faults the fixture and blocks all further policy/cleanup effects; known rejection preserves last confirmed state. Cleanup failure returns error and cleanup-pending with counters but keeps the new accepted plan. Retry first refreshes/confirms current time-sensitive policy. Clock/timer injection proves stale event/cancel behavior; the real clock uses standard time.Timer. Run re-reads current generation/NextChange after each wakeup and operation, refreshes immediately across elapsed boundaries, and stops/joins on cancellation. Concurrent State reads return owned copies. Check ctx again immediately before starting each effect.

- [ ] **4. Verify green.** Run `go test -race ./internal/validation/planbridge -run "TestBridge(Coordinator|Driver)" -count=1`; require PASS for every named test, no unexpected skip. Keep prior tasks and M0 regression tests passing.

- [ ] **5. Commit.** Stage only the task files and commit as `feat(validation): reconcile confirmed plans and historical cleanup`. Preserve failure/green evidence in the final validation record.

### Task 6: Real dual-stack contract cases

**Files:** Task 6 paths above; external integration package `planbridge_test`.

**Interfaces consumed:** Tasks 1-5, kerneltest OpenTCP/OpenUDP/SendTCP/Exchange, actual nftables/ctnetlink adapters.

**Interfaces produced:** Integration/evidence only; no new domain API.

- [ ] **1. Write failing tests**; build literal fixtures in this package. The following are assertion excerpts with case-local variables. Fix t0=`2026-10-15T10:00:00Z`, HTTPS end=t0+2s, SSH end=t0+4s and future start=t0+1s; pure unit cases use fake clocks:

Use the exact required kernel cases in the scenario table below; assert packet/state observations, not only conversion output.

- [ ] **2. Verify red.** Run `PICKETX_KERNEL_TESTS=1 go test -tags=integration ./internal/validation/planbridge -run "^TestBridge" -count=1 -timeout=8m`. Expect missing API/behavior to fail; for integration cases that already pass, record the baseline and add a literal counterexample instead of breaking working code artificially.

- [ ] **3. Implement.** Implement all 16 required kernel cases below using exact logical inputs and independent literal allow/deny expectations, then compare Core.Decide, generated members and real packets. Run both address families in each case; annotate UDP/NAT protocols. Real denial must follow successful unfiltered probe. Use separate topologies for ordinary/NAT and strict/graceful modes. Log exact absolute boundaries, first observation and apply/confirm times; require <=1s without changing expiry. Partial/dump failure tests inject failures around real adapters operating on real flows, and label that injection explicitly; mocks alone do not satisfy the kernel half. Invalid-batch case must get a real kernel transaction rejection, not an injected returned error. Never promise socket termination after unbinding; test owned-table absence and unmanaged controls there.

- [ ] **4. Verify green.** Run `PICKETX_KERNEL_TESTS=1 go test -tags=integration ./internal/validation/planbridge -run "^TestBridge" -count=1 -timeout=8m`; require PASS for every named test, no unexpected skip. Keep prior tasks and M0 regression tests passing.

- [ ] **5. Commit.** Stage only the task files and commit as `test(validation): prove core bridge with real dual-stack packets`. Preserve failure/green evidence in the final validation record.

#### Required bridge kernel scenarios

| Test | Design acceptance class | Mandatory observations |
| --- | --- | --- |
| `TestBridgeGiteaIndependentDeadlines` | Gitea-shaped independent limits | TCP/443 and TCP/22, exact sources plus /26 and /80 contributions, literal Core results and actual new connections agree |
| `TestBridgeDenyAndContributionRevocation` | Deny and revocation | deny wins; removing one Grant preserves wider independent Grant and other entry/source controls |
| `TestBridgeUDPCurrentPermission` | UDP | UDP/8002 authorized echo, revoke/strict deletion then old UDP cannot bypass current policy |
| `TestBridgeNativeExpiryGraceful` | Native expiry | no reapply; new TCP denied by end+1s while previously established graceful echo continues |
| `TestBridgeInactiveAndOmittedMembership` | Invalid or inactive input | future/revoked/expired absent, submillisecond omitted receipt, protected tuple denies |
| `TestBridgeFutureBoundaries` | Future boundaries | real driver activates future allow/deny, expires deny, observes within 1s; stale event/cancel control never reinstalls old permission |
| `TestBridgeReplayAndLateApply` | Replay/late application | near-expiry backend reconstruction, repeated application and actual lock/preparation delay retain original end; expired never becomes permanent |
| `TestBridgeHistoricalStrictRetry` | Historical strict cleanup | old established flow survives expiry plus an intervening confirmed apply, injected cleanup failure preserves history, retry actually selects/deletes it |
| `TestBridgeStrictPreservesIndependentFlows` | Strict preservation | overlap/other entry/source, unmarked and foreign mark flow survive; protected withdrawn new TCP and ACK/PSH remain denied with drop counter proof |
| `TestBridgeRejectedAtomicBatch` | Failed apply/confirmation | real invalid combined batch preserves old admission and NAT reachability; no revoker; separate readback-failure seam yields unknown |
| `TestBridgeInterruptedAndPartialCleanup` | Interrupted/partial cleanup | real established flows plus actual ctnetlink adapter with injected dump/delete failure: incomplete dump deletes zero; partial failure retains counts and explicit retry finishes |
| `TestBridgeBindingOwnership` | Binding ownership | delete one of two Bindings retains protection and remaining origin IDs; delete final Grant still protects |
| `TestBridgeEmptyBindingStrict` | Empty owned desired | last Binding removed: exact table absent and historical marked Remaining == 0, foreign table/mark and unmanaged port remain |
| `TestBridgeRemovedAgentStrict` | Empty owned desired | complete validated Agent removal produces withdrawals, table absence and zero owned remaining; explicit cleanup retry works after accepted empty state |
| `TestBridgeEmptyGraceful` | Empty owned desired | last Binding removed: confirmed table absence, no conntrack deletion, unmanaged controls available |
| `TestBridgeNATOriginalReply` | NAT original/reply tuples | VIP TCP/8000 withdrawn while 8001 remains; observe Server/9000 reply and original/reply/zone/mark precision |

The design has 16 acceptance classes: context and provenance are unit-only; the 14 remaining classes map to these 16 kernel cases (empty ownership has three distinct cases). All `both` classes also have unit assertions in Tasks 1/3/4/5. Each case records IPv4 and IPv6 subresults; the evidence gate requires both, so a passed parent with a skipped/missing family cannot satisfy acceptance. A top-level pass alone is insufficient.

### Task 7: Evidence gate and delivery

**Files:** Task 7 paths above; add bridge result parsing and structured metadata without deleting M0 acceptance.

**Interfaces consumed:** existing Report/ScenarioResult, ReadEvents and CheckRequired plus Task 6 case list.

**Interfaces produced:** Integration/evidence only; no new domain API.

- [ ] **1. Write failing tests**; build literal fixtures in this package. The following are assertion excerpts with case-local variables. Fix t0=`2026-10-15T10:00:00Z`, HTTPS end=t0+2s, SSH end=t0+4s and future start=t0+1s; pure unit cases use fake clocks:

```go
// TestBridgeEvidenceRequiredAndDuplicate: missing/skipped/failed/duplicate bridge top-level scenario or wrong shared revision fails; duplicate terminal events for one case cannot be collapsed to pass.
if missingErr == nil || skippedErr == nil || duplicateErr == nil || wrongRevisionErr == nil { t.Fatal("gate accepted invalid evidence") }
// TestBridgeEvidenceFailureArtifact: malformed/missing event stream still produces report.json with errors and available evidence; old bridge output removed on rerun; unit pass cannot satisfy kernel gate.
if exitCode == 0 || !reportExists || len(reportErrors) == 0 || staleBridgeAccepted { t.Fatal(exitCode, reportErrors) }
```

- [ ] **2. Verify red.** Run `go test ./internal/validation/report ./cmd/m0-report -count=1`. Expect missing API/behavior to fail; for integration cases that already pass, record the baseline and add a literal counterexample instead of breaking working code artificially.

- [ ] **3. Implement.** Extend ScenarioResult/Report with backward-compatible structured bridge evidence for immutable shared revision/module version, scope identity, family, tuple/provenance, phase/timestamps and cleanup outcome. Use a JSON-prefixed event emitted from integration tests; keep human text for diagnostics. ReadEvents must reject duplicate top-level terminal results while still accumulating ordinary output; record family outcomes separately and gate them. Append exactly the 16 required bridge names to all 17 M0 names. Runner builds the new bridge test binary, removes stale bridge events, runs each binary in disposable unshare --net and always attempts the reporter; capture build/setup failures with current metadata too. Reporter writes report.json with available observations and explicit errors even if input is absent/malformed, then exits nonzero. CI uploads current failure artifacts with always(). Inspect go list -deps: no Echo/GORM/Controller/Web linked through bridge; record actual go list -m all and immutable checksum. Paired docs report only actual kernel/toolchain results and remaining production work.

- [ ] **4. Verify green.** Run `go test ./internal/validation/report ./cmd/m0-report -count=1`; require PASS for every named test, no unexpected skip. Keep prior tasks and M0 regression tests passing.

- [ ] **5. Commit.** Stage only the task files and commit as `test(validation): gate bridge evidence and document results`. Preserve failure/green evidence in the final validation record.

## Full verification and handoff

- [ ] Run all unit/regression, race, vet, build and format checks in PicketXD:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
test -z "$(gofmt -l cmd internal)"
go list -deps ./internal/validation/planbridge
go list -m all
PICKETX_KERNEL_TESTS=1 scripts/test-kernel.sh
```

- [ ] Require all **33** kernel scenarios (17 retained + 16 bridge), both family subresults for bridge, current shared immutable revision/checksum and failure artifacts. Missing privilege means acceptance remains incomplete; never replace it with mocks or old M0 evidence.
- [ ] Request one fresh whole-branch review covering the design, every task, failure-state transitions, expiry/readback semantics and exact deletion filters. Fix blocking findings and rerun affected checks before integration.
- [ ] Publish an implementation PR in PicketXD and a paired central status update linking the exact tested commit/run/artifacts. Preserve native inline execution; this plan does not choose per-task parallel implementers.
- [ ] Review this written plan before implementation. The broader [basic-function roadmap](../../BASIC_FUNCTION_ROADMAP.md) records subsequent product slices; finishing M1.3 alone does not complete L4/L7 or management UI.
