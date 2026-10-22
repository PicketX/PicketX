# HTTPRoute Removal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline; one fresh whole-branch reviewer before integration.

**Goal:** Safely retire a route through the real API/console and prove ingress fails closed.
**Architecture:** Reuse source-local document removal and managed full Replace. Extend
the existing route page's single CAS draft; no new authority or runtime integration.
**Tech Stack:** Existing Go/Echo5/GORM/Casbin/SCS; React/TypeScript/Vite/i18next/npm.
**Spec:** docs/superpowers/specs/2026-10-19-http-route-removal.md

## Global Constraints
- Strict JSON16KiB {reason, acknowledge_removal:true}; strong If-Match and existing auth/CSRF.
- No cascade, caller configuration changes, Grant revocation or stream-termination claim.
- Preserve facts/unrelated bytes; whole-candidate compilation and atomic audit remain mandatory.
- User preauthorized design and inline execution; no routine approval pauses.

## Context Map
| File | Purpose / change |
| --- | --- |
| pkg/configuration/http_route_removal.go | UID+kind lookup, shared removeDocument |
| internal/management/http_route_removal.go | grantBase, full Replace, typed not-found |
| internal/controller/http_route_removal.go | strict DTO, bearer/session HTTP adapters |
| internal/controller/{management,browser}.go | routes and error mapping |
| web/src/features/http-routes/HTTPRoutes.tsx | one-draft removal/rebase/receipt handling |
| web/src/locales/{en,zh-CN}.json | bilingual labels/errors |
| scripts/normalize-management-api.mjs, docs/api/swagger.json | reproducible strict contract |
| internal/controller/http_route_removal_test.go | actual HTTP/SQLite security/atomicity |
| web/e2e/{http-routes,http-route-removal}.mjs | actual browser/ingress acceptance |

Dependencies/reference patterns: configuration/binding_removal.go, resource_removal.go;
management/inventory.go and controller/resource_removal_test.go; resource-removal.mjs.
Risk: additive public endpoint; no migration/config requirement; denied missing route
must be checked through actual ingress, not inferred from a deleted list item.

## Review Focus
- Wrong-kind UID must not remove a Resource or Binding (Task1 literal404).
- Failed audit must retain persistent source/facts/revision (Task1 SQL assertion).
- Stale route draft must not delete a newer same-name UID (Task2 external deletion/rebase).
- Invalid receipt must not discard intent or auto-retry (Task2 faulted200/204).
- Existing source Grant must not bypass missing HTTPRoute (Task2 real ingress denial).

### Task 1: API/source removal
**Interfaces:** Produce RemoveHTTPRoute(uid string)([]byte,error) on Bundle and
Service.RemoveHTTPRoute(p Principal,expected uint64,reason,uid string)(InventoryChange,error).
Consume existing grantBase/Replace/removeDocument. Browser consumes200{uid,revision}/ETag.
- [ ] Write TestHTTPRouteRemovalHTTP/ValidationAndRollback/CookieCSRF: seed real route,
  remove exact UID200, wrong Resource UID404, stale412, absent404, strict/auth rejections,
  audit-trigger503 preserves direct SQLite policy/facts/revision and audit count.
- [ ] Run go test ./internal/controller -run TestHTTPRouteRemoval -count=1; expect missing route404 RED.
- [ ] Implement focused helper/service/adapter and register routes; normalize generated schema.
- [ ] Run full configuration/controller/publication tests; expect PASS. Run complete Go suite,
  race/vet/build; native CI must cover any documented local Unix EPERM, never skip it.
- [ ] Commit feat(management): remove HTTP routes with revision and audit protection.

### Task 2: Route console and real ingress verification
**Interfaces:** Consume Task1 endpoint and existing inventory; produce explicit removal
draft mode on HTTPRoutes. Reuse ingress.verifyExactRoute and actual API fixture.
- [ ] Write browser tests before UI: Remove HTTP route missing RED. Check cancel permits,
  active Grant remains after removal but request denies, independent sibling permits;
  wrong receipt retains draft, stale412 requires explicit rebase/reset, absent UID blocks.
- [ ] Run native Web suite on test-only head; expect missing Remove HTTP route control.
- [ ] Implement confirmation/reason/ack/cancel and strict receipt validation, bilingual
  feedback; preserve single draft and existing loading/error/empty/stale behavior.
- [ ] Run npm ci/typecheck/lint/build and entire native real browser suite; expect PASS;
  download/digest-check screenshots and visually inspect375px/1440px.
- [ ] Commit feat(web): safely retire HTTP routes from console.

Final: independent full-branch review, one TDD fix pass for blockers, all exact-head
CI/raw evidence/tree checks, expected-head merge, main verification, durable handoff.
