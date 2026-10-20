# Controller CLI diagnostics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline. Steps use checkbox syntax.

**Goal:** Shipped CLI reads actual Controller desired/Agent report status safely.
**Architecture:** Shared internal/controllerclient + thin Cobra adapters.
**Tech Stack:** Existing Go/Cobra/stdlib HTTP/TLS; no dependencies.
**Spec:** docs/superpowers/specs/2026-10-20-controller-cli-diagnostics.md

## Global Constraints
- --controller/--key-file required; --agent optional; --format text|json defaulttext.
- --timeout default5s positive<=30s; --ca-file adds verified trust only.
- Only literal loopback HTTP; no redirects/proxy/env credentials/TLS bypass.
- Read publication/observations/publication; reject inconsistent authority/revision.
- Deduct elapsedTTL; output enforcement_confirmed=false, no policy/facts/key/errors.
- Doctor success: readiness and nonempty all reported-applied, otherwise nonzero.
- Preserve all original CI/native/recovery/four artifacts.

## Review Focus
- Redirects, proxies and errors cannot expose Bearer/private paths.
- Slow responses and concurrent publication cannot freshen mismatched/history reports.
- APPLIED/empty inventory/readiness cannot imply independent kernel verification.
- Ambiguous/unbounded JSON and hostile control characters fail with fixed diagnostics.
- Bad TLS/CA/SAN and unsafe key files cannot downgrade authentication.

### Task 1: Complete CLI status and doctor
**Files:** Spec Context Map.
**Interfaces:** controllerclient.Options{Origin,KeyFile,CAFile,Agent string; Timeout time.Duration};
Inspect(context.Context,Options,bool)(Report,error); Report includes Scope/Authority/
Revision/ObservedAt/Ready/Nodes/EnforcementConfirmed; Node includes AgentRef/
DesiredRevision/ReportedRevision/State/FreshForMillis. Existing cli.New adds commands.
- [ ] Existing CLI tests first: unknowncommands RED; valid disconnected/empty status,
  doctor report+nonzero, missing args/unsafe file/no secret regression assertions.
- [ ] Actual HTTP/TLS client tests first: fixed authenticated paths, healthy APPLIED,
  RECEIVED/FAILED/disconnected/stale/mismatch/empty, publication race, elapsedTTL;
  redirect zero destinationhits, denied/oversize/duplicate/unknown/trailing/badnull,
  timeout, badorigin/CA/SAN/private file. Run targeted tests, expected missingbehavior RED.
- [ ] Implement minimum typed client and commands; targeted full normal/race GREEN.
- [ ] Actual Controller/gRPC CLI integration plus shipped empty-PATH status/doctor/
  wrongaudience/stoppedserver checks. Full Go normal/race/vet/build, lockedWeb
  ci/typecheck/lint/build, all Node/helper tests; inherited EPERM explicitly recorded.
- [ ] Sync docs/commit; task-done runs go test -count=1 ./internal/cli ./internal/controllerclient.
- [ ] ONE whole-branch review; root one Critical/Important regression-first pass,
  defer Minor; ledger all declined-scope rulings/costs.
- [ ] Exacthead all3jobs/raw/fouractualarchives/source/static/notices/screenshots,
  expectedhead merge/fresh main and immediate release; whole-goal audit.
Preflight: single integrated task, no shared-task interfaces. Self-review complete;
all five review-focus classes mapped to tests. Integration remains separate gate.

