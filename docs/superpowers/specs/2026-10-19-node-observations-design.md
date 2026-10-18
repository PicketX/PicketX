# Node observations design

## Intent and scope
Operators need to distinguish distributed desired state from an Agent assertion.
Build a real read-only observation API and console page, not simulated execution.
Keep Go/Echo, SCS/Casbin and React/i18next; no protocol, Agent, kernel or database
migration. Routine approval is waived by the autonomous delivery instruction.

## Chosen approach
Read a bounded, sorted snapshot of currently credentialed Agent scopes under the
existing Agent service lock. Deduplicate multiple credentials. Report authority,
distributed revision/hash, connection presence and latest report revision/hash,
phase, trusted receive time and untrusted Agent observation time. Omit raw Agent
detail and all credential material. Clear report availability on service restart;
removed credential scopes disappear. The inventory is not every declared Agent.

Use a 60-second display freshness window from Controller receive time. A report
is current only if it belongs to the active reporting session, matches the current
distributed revision/hash, and its age is nonnegative and strictly below60 seconds.
Expose each axis independently: connection (connected/disconnected), report
freshness (unknown/stale/fresh), and matches_desired. APPLIED always means
Agent-reported, never independent kernel proof. A new connection cannot refresh
an old report. FAILED/RECEIVED remain distinct phases. Expiry does not refresh
timestamps or establish current permissions. No report heartbeat exists yet.

The publisher serializes observation with commit/delivery and checks readiness;
failed delivery is503 rather than a misleading current snapshot. Reuse
publication:read permission with live credential/RBAC checks and existing cookie
security. Add GET /api/v1/agent-observations and cookie equivalent; no-store.
Reject query parameters. Encode uint64 revisions as decimal strings for JS safety.

## UI
Read-only Nodes page mounts on entry; bounded15-second nonoverlapping polling,
abort on unmount, refresh after visibility resume, explicit refresh button.
Show loading/empty/error/last-success/browser-stale states. Retained data after
failure is visibly historical. Derive report ageing from server response freshness
budget plus elapsed browser time, not the browser's absolute clock. English/Chinese,
375px cards and existing tokens. No green access-enabled claim.

## Alternatives and costs
Persisted status introduces stale recovery/epoch semantics without current need;
new heartbeat protocol crosses Agent compatibility and execution scope. Defer both.
Polling adds bounded delay; only credentialed scopes are shown; raw error detail
is omitted to avoid exposing secrets. TLS/enrollment/approval and privileged Linux
joint acceptance remain separate, unchanged requirements.

## Acceptance
RED→GREEN tests for atomic views, revision mismatch, session replacement,
disconnect, missing report, credential removal/deduplication,60-second boundary,
Agent clock skew and large revisions. Real runtime gRPC→HTTP tests plus protected
cookie/API and OpenAPI contracts. Real Chromium against the running Controller
checks Nodes entry, empty/unreported status, refresh errors, viewer and mobile.
Protocol clients used in tests are explicitly assertions, not kernel evidence.
Full Go/race/vet/build and locked Web checks; native CI executes Unix test without
skip. One independent whole-branch review then exact-head CI/merge/main acceptance.
