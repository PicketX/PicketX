# Capability refresh investigation

## Intent and bounded scope
Resolve PR36 main39300533866 attempt1 create-only user's empty operations/navigation
after reload; same-source attempt2 passed, not a repair. Preserve Casbin, identity,
rate limits and backend enforcement; no Agent/kernel/dependency-stack change.
Autonomous mandate waives routine design/plan/push/merge pauses; one independent
whole-branch review remains. Isolated branch starts from exact Core85fa14ef.

## Investigation
Failure: access-requests.mjs167 after creator.reload, authenticated creator and
role:create-only visible, no operations/navigation. Root cause not established.
Compare real SQLite/Access periodic reload with public /me/browser-load evidence.
No production fix without specific failing behavioral reproduction. Never publish
cookies/tokens/passwords/credential documents/private SQLite files.

## Context Map (self-reviewed)
| Files | Role |
| --- | --- |
| internal/managementaccess/access.go, access_test.go | Real shared policy reload/evaluation |
| internal/controller/management.go, access_requests_capabilities_test.go | /me operations and API capability contract |
| web/src/App.tsx, lib/console.ts | Session/bootstrap/load/no-store boundary |
| web/e2e/access-requests.mjs, console.mjs | Actual custom-role login/reload/Chromium |
| docs/AUTONOMOUS_DELIVERY{,.zh-CN}.md | Lease and continuation |
Dependencies: official Casbin/GORM Adapter and React19, no new dependencies.
References: ADR0001, existing periodic reload/failure and capability API tests.
No API/schema/migration/config change intended; read product/architecture contracts
before any discovered production behavior change. Diagnose first, scope fix second.

## Acceptance
Reproduce before fixing; full Go/race/vet/build and locked Web checks, actual native
browser/service gates; one independent review, exact-source/expected-head merge
and main raw/artifact verification. Instrumentation alone is not a claimed fix.

## Root-cause extension — 18:48Z
Pinned Playwright coreBundle.js25672..25736 calls predicate() without awaiting its
return before its truth check. An async function returns a truthy Promise: polling
finishes even when that Promise resolves false. Existing creator readiness did not
wait for the30s periodic policy propagation. Replace ONLY the test wait with an
explicit awaited node-side loop. Guarantee /me200[] before real custom-rule INSERT;
normal periodic reload must produce actual permission before any strict reload.
Keep one bounded deadline/abort, reject non200/malformed/transport immediately,
only poll valid200 missing operation. Real-HTTP tests and actual native RED/GREEN;
not a production Casbin/React repair or policy widening. Layout/tokens untouched.
