# HTTPRoute console implementation plan

> REQUIRED SUB-SKILL: superpowers:executing-plans. Inline implementation, one fresh whole-branch review.

**Goal:** Structured exact HTTPRoute creation/read with actual request authorization.
**Architecture:** Append to existing publication; shared compiler remains authority.
**Stack:** Existing Go/Echo5/GORM/Casbin/SCS and React/TypeScript/Vite/i18next/npm.
**Spec:** docs/superpowers/specs/2026-10-19-http-route-console-design.md

## Global Constraints
- Strict mutation JSON16KiB; policy4MiB/4096 objects; reason512 bytes.
- Name128 bytes; host253; methods1..7; exact paths1..128, each8192 bytes.
- Pinned resource UID, existing TCP endpoint, server-generated route UID.
- Preserve facts, existing source and single database authority; no Agent changes.
- Current publication:read/write, strong CAS, cookie CSRF, atomic audit.
- User waived routine design/plan/push/merge confirmations; never bypass gates.

## Context Map
| Files | Purpose |
| --- | --- |
| pkg/configuration/inventory.go, types.go, candidate.go; pkg/httpauth | Detached route inventory/addition; reference strict compiler/selector semantics |
| internal/management/inventory.go | grantBase/random UID/Replace; preserve facts |
| internal/controller/http_routes.go, http_routes_test.go, management.go, browser.go | Strict DTO and bearer/cookie routes, real SQLite tests |
| scripts/normalize-management-api.mjs, docs/api/swagger.json, management_test.go | Generated bounded public contract |
| web/src/features/http-routes/HTTPRoutes.tsx, App.tsx, locales | Existing theme, real catalog/create, CAS/draft lifecycle |
| web/e2e/console.mjs, http-routes.mjs, ingress.mjs | Real browser/API/Nginx/application acceptance |
| docs/HTTP_AUTHORIZATION*, AUTONOMOUS_DELIVERY* | Usage, evidence, remaining scope and lease |

Dependencies: Inventory/Replace -> publication.Repository -> shared compiler;
console request/SCS/CSRF and real ingress fixture. Self-reviewed additive API:
no migration, new authority, runtime config, protocol or kernel changes.

## Review Focus
1. Pinned TCP endpoint and exact selectors: invalid-input table/no revision change.
2. CAS/audit rollback/persistence: concurrent stale write and SQLite reopen tests.
3. Route does not imply source access: real matching/mismatched/revoked request tests.
4. Dirty/ambiguous/read-only form: browser412/rebase/read failure/viewer cases.
5. Caller UID binding is explicit: real Nginx pinning, no client-controlled route scope.

### Task 1: Strict route API
**Files:** configuration/management inventory, controller routes/tests, OpenAPI.
**Interfaces:** Inventory.http_routes; CreateHTTPRoute(p,expected,reason,input); POST http-routes bearer/cookie.
- [x] Observe API404 RED, implement minimal source addition through complete Replace.
- [x] Test invalid/null/unknown/duplicate/pinning/TCP/selectors, CAS, RBAC, CSRF, audit/reopen.
- [x] Generate and test OpenAPI; focused Go suite GREEN; commit.

### Task 2: Real route page
**Files:** HTTPRoutes/App/locales; e2e console/http-routes/ingress.
**Interfaces:** Task1 JSON/ETag; csrf/canWrite/onUnauthorized/onDirty/onBusy props.
- [x] Observe missing navigation RED in actual native browser before UI implementation.
- [x] Implement explicit selectors, catalog/status/draft guards and English/Chinese parity.
- [x] Exercise real browser creation/CAS/errors/viewer/mobile and Nginx source grant/revoke.
- [x] Locked npm ci/typecheck/lint/build and native e2e GREEN; commit.

### Task 3: Acceptance
**Files:** full branch and focused bilingual checkpoint/docs.
**Interfaces:** exact reviewed source -> expected-head merge -> verified main.
- [x] Full Go/race/vet/build and locked Web; local UnixEPERM recorded, native no skips.
- [x] One fresh independent whole-branch review; fix blockers, ledger all deferred minors.
- [ ] Exact local/remote/tested tree, required raw CI, expected-head merge and mainCI; lease release.

## Ledger
Base Core main38e4f93c02a1c1fb72131b612a315c2fd6207a44/treeed5135eff3f2c1ad09980f13846f55729786a84c,
mainCI39142252279 accepted167/429, zero fail/skip/race. Agent95a5f7f/CI39118687475
unchanged green. No open PRs at acquisition; PR19 lease released. Existing clean
host-owned isolated worktree reused (local94d615e has identical base tree).
Baseline full Go ran: only TestAuthorizationRuntimeTCPUnixAndShutdown/unix
fails on sandbox socket EPERM; no required scenario skipped. Pre-flight task
interfaces are compatible and bounded; overall product not complete.

Task1 local7396903: API404 and undocumented OpenAPI observed RED→GREEN; full
controller/configuration packages pass. Task2 local84588c8/remote f43fcdf and
actual CI checkout23c3ab1 share tree708ae952da90843ce3980e671876ff5c96d60f3c.
Browser RED39146482257/Web117731241046 (missing navigation) → GREEN
39147016947/Web117732810602. Real route UID→explicit caller/Nginx→application,
source deny/grant/revoke, exact selectors, CAS412/draft/read-error/viewer/mobile
passed. Screenshot artifact12143592283 digest
a9e4bdcac115e28fd914ade466a8a186ca5a9efc435cb6b9b92cfc7f6c4195c0 verified;
desktop,375px create/error and viewer images inspected. Native Go still running
at this entry; local full ordinary/race fail only known UnixEPERM, vet/build
and locked Web passed. Full branch review/merge/main acceptance remain pending.

Native Go117732810809 subsequently GREEN172/434, zero fail/skip/race, actual
Unix100-run stress and static Controller/CLI gates passed. Fresh review of
94d615e..2840135 requested; implementation unchanged during review.

Final review: independent fresh read-only review94d615e..2840135 READY TO MERGE;
no Critical/Important. Detailed findings/rulings in AUTONOMOUS_DELIVERY.md.
Final: minor (deferred): pristine target catalog/tab activation freshness; explicit rebase works.
Final: minor (deferred): direct browser faults for committed/lost POST response and post-success read failure.
Final: Ruling: caller/ingress setup stays separate trusted authority — explicit operator setup required.
Final: Ruling: edit/delete stays advanced configuration — incomplete structured lifecycle.
Final: Ruling: verified identity/OIDC/approval remains separate — no user self-service.
Final: Ruling: live applied/kernel/TLS/active-stream termination unchanged — no combined production acceptance.
Only final docs changed after review. Exact final head/native/main CI evidence
will be appended to PR20; no recursive documentation-only acceptance loop.
