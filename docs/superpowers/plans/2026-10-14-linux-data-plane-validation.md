# Linux Data-plane Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task for the recommended direct execution. Use superpowers:subagent-driven-development only if the user selects delegated execution. Steps use checkbox syntax for tracking.

**Goal:** Produce a repeatable M0 Linux validation suite and truthful evidence for nftables enforcement and strict scoped conntrack revocation.

**Architecture:** Implement test-only fixtures and adapters in PicketXD, with real packet probes inside isolated network namespaces. Snapshot fixtures represent expected effective permissions; they are not the production Grant compiler or a published configuration schema.

**Tech Stack:** Go 1.27.0, google/nftables v0.3.0, initial vishvananda/netlink v1.3.1, Linux network namespaces and GitHub Actions.

**Spec:** [M0 design](../specs/2026-10-14-linux-data-plane-validation.md)

**Status:** Proposed for review; tasks not executed. All file paths below are relative to PicketX/PicketXD.

## Global Constraints

- Keep the implementation in PicketXD; align its empty repository's Apache-2.0 LICENSE with the agreed GPL-3.0-or-later community direction before adding Agent code.
- Own only `table inet picketx` and `picketx_` objects inside isolated test namespaces. Baseline PREROUTING priority is -110; test marker is 0x5049434b.
- IPv4 192.0.2.0/24 and IPv6 2001:db8:1::/64 are fixture addresses, not production defaults.
- Pin go.mod/go.sum. No runtime C library or nft/conntrack command dependency.
- Require explicit `PICKETX_KERNEL_TESTS=1` for kernel tests; opt-in with missing privilege/setup is failure, not skip.
- Do not implement or freeze manifest/protobuf/business schemas, Controller transport, event subscription or activity renewal.
- A skipped kernel suite cannot satisfy M0. Record the actual tested kernel and dependency versions.

## Review Focus

- No privilege or namespace setup failure: fail before rule mutation in opted-in runs; Task 1.
- Overlapping scope and non-owned connections: preserve remaining effective access and unmarked flows; Tasks 2 and 3.
- Restart near an absolute deadline: never reissue the original full TTL; Task 2.
- NAT original/reply tuple mismatch: delete the intended owned mapping only; Tasks 3 and 4.
- Interrupted dump or partial deletion: retain a failed/incomplete result with evidence; Tasks 3 and 4.

---

### Task 1: Bootstrap private validation fixtures and environment checks

**Files:** Create `go.mod`, `go.sum`, `AGENTS.md`, `README.md`, `README.zh-CN.md`, `THIRD_PARTY_NOTICES.md`, `internal/validation/fixture/snapshot.go`, `snapshot_test.go`, `internal/validation/kerneltest/preflight_linux.go`, `preflight_linux_test.go`; modify `LICENSE`.

**Interfaces:**
- `fixture.Endpoint`: Destination netip.Prefix, Protocol uint8, PortFirst/PortLast uint16.
- `fixture.Rule`: Source netip.Prefix, Endpoint Endpoint, ExpiresAt time.Time; a zero deadline is a test-only non-expiring fixture.
- `fixture.Snapshot`: Protected []Endpoint, Allow/Deny []Rule.
- `fixture.Validate(Snapshot) error`: reject invalid/noncanonical prefixes, source/destination family mismatch within a rule (a snapshot may contain both IPv4 and IPv6), protocol other than TCP(6)/UDP(17), reversed/zero port ranges and empty protection scope.
- `kerneltest.Preflight(ctx context.Context) (string, error)`: return kernel release only when the opted-in process can create/use an isolated test namespace.

- [ ] Write `TestValidateSnapshot` for valid dual-family inputs and each rejected fixture shape.
- [ ] Write `TestPreflightMissingPrivilege`: opted-in missing capability returns an error before invoking rule mutation; use internal `preflight(ctx context.Context, probe func(context.Context) error) (string, error)` for injected namespace checks.
- [ ] Run those tests before implementations; verify failures originate from the absent validation/preflight behavior.
- [ ] Implement fixture validation and preflight; initialize the Go module and notices, align LICENSE, and document only the validation commands and current non-production status.
- [ ] Run `go test ./...`, `go vet ./...`, `go build ./...`; confirm no external binary is invoked by fixture/backend runtime code.
- [ ] Commit: `chore(agent): bootstrap Linux data-plane validation`.

### Task 2: Validate nftables admission, sets, deadlines and batch failure

**Files:** Create `internal/validation/nftbackend/backend_linux.go`, `backend_linux_test.go`, `backend_integration_test.go`, and `internal/validation/kerneltest/topology_linux.go`.

**Interfaces:**
- `nftbackend.New(conn *nftables.Conn) *Backend`.
- `(*Backend).Apply(ctx context.Context, snapshot fixture.Snapshot, now time.Time) error`: validate first, stage one owned nftables batch, omit expired memberships, and calculate remaining timeout from the absolute deadline.
- `(*Backend).Close() error`: close its connection; teardown owns namespace cleanup, not unrelated tables.
- `kerneltest.WithTopology(t *testing.T, run func(*Topology))`: establish isolated client/enforcement namespaces, echo endpoints and deterministic cleanup.
- `(*Topology).Probe(ctx context.Context, source, destination netip.Addr, protocol string, port uint16) error`: perform real packet/connection probes with deadlines.
- `(*Topology).OpenTCP(ctx context.Context, source, destination netip.Addr, port uint16) (net.Conn, error)`: retain an established fixture connection for graceful/strict expiry tests.

- [ ] Write `TestApplyRejectsInvalidSnapshotBeforeFlush` using a small internal batch seam; assert invalid input queues/sends no mutation.
- [ ] Write tagged integration cases `TestDefaultDenyDualStack`, `TestScopedAllowDualStack`, `TestDenyPrecedence`, `TestConcatenatedIntervalSets`, `TestOverlapRetainsMembership`, `TestIndependentDeadlines`, `TestNativeTimeoutWithoutController`, `TestGracefulExpiry`, `TestRestartPreservesDeadline`, and `TestInvalidBatchPreservesPreviousState`.
- [ ] Verify each newly introduced case fails against missing behavior on capable Linux. Tests skipped without opt-in are not the red phase or kernel evidence.
- [ ] Implement the minimum fixture backend and topology. Inject a genuinely invalid Netlink operation for the failed-batch case; confirm the preceding policy still controls traffic. Keep any unsupported required set/timeout combination visible as failure.
- [ ] Run unit checks and opted-in integration cases; inspect Go test JSON for actual executions and their packet observations.
- [ ] Commit: `test(agent): validate nftables admission and expiry`.

### Task 3: Select and delete only withdrawn owned connections

**Files:** Create `internal/validation/ctbackend/select_linux.go`, `select_linux_test.go`, `backend_linux.go`, `backend_linux_test.go`.

**Interfaces:**
- `ctbackend.SelectWithdrawn(flows []*netlink.ConntrackFlow, before, after fixture.Snapshot, mark uint32, now time.Time) []*netlink.ConntrackFlow`: match original tuples at the documented pre-DNAT policy boundary, require ownership mark, and exclude still-allowed flows after deny/expiry evaluation.
- `ctbackend.Result`: Selected, Deleted, Remaining int.
- `ctbackend.New(handle *netlink.Handle) *Backend`.
- `(*Backend).Revoke(ctx context.Context, before, after fixture.Snapshot, mark uint32, now time.Time) (Result, error)`: dump, scope selection, delete, and confirm the withdrawn owned scope no longer remains; use an internal injectable dump/delete seam for error tests.

- [ ] Write `TestSelectWithdrawn` with IPv4/IPv6, original/reply NAT tuples, exact-host/prefix overlap, another port/protocol, new deny, expired contribution and unmarked-flow fixtures.
- [ ] Write `TestRevokeReportsInterruptedDump` and `TestRevokeReportsPartialDeletion`; assert error is non-nil and incomplete counts cannot be returned as successful cleanup. Write `TestRevokeNeverFlushesTable`.
- [ ] Run unit tests and confirm failures before implementing selection/deletion behavior.
- [ ] Implement scoped matching, vishvananda/netlink integration, returned evidence and post-deletion confirmation. Treat actual library limitations as findings; never silence incomplete dumps or use whole-table flush.
- [ ] Run `go test ./...`, `go vet ./...`, `go build ./...`.
- [ ] Commit: `test(agent): validate scoped conntrack revocation`.

### Task 4: Run the integrated contract on real Linux and publish evidence

**Files:** Create `internal/validation/ctbackend/backend_integration_test.go`, `internal/validation/report/report.go`, `report_test.go`, `internal/validation/scenario/apply_linux.go`, `apply_linux_test.go`, `scripts/test-kernel.sh`, `.github/workflows/validation.yml`, `docs/validation/M0.md`, `docs/validation/M0.zh-CN.md`.

**Interfaces:**
- `report.ScenarioResult`: Name, Status, Detail string; Selected, Deleted, Remaining int.
- `report.Report`: Kernel, Go string; Libraries map[string]string; Scenarios []ScenarioResult.
- `report.Write(w io.Writer, report Report) error`: emit JSON; preserve pass/fail/skipped/unsupported distinctions.
- `scenario.Applier` and `scenario.Revoker`: the Apply/Revoke methods defined in Tasks 2 and 3.
- `scenario.ApplyStrict(ctx context.Context, applier Applier, revoker Revoker, before, after fixture.Snapshot, mark uint32, now time.Time) (ctbackend.Result, error)`: never invoke Revoke when Apply fails.
- `scripts/test-kernel.sh`: require capable Linux, run prebuilt tagged tests inside an isolated parent namespace, preserve test exit status and cleanup on failure.

- [ ] Write `TestApplyStrictRejectsUnappliedState`; a failing applier must produce an error and zero revoker calls. Write `TestReportPreservesIncompleteAndSkippedResults`; assert JSON does not convert failures, unsupported results or skips into passes.
- [ ] Write integration cases `TestStrictRevokeDualStack`, `TestStrictRevokePreservesOtherGrant`, `TestStrictRevokePreservesUnrelatedFlow`, `TestOldTCPPacketsCannotResurrect`, `TestUDPAfterDeletionRequiresCurrentAllow`, `TestNATScopedDeletion`, `TestApplyFailureDoesNotTriggerDeletion`.
- [ ] Run new cases against absent combined behavior and verify real failures; then implement the coordinator ordering and report/runner behavior.
- [ ] Add a unit job and an isolated Linux kernel job; pin action revisions. Build/download modules before privileged execution, archive test JSON and environment/scenario reports, and fail the kernel job if required scenarios did not execute.
- [ ] Run `go test ./...`, `go vet ./...`, `go build ./...`, then `PICKETX_KERNEL_TESTS=1 scripts/test-kernel.sh` on capable Linux. Review each acceptance row against observed results.
- [ ] Write findings with the actual kernel/versions, unsupported behavior, partial failures and any reviewed dependency adjustment; never mark M0 complete from skipped tests.
- [ ] Commit: `test(agent): run Linux enforcement conformance suite`.

## Completion and next stage

Open an implementation PR in PicketXD with actual checks and remaining limitations. Review code and artifacts before merging. M0 findings determine the production backend contract; M1 then implements shared Core/domain schemas and the first Gitea workflow incrementally. The current Controller documentation PR contains no Agent code and is not kernel verification.
