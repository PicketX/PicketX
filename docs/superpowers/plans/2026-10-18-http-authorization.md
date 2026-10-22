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
| web/e2e/console.mjs, .github/workflows/ci.yml | real acceptance | Nginx/browser/application |
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
- [x] Write failing actual Decode/Compile/Decide tests: no route/unknown, source IPv4/IPv6, overlapping independent deadlines, future/revoked, explicit deny, exact Host/method/path, malformed/encoded URI, pinned references and no route UID retarget.
- [x] Run focused tests; Expected: missing implementation RED.
- [x] Implement strict route/context and configuration integration; unchanged network projection.
- [x] Run focused and full `go test -count=1 ./...`; Expected: PASS; commit.

### Task 2: Durable scoped local decisions
**Files:** internal/{publication,authz,controller,controllerapp,runtimeconfig}, generated OpenAPI and tests.
**Interfaces:** Store.DecideHTTP(caller string,q httpauth.Query) (HTTPDecision,error), Store.HTTPAudit(after int64,limit int); authz.Service.Decide(token string,q httpauth.Query), dedicated controller.NewAuthorizationHandler; runtime assembly typed authz config.
- [x] Write failures for full runtime TCP+Unix, scope/credential rotation, strict JSON/header duplicates, schema2 upgrade/reopen, audit failure with no allow, RBAC audit visibility, startup-existing Unix object and shutdown.
- [x] Run tests; Expected: missing routes/config/schema RED.
- [x] Implement official Echo/validator adapters, private credential validation, schema3/evaluate+audit, isolated listeners/readiness and bounded lifecycle; document/generate both API groups.
- [x] Full Go ordinary/race/vet/build, pinned generation, local OpenAPI; Expected: PASS; commit (full native suite in Linux CI; local Unix EPERM retained).

### Task 3: Real Nginx/browser/application acceptance
**Files:** examples/http/*, web/e2e/console.mjs, .github/workflows/ci.yml, bilingual docs/checkpoint.
**Interfaces:** existing real console e2e publishes policy through SCS/CAS; Nginx internal auth_request calls dedicated local API using fixed routing target and sanitized source metadata.
- [x] Add actual Nginx/HTTP acceptance first; Expected: RED until setup/integration exists, never skip absent tool.
- [x] Provide minimal documented Nginx config/template and executable test lifecycle; real application count/content proves blocked calls never forwarded. Exercise reused connection target/method/path/source spoofing, source dual-stack decisions, expiry/revoke/deny, authz unavailable, audit records; actual browser publish→proxy→application and revoke.
- [x] Locked Web typecheck/lint/build and actual Chromium/Nginx CI; full Go/race/vet/build/static retained. Expected: PASS.
- [ ] Fresh whole-branch independent review; resolve blockers with RED→GREEN and full suite; commit final evidence. Verify local/remote/CI exact tree, expected-head merge and green main, release lease; goal remains incomplete.

## Self-review / ledger
Spec coverage checked: all contracts assigned Tasks1–3; route fields/API names consistent; no unowned shared interface. Context map self-reviewed under standing autonomous authority.
Pre-flight Task1 Engine→Task2 Store→Task3 real adapters have explicit signatures; existing publication authority stays singular. No frontend product layout change (existing live console only extended acceptance), so UI design skills reserved for later structured page increment.
Ruling: alpha exact-path source gate before verified identities — delivers real per-request enforcement without inventing trust; cost: no dynamic path matching/user-identity authorization in this slice.
Baseline uncached full Go suite passed; clean existing linked worktree verified. Heartbeat2026-10-18T12:48Z; active lease codex/http-authorization until14:15Z or merged+green main/explicit handoff.

Task1 complete8878876: missing package/Candidate.HTTP RED→GREEN; exact source/request, IPv4/IPv6, independent/future/revoked Grants and deny/invalid-route/retarget tests; uncached complete Go suite passed. Native skill workspace/briefs/ledger are now available and used.
Task2 active: store/schema/API tests RED→GREEN, full audit/reopen/fault and current management RBAC passed. Initial test fixtures incorrectly used nonprivate parent and simulated v1 while retaining new v3 table; corrected fixtures to real private/legacy shape, no production validation weakened. Local actual TCP listener passes. Local Unix socket creation returns EPERM; required test remains failing locally, never skipped. Real Linux CI is mandatory acceptance for that listener, not a mock claim.

Task2/3 code evidence: local1f0ed13 tree0375ef7 equals remote6495222. CI39129578891 Go job117681813833 green149 top-level/399 total, zero fail/skip/race; actual Unix creation0600/callback/shutdown/cleanup, complete race/vet/build/generation/OpenAPI/static Controller/CLI gates pass. Web job117681813678 RED on overstrict malformed-error connection retention, not an authorization bypass. Correction6af9670/remote6b5a70a treeafbe209 verified; actual CI checkoutd7bfcf1f09570750d0a8e38dcd2924512cc1ef93. Web39129977666 job117682983128 GREEN: actual Nginx1.24.0/auth_request, Chromium→real console→source decision→actual HTTP application, keepalive target/Host/method/path/source spoof isolation, revoke/absolute deadline, IPv6 JSON context, durable audit and caller removal/malformed fail-closed. Complete first console flows retained. Full Go still running; this reviewed code has a blocking deadline finding below, so not accepted for merge.
Ruling: required native Unix/Nginx/Chromium execution is supplied by Linux CI instead of unavailable local capabilities; Tasks2/3 verification converge at that gate — real source/test failures remain failures, no skip — cost: remote CI is the single full integration execution environment.
Ruling: malformed callback errors may close a connection; retain strict same-socket assertions for valid unauthorized target/Host/method requests, check encoded-path failure separately — contract is fail-closed, not transport preservation — cost: no promise to retain malformed-request connections.
Final review /root/http_review covered74575a4..6af9670; no Critical/Minor, one Important stale decision time across publication/SQLite waits. Root reproduced expiry and scheduled deny activation behind both real mutex and actual DB sole connection: four RED Allowed:true after boundary. Fix refreshes UTC after lock and re-evaluates/re-audits after durability if NextChange passed, bounded2 then unavailable. All four GREEN; focused uncached race passed. Complete exact-head suite and final CI pending; no second review.
Final rulings on declined judgments:
- Remote CI/merge/main: executor verifies exact head/tree/raw evidence; cost one integration verifier, reviewer did not repeat.
- Nginx/Chromium: actual CI harness (not reviewer-local) is evidence; cost no independent browser rerun.
- Unix create/cleanup: retain mandatory real CI, local EPERM never acceptance; cost unvalidated local deployment environment.
- Complete Go/Web/static/generated gates: executor/CI must run all; cost reviewer independently ran focused suites only.
- Production HTTPS/load-balancer/real_ip/upstream bypass: deployment-owned prerequisites documented, not loopback acceptance; cost real deployment still needs verification.
- Verified humans/cookie propagation/native DB/SSH AuthZ: unsupported/rejected, source/caller never substituted; cost narrower alpha policy capability.
- Agent/kernel/combined workflow: unchanged PR16 baseline only; cost no new combined Linux workflow acceptance yet.
- Existing streams/upgrades/in-flight work: not terminated by per-request gate, disable upgrades if required; cost no stream cancellation guarantee.
- HA/crash beyond SQLite/latency/capacity/contention/retention: unmeasured/no operational promise; cost storage/capacity/retention must be planned before scale.
- Query-sensitive actions/prefix/wildcards/alternate ingress: exact-path only; cost these deployments cannot infer safe authorization from example.
- Malicious local admin/compromised trusted ingress: trusted authority controls context/mapping/key, out of threat boundary; cost no defense against compromised trusted integrator.

Task2 complete / Task3 code acceptance: fixed local8cb7ece treea2932abdfd22ac650d9a7abc562bee4706569779 equals remotefe43e0772de1ec54b40cfa10fb0e9cf0188d1d08 and fetched CI checkout55f510aad6db8f03a297aa0d1a6d1429d270fd63. CI39130425080 Go117684282808/Web117684283026 both success. Full raw150/405, zero fail/skip/race, including actual valid IPv4/IPv6 TCP and Unix decisions/cleanup and all four deadline regressions. Full static/generation/local OpenAPI gates and actual Chromium/Nginx source-Grant/explicit-deny/revoke/deadline/application/audit flow GREEN. Artifact12137651861 SHA256168e64403b3081383f3f9f94ece5dbde4ad6dfe15ae202ef38ec847650ac4abc independently verified; application/mobile screenshots inspected. No deferred minors or unresolved finding. Local all-suite Unix EPERM remains environment-specific negative evidence, not claimed passing; all required actual Linux cases passed. New real runtime fixture initially used deliberately expired static example dates; fixed only the test fixture to future independent absolute deadlines, not production validation.
Final: fixed stale-clock Important finding — TestHTTPDecisionRefreshesTimeAcrossQueueAndAudit four cases RED→GREEN, complete Linux race150/405 GREEN. Only final metadata exact-head CI, expected-head merge and main verification remain; final results will be appended durably to PR16. Lease auto-releases only after that acceptance. Product remains incomplete, next structured management/approval/status and final combined identity/Linux workflow.
