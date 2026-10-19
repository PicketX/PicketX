# HTTP path-prefix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Explicit safe path-segment prefixes through policy, management and real console/ingress.

**Architecture:** Extend shared HTTPRoute selectors inward, expose through existing
CAS publication and inventory. Keep ingress and source authorization unchanged.

**Tech Stack:** Existing Go/Echo5/validator/GORM and React/TypeScript/Vite/i18next/npm.

**Spec:** ../specs/2026-10-19-http-path-prefixes.md

## Global Constraints

- Prefixes match path segments; exact paths remain exact. Explicit `/` is broad.
- Omitted selector lists contribute nothing; explicit empty/null lists are invalid.
- Combined maximum128,16KiB JSON body, no new dependencies/Agent wire/schema migration.
- Preserve Grant union, deny precedence, absolute deadlines, caller and pinned target scope.
- Autonomous inline execution; one fresh whole-branch reviewer; no routine approval pauses.

## Context Map

| Files | Responsibility/change |
| --- | --- |
| pkg/httpauth/engine.go | Validate/clone/union-match new selectors |
| pkg/configuration/{types,decode,candidate,inventory}.go | Strict presence, source roundtrip, inventory |
| internal/controller/http_routes.go | DTO validation and forwarding |
| pkg/configuration/http_test.go; internal/controller/http_routes_test.go | Pure and durable real API regressions |
| apps/web/src/features/http-routes/HTTPRoutes.tsx; src/locales/{en,zh-CN}.json | Separate selector fields/details and errors |
| apps/web/e2e/{http-routes,ingress}.mjs | Real browser/Nginx prefix acceptance |
| docs/HTTP_AUTHORIZATION*.md; docs/openapi/* | Contract/generated schema |

Dependencies: management inventory service already forwards typed routes;
publication compiles complete candidates and persists original source. Strict YAML
rejects all nulls; API strict decoder must preserve same semantics. Existing
console exact-route and removal tests are reference patterns. Risk: additive API
and schema change, no SQL migration; no implicit broadening of legacy documents.
Map self-reviewed against actual imports/consumers before implementation.

## Review Focus

- Prefix lookalikes and encoded dot paths never reach application (Task1/2).
- Legacy exact routes and omitted new fields remain narrow (Task1/2).
- Prefix-only source survives unrelated mutation and SQLite restart (Task1).
- Compiled and inventory lists cannot be externally mutated (Task1).
- Broad root selection is explicit and visible in EN/ZH mobile form (Task2).

### Task 1: Shared selector and durable API contract

**Files:** Shared and Controller files/tests above; HTTP authorization EN/ZH docs.
**Interfaces:** Produces Route.PathPrefixes, InventoryHTTPRoute.PathPrefixes and
CreateHTTPRouteRequest.PathPrefixes ([]string); consumes existing Compile/Decide,
CreateHTTPRoute, Inventory and publication persistence.

- [x] Add failing TestHTTPPathPrefixes boundary/root/legacy/deny/expiry/clone and
  invalid selectors tests; API prefix-only create/read/restart and invalid inputs.
- [x] Run focused Go tests; observe rejection of new field before implementation.
- [x] Implement additive lists, presence checks, clone and segment matching;
  keep existing target/caller authorization and limits. Document EN/ZH contract.
- [x] Run focused normal/race, complete Go normal/race/vet/build; report host
  Unix EPERM honestly and require complete native CI coverage.
- [x] Commit conventional backend feature with its tests.

### Task 2: Real console and ingress acceptance

**Files:** UI/locales/e2e files above; generated OpenAPI and normalized contracts.
**Interfaces:** Consumes Task1 snake_case optional selectors; produces real
prefix-only browser creation and nested Nginx allow/deny/revocation receipts.

- [x] Add real e2e form prefix-only creation, inventory and Nginx boundary,
  malformed path, method/Host/source and reused-connection revoke assertions.
- [x] Observe native tests-first RED on missing prefix field before UI code.
- [x] Add separate optional textareas, combined validation and explicit root
  warning; prefix count/details in inventory and removal. Preserve freshness/CAS.
- [x] Regenerate pinned OpenAPI; locked npm ci/typecheck/lint/build, complete Go
  gates, native real browser/Nginx suites, inspect EN/ZH375/1440 artifacts.
- [x] Commit console feature/tests and record exact native evidence.

### Task 3: Independent review and safe integration

**Files:** docs/AUTONOMOUS_DELIVERY.md and docs/validation/HTTP_PATH_PREFIXES_REVIEW.md.
**Interfaces:** Consumes full base→HEAD diff and exact native source/logs/artifacts.

- [x] Fresh whole-branch review using requesting-code-review; regrade by effect.
- [x] Fix Critical/Important once with observed RED→GREEN; ledger deferred Minors.
- [ ] Verify final local/remote/actual CI trees, all raw mandatory gates and
  independently downloaded archive hashes; exact expected-head merge only.
- [ ] Recheck main source/CI/artifacts; final PR metadata releases short lease
  immediately. Persist remaining Gitea/release scope without claiming completion.

## Self-review

The two implementation tasks share only typed selector fields; nil omission is
kept end-to-end. All five review-focus inputs have explicit tests. No foundation
subproject or static-only UI is introduced. User authorization selects inline
execution and removes routine design/execution/push/merge confirmation pauses.
