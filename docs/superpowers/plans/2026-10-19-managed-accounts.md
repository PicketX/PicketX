# Managed accounts implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Real database-backed account lifecycle in API/console with durable revocation.
**Architecture:** Framework-free account contracts; managementaccess owns GORM user
authority and Casbin/auth integration; service checks operations; Echo stays thin;
existing React console composes real directory and audit controls.
**Tech Stack:** Existing Go/Echo5/GORM official SQLite/Casbin/SCS/Koanf/Cobra;
React/TypeScript/Vite/shadcn/Tailwind/i18next/npm, no added dependencies.
**Spec:** docs/superpowers/specs/2026-10-19-managed-accounts.md

## Global Constraints
Explicit management.users_mode=file|database, no implicit import/fallback; max128;
bcrypt12/12–72 UTF8 bytes/four work slots; reserved principal:account- random IDs;
one authoritative user source; account version binds sessions; no password/hash
responses/logs/audit; publication/Grant/Agent state unchanged; bilingual live UI.
Standing autonomous authorization covers design/plan/inline/expected-head merge.

## Context Map (reviewed before implementation)
| Files | Responsibility / changes |
| --- | --- |
| internal/account/model.go (new) | Framework/ORM-free directory/input/audit contracts |
| internal/managementaccess/accounts*.go (new) | Versioned authority/CAS/audit/DB validation |
| internal/managementaccess/{access,credentials,password}.go | Mode, RBAC migration, session-version integration |
| internal/management/accounts.go (new) | Permission-checked directory service |
| internal/runtimeconfig/config.go, controllerapp/runtime.go | Strict opt-in typed assembly |
| internal/controller/accounts*.go (new), management.go, browser.go | Strict API and capability discovery |
| apps/web/src/features/accounts/Accounts.tsx (new), App/locales | Real CRUD/unknown/stale/permission UI |
| apps/web/e2e/accounts.mjs (new), console.mjs | Actual lifecycle against second DB-mode Controller |
| scripts/normalize-management-api.mjs, docs/api | Reproducible strict schemas and paths |
| scripts/check-static-controller.sh, docs guides/checkpoint | Version assertions, actual process and handoff |

References: existing private-file human login/password tests, request CAS and UI
writer ownership, publication transaction audits, runtime strict configuration.
Risks: new additive API, versioned identity tables/RBAC migration and explicit
configuration; no existing endpoint semantics/dependencies or GPL boundary change.

## Review Focus
Mixed/stale authority after mode/restart/file changes; concurrent stale edits and
audit atomicity; disable/re-enable cookie resurrection; secret leakage/CPU work
bound; account-only roles and navigation/read/write ownership.

### Task 1: Durable user authority, API and session lifecycle
**Interfaces:** Account{ID,Username,Roles,Disabled,Version}, Change{Username,Password,
Roles,Disabled,Reason}, Page{Mode,Revision,Accounts,Events,ObservedAt}. Access exposes
Accounts() (account.Page,error), SaveAccount(expected uint64, actor,id string,
change account.Change) (account.Account,uint64,error), using id="" for create.
Service exposes Accounts(p), SaveAccount(p,expected,id,change); no ORM/HTTP types.
- [x] Write real SQLite tests for create/CAS/audit/reopen/rollback, DB/file isolation,
  same-hash generation revocation and deny. Run focused tests; expect RED missing methods.
- [x] Implement versioned persistence, settings/credential/session integration and
  service. Run focused packages; expect GREEN (known full baseline Unix EPERM separate).
- [x] Add strict HTTP/Origin/CSRF/Bearer/secret-free/ETag regressions; observe RED
  missing routes then implement and reproduce OpenAPI; expect GREEN.
- [x] Commit and task-done with go test ./internal/managementaccess ./internal/management ./internal/controller ./internal/runtimeconfig.

### Task 2: Real bilingual console and native lifecycle acceptance
**Interfaces:** consumes Task1 routes/page/ETag and me.operations account:read/write.
Produces real Accounts tab, browser fixture and immutable native acceptance.
- [x] Add actual browser lifecycle to console suite; run native tests-only first,
  expect RED missing Accounts control or unfinished behavior, not simulated success.
- [x] Read UI/UX skills and implement established composition with loading/error/
  empty/stale, read vs write ownership, guarded revision and ephemeral secrets.
- [x] Run npm ci/typecheck/lint/build and full Go test/race/vet/build; report any
  baseline Unix EPERM/coalesce failure by name; do not skip native coverage.
- [x] Full CI and real375/1440 evidence; one independent whole-branch review, one
  TDD Important blocker fix pass; exact local/remote/checkout trees and raw logs.
- [x] Commit, task-done and record integration gate in PR33 metadata. Expected-head
  merge, final native CI/main verification and lease release remain mandatory before
  handoff, recorded in final PR metadata to avoid a self-referential checkpoint loop.
  Remaining DB-human Linux conjunction/Gitea/deployment are not claimed complete.

Completion: native exact-head Go/Web gates with actual browser proof and independent
review; final main verification remains required before handoff.

Code acceptance/review: docs/validation/MANAGED_ACCOUNTS_REVIEW.md. Local final
normal/race host Unix EPERM is not a pass; exact unrestricted CI is the full gate.
