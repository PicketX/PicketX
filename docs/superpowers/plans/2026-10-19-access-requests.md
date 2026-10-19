# Durable Access Requests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:executing-plans.
> User's standing authorization selects inline execution without routine pauses.

**Goal:** Real requester submission, independent approval/rejection and live console.
**Architecture:** Existing publication Store adds versioned SQLite workflow storage
and transactional approval hook; management owns business validation; thin Echo DTOs.
**Tech Stack:** Existing Go/Echo5/GORM/SQLite/Casbin/SCS and React/TS/npm, unchanged.
**Spec:** docs/superpowers/specs/2026-10-19-access-requests.md

## Global Constraints

Base local00c9729 and remote3f06ad3 share tree9e7697f8455aeabd56ffbe11d7c4585abb0f6135.
Host-owned isolated branch is retained; no dirty overwrite/force push. Existing
local Unix EPERM requires real native CI, no skipping. English/Chinese docs sync.
Approval is desired publication, never confirmed application. No Agent code change.

## Review Focus

- Ownership/role forgery and request-only identity leaking policy/facts or peers.
- Transaction failure/concurrent decisions creating approval without one durable Grant.
- Approval after target replacement or expiry broadening frozen intent/deadline.
- Stale UI and unknown write outcomes silently rebasing/retrying permission writes.
- Migration of populated/custom-policy databases and restart preserving provenance.

### Task 1: Deliver durable request workflow vertical slice

**Files:** internal/accessrequest/model.go; publication/{store,requests}.go;
management/access_requests.go; controller/access_requests.go; controllerapp/requests.go;
managementaccess/access.go; related test files; App.tsx and features/requests/Requests.tsx;
locales; e2e/access-requests.mjs; generated OpenAPI; bilingual guides/checkpoints.

**Interfaces:** domain Record/Input/Decision/Page, Store CreateRequest/ListRequests/
GetRequest/DecideRequest, publisher bridge with existing CheckPublication/Publish,
Service RequestCatalog/CreateAccessRequest/AccessRequests/DecideAccessRequest;
HTTP /access-request-catalog and /access-requests[/:uid/decision] on both adapters.

- [x] Write failing HTTP lifecycle tests, real Store atomicity/recovery/CAS tests.
  Run focused Go; expected missing route/capability RED. Implement strict business,
  versioned migrations and transactional publication; expected focused GREEN.
- [x] Add real runtime approval→gRPC/restart test and requester-only isolation tests;
  expected durable independent absolute Grant and no self-approval/duplicate writes.
- [x] Add actual browser workflow. Initial fixture failures were not absent-UI RED;
  real held-read navigation RED and full workflow GREEN are recorded in the review.
- [x] Implement EN/ZH live page and permission-aware shell, preserve all old flows.
  Locked npm ci/typecheck/lint/build expected GREEN; browser true two-person API,
  SQLite/audit/decision/revocation and375/1440 expected GREEN on native CI.
- [x] Full Go normal/race/vet/build; regenerate/check OpenAPI; locked Web checks.
  Record exact local failures, no invented native success. Commit logical changes.
- [x] One fresh whole-branch review; fix blockers RED→GREEN; complete code-head CI,
  compare local/remote/CI trees and raw logs. CI39209796875 passes231/581 tests and
  all native/browser/static gates. Details: validation/ACCESS_REQUESTS_REVIEW.md.
- [ ] Final metadata-only head gates, expected-head merge and main CI; immutable
  final result/lease release are tracked in PR32 (not inferred from code-head CI).

## Context map (reviewed before implementation)

| Files | Responsibility/change |
|---|---|
| publication/store.go,managed.go | Schema4 and approval transaction extension |
| accessrequest/model.go | Framework/ORM-free durable workflow types |
| management/access_requests.go | Own/all checks, frozen scope, bounded approval |
| controller/access_requests.go,browser.go,management.go | Strict DTO/routes/operations |
| controllerapp/requests.go | Serialize commit/delivery with publisher lifecycle |
| managementaccess/access.go | Versioned new operation defaults, deny precedence |
| App.tsx,features/requests | Request-only bootstrap and real workflow states |
| e2e/console.mjs,access-requests.mjs | Actual human identities/publication/SQLite |
| *_test.go,docs/api,scripts/normalize-management-api.mjs | Real behavior/contract gates |

Reference patterns: grants.go explicit source/name/pinned scopes; inventory.go
network view incl addresses; AcceptManaged revisionCAS+audit; browser SCS resolver;
human-login.mjs actual credentials and browser proof. Risk: additive public API,
versioned business/RBAC migrations, no runtime config or dependency changes.

中文：内联执行、先失败后实现、真实端到端及独立全分支评审；验证后安全合并。
