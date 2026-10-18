# Trusted HTTP Authorization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Native inline execution; one fresh independent whole-branch reviewer before integration.

**Goal:** Run source-Grant HTTP authorization through a scoped local API and actual Nginx.
**Architecture:** Immutable HTTPRoute constraints reuse Core's Grant/deny evaluator. Accepted publication and decision records share the existing SQLite owner; a separate Echo listener authenticates scoped integrations. Existing console publishes the same authority.
**Tech Stack:** Existing Go1.27/Echo5/Koanf2.3.8/GORM official SQLite/Casbin/SCS and React/TypeScript/Vite/npm; Nginx only as external ingress test/deployment dependency.
**Spec:** docs/superpowers/specs/2026-10-18-http-authorization.md

## Global Constraints
- Existing Apache/GPL boundary, Agent wire unchanged; no kernel/source copied.
-32–256 printable ASCII dedicated Tokens; loopback TCP/Unix0600 only; caller scope never grants permission.
- Exact host/method/path; independent absolute deadlines, Grant union, deny priority; cacheTTL0.
- All valid scoped decisions durable before response; schema3; no raw request/secret audit.
- No repeat routine approvals; safe expected-head updates/merge, complete tests and real CI required.

## Review Focus
- Encoded/dot/slash paths and duplicate context headers must not widen a route (Task1/2).
- Route UID reuse or non-TCP reference must fail complete publication (Task1).
- Audit storage failure must never return allow (Task2).
- Unknown caller or another route cannot consume authority via forged headers (Task2/3).
- Reused HTTP connection must reevaluate expiry/revoke and actual target (Task3).

## Context Map

### Files to Modify
| File | Purpose | Changes |
|---|---|---|
| pkg/configuration/{types,decode,candidate}.go | strict alpha publication | HTTPRoute parse/compile/lifecycle |
| pkg/httpauth/{engine,context}.go (new) | pure request authorization | strict context and immutable core-backed engine |
| internal/publication/{store,http}.go | accepted intent/durability | schema3, evaluate+audit |
| internal/authz/{service,credentials}.go (new) | trusted integration boundary | scoped credential resolver |
| internal/controller/{authz,management}.go | Echo DTO/HTTP | dedicated decision handlers/audit routes |
| internal/controllerapp/{runtime,authz}.go | owned lifecycle | listener/assembly/readiness |
| internal/runtimeconfig/config.go | Koanf typed boundary | disabled-by-default listener settings |
| apps/web/e2e/console.mjs, .github/workflows/ci.yml | real acceptance | Nginx/browser/application |
| docs/api, docs/HTTP_AUTHORIZATION* | generated/deployment contract | bilingual scope/examples |

### Dependencies
Core Engine.Decide/Validity reused; Candidate.Input unchanged. Store.Current holds immutable candidate. Existing management audit authorization and SCS reusable. Dedicated listener must not register management/console.

### Tests and reference patterns
configuration compile tests (strict objects/lifecycle), core network tests (independent lifetimes), publication managed tests (transactions/reopen), controller management tests (strict DTO/cursor), controllerapp runtime tests (real listeners/shutdown), actual console e2e (real Go/SQLite/Chromium).

### Risk Assessment
- [x] Additive alpha configuration/API contract; older Core rejects HTTPRoute, Agent wire unchanged.
- [x] SQLite versioned schema3 migration.
- [x] Opt-in runtime settings, no current deployment behavior change.

### Task 1: Immutable source-backed HTTP routes
**Files:** pkg/httpauth/*.go, pkg/configuration/{types,decode,candidate}.go and tests.
**Interfaces:** `httpauth.Compile(core.Snapshot, []Route) (*Engine,error)`; `Engine.Decide(Query) (core.Decision,error)`; `Candidate.HTTP() *httpauth.Engine`.
- [ ] Write failing actual Decode/Compile/Decide tests: no route/unknown, source IPv4/IPv6, overlapping independent deadlines, future/revoked, explicit deny, exact Host/method/path, malformed/encoded URI, pinned references and no route UID retarget.
- [ ] Run focused tests; Expected: missing implementation RED.
- [ ] Implement strict route/context and configuration integration; unchanged network projection.
- [ ] Run focused and full `go test -count=1 ./...`; Expected: PASS; commit.

### Task 2: Durable scoped local decisions
**Files:** internal/{publication,authz,controller,controllerapp,runtimeconfig}, generated OpenAPI and tests.
**Interfaces:** Store.DecideHTTP(caller string,q httpauth.Query) (HTTPDecision,error), Store.HTTPAudit(after int64,limit int); authz.Service.Decide(token string,q httpauth.Query), dedicated controller.NewAuthorizationHandler; runtime assembly typed authz config.
- [ ] Write failures for full runtime TCP+Unix, scope/credential rotation, strict JSON/header duplicates, schema2 upgrade/reopen, audit failure with no allow, RBAC audit visibility, startup-existing Unix object and shutdown.
- [ ] Run tests; Expected: missing routes/config/schema RED.
- [ ] Implement official Echo/validator adapters, private credential validation, schema3/evaluate+audit, isolated listeners/readiness and bounded lifecycle; document/generate both API groups.
- [ ] Full Go ordinary/race/vet/build, pinned generation, local OpenAPI; Expected: PASS; commit.

### Task 3: Real Nginx/browser/application acceptance
**Files:** examples/http/*, apps/web/e2e/console.mjs, .github/workflows/ci.yml, bilingual docs/checkpoint.
**Interfaces:** existing real console e2e publishes policy through SCS/CAS; Nginx internal auth_request calls dedicated local API using fixed routing target and sanitized source metadata.
- [ ] Add actual Nginx/HTTP acceptance first; Expected: RED until setup/integration exists, never skip absent tool.
- [ ] Provide minimal documented Nginx config/template and executable test lifecycle; real application count/content proves blocked calls never forwarded. Exercise reused connection target/method/path/source spoofing, source dual-stack decisions, expiry/revoke/deny, authz unavailable, audit records; actual browser publish→proxy→application and revoke.
- [ ] Locked Web typecheck/lint/build and actual Chromium/Nginx CI; full Go/race/vet/build/static retained. Expected: PASS.
- [ ] Fresh whole-branch independent review; resolve blockers with RED→GREEN and full suite; commit final evidence. Verify local/remote/CI exact tree, expected-head merge and green main, release lease; goal remains incomplete.

## Self-review / ledger
Spec coverage checked: all contracts assigned Tasks1–3; route fields/API names consistent; no unowned shared interface. Context map self-reviewed under standing autonomous authority.
Pre-flight Task1 Engine→Task2 Store→Task3 real adapters have explicit signatures; existing publication authority stays singular. No frontend product layout change (existing live console only extended acceptance), so UI design skills reserved for later structured page increment.
Ruling: alpha exact-path source gate before verified identities — delivers real per-request enforcement without inventing trust; cost: no dynamic path matching/user-identity authorization in this slice.
Baseline uncached full Go suite passed; clean existing linked worktree verified. Heartbeat2026-10-18T12:48Z; active lease codex/http-authorization until14:15Z or merged+green main/explicit handoff.
