# Node observations implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement task-by-task.

**Goal:** Real read-only node observations with honest report freshness.
**Architecture:** Agent service supplies atomic projection; publisher guards delivery
readiness, management enforces publication:read, Echo/cookie adapters expose it.
React polls the real endpoint, retaining explicitly historical data on error.
**Tech Stack:** Existing Go/Echo/gRPC/Casbin/SCS and React/TypeScript/i18next/npm.
**Spec:** docs/superpowers/specs/2026-10-19-node-observations-design.md

## Global constraints
- Report freshness60 seconds; poll15 seconds; revisions decimal strings.
- No protocol/Agent/kernel/database migration or enforcement proof.
- Preserve single authority, exact current-session correlation, live RBAC and no-store.
- Native full Go/race/vet/build, locked Web checks and browser acceptance required.

## Context map
| File | Role/change |
| --- | --- |
| internal/agentsync/observations.go | Atomic sorted credential-scope projection |
| internal/agentsync/observations_test.go | Correlation/freshness/lifecycle boundaries |
| internal/controllerapp/management.go, runtime.go | Serialize ready publisher observation; explicit interface |
| internal/management/service.go | Authorized observation operation |
| internal/controller/observations.go, management.go, browser.go | Bearer/cookie GET and DTO docs |
| web/src/features/nodes/Nodes.tsx, App.tsx, locales/* | Real polling cards and navigation |
| web/e2e/console.mjs, nodes.mjs | Existing real browser fixture and new observation scenarios |
| internal/controllerapp/observations_test.go | Actual gRPC/HTTP assertions, not kernel proof |
| docs/api/* | Regenerated OpenAPI contract |

Dependencies: existing agentsync session/report lock, publisher.mu, management
principal checks, request AbortSignal, console shared style and browser harness.
No public protocol break, database migration or configuration key change.
Context map reviewed against current sources; spec self-review found no conflict.

## Review focus
1. New session must not refresh a previous session's report: Task1 test replacement.
2. Pending revision must not inherit prior APPLIED: Task1 mismatch test.
3. Clock skew must not produce fresh status: Task1 negative age and untrusted clock test.
4. Revocation/failure must not leak old live state: Task2 runtime/auth tests.
5. Hidden/unmounted UI must ignore obsolete responses: Task3 disposal/abort guards;
   delayed-response/visibility race browser regression is explicitly deferred below.

### Task 1: Atomic service observations
Produces `Service.Observations() (Observations, error)`; private `observationsAt(time.Time)`
under lock allows deterministic boundary tests. DTOs in agentsync are plain Go/JSON.
- [ ] Write tests: missing report, fresh/current APPLIED,60-second exact boundary,
  backward time, stale revision/hash, disconnect/new session, dedup/revoke/closed,
  decimal revision greater than2^53, sorted independent snapshots.
- [ ] Run `go test ./internal/agentsync -run Observations -count=1`; expected missing API RED.
- [ ] Implement projection without changing stream acceptance or report mutation.
- [ ] Run `go test -race -count=1 ./internal/agentsync`; expected PASS; commit.

### Task 2: Protected observation API
Consumes Task1 DTO; publisher `AgentObservations() (agentsync.Observations,error)`;
management `AgentObservations(Principal)` uses repository capability and read RBAC.
- [ ] Add real runtime gRPC→HTTP tests: unreported, applied assertion, publication
  mismatch, disconnect/reconnect, unauthorized/forbidden and failed readiness.
  Add Bearer/cookie route contract and unknown-query rejection tests.
- [ ] Run new focused tests; expected404 RED.
- [ ] Implement adapters/assembly and regenerate pinned OpenAPI.
- [ ] Run controller/management/agentsync tests and focused runtime tests; expected PASS.
- [ ] Commit API; full native CI includes required local Unix EPERM scenario.

### Task 3: Real console and delivery
Consumes GET /console/api/v1/agent-observations; DTO matches Task1 fields.
- [ ] Add browser test missing Nodes navigation; execute native CI for RED before UI.
- [ ] Implement read-only Nodes card page with abortable nonoverlapping15-second
  polling/visibility resume, freshness countdown, error/empty/loading/stale and i18n.
- [ ] Execute npm ci/typecheck/lint/build and real Chromium empty/unreported/viewer/
  refresh failure/mobile scenarios; no mocked execution claim. Inspect screenshots.
- [ ] Run complete Go ordinary/race/vet/build and exact-source native CI; record
  local Unix EPERM honestly; native suite must pass without skip.
- [ ] Commit bilingual usage/checkpoint; independent entire branch review; fix blockers
  RED→GREEN, then exact-head CI/expected-head merge/mainCI and release lease.

## Ledger
Pre-flight: Task1 DTO matches Task2 adapter and Task3 browser; no conflict.
Ruling: keep observations in memory — no stale restart resurrection; cost no history.
Ruling: show credentialed scopes only — explicit inventory boundary; cost no enrollment inventory.
Ruling: no heartbeat/kernel change — avoid compatibility expansion; cost no continuous proof.
Baseline full Go: only actual Unix socket EPERM locally; required nativeCI retained.
Independent review91ef769..cb61530: no Critical/Important. Deferred Minor items are
queued visibility-resume refresh while an old request is active, and delayed-response/
unmount/visibility race browser assertions. Existing settled navigation is tested;
do not claim full lifecycle race coverage. Exact-source native CI/visual gate required.

## UI design review
Reuse background #f5f7fa, text #18283b, action #145da0, muted #526174,
white surface and system sans16px/1.5 from the accepted console. Left-aligned
node cards pair distributed revision with explicitly attributed Agent assertion;
hashes are expandable, long identifiers wrap. No new theme, animation or live-success
badge. UI skill's operations pattern fits timestamp/stale semantics; its marketing
sections/glass theme do not fit this existing console and are not adopted.
Use contextual status announcements without focus theft; color never carries state
alone. Refresh button retains44px target. Review375px and desktop screenshots.
