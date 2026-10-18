# Safe Resource Removal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Safely remove unreferenced Resource definitions through the real API/UI.
**Architecture:** Shared source-preserving deletion plus explicit reference guard;
existing complete publication transaction and one-draft console, no second authority.
**Tech Stack:** Existing Go/Echo5/GORM/Casbin/SCS; React/TypeScript/Vite/npm.
**Spec:** docs/superpowers/specs/2026-10-19-resource-removal.md

## Global Constraints

- Strict JSON16KiB, reason<=512 bytes, acknowledge_removal:true; UID not name.
- Never cascade; all Grant/Deny/Binding/HTTPRoute references block, even inactive.
- Existing CAS/RBAC/CSRF/audit, full compiler and UTF-8 source preservation mandatory.
- Keep existing visual rules, bilingual copy and no automatic retry/implicit rebase.
- Native complete tests and independent review before expected-head merge.

## Review Focus

- Revoked/expired/dynamic references still block removal (Task1).
- Wrong-kind UID or stale missing target must not bypass ordering (Task1).
- Source formatting and persisted bytes must survive shared helper extraction (Task1).
- Malformed successful receipts retain the exact removal draft (Task2).
- Refresh/rebase while another writer references/removes the target (Task2).

## Context map

pkg/configuration/binding_removal.go owns byte-preserving removal; types.go and
compile.go define pinned references. Add resource_removal.go for reference guards.
internal/management/inventory.go owns structured CAS/Replace operations;
internal/controller/inventory.go and management.go/browser.go own DTO/routes/errors.
scripts/normalize-management-api.mjs plus docs/api/swagger.json own strict OpenAPI.
apps/web/src/features/resources/Resources.tsx owns the single draft and inventory;
locales/en.json/zh-CN.json own copy. apps/web/e2e/resources.mjs owns real browser
resource acceptance (verify filename before edit). RESOURCE_MANAGEMENT bilingual
docs and AUTONOMOUS_DELIVERY.md carry user-facing boundaries and handoff.

### Task 1: Safe Resource removal API

**Files:** configuration resource_removal.go/test; binding_removal.go;
management/inventory.go; controller inventory.go, management.go, browser.go,
resource_removal_test.go, binding_removal_test.go; API generator/schema.
**Interfaces:** Bundle.RemoveResource(uid string)([]byte,error),
Service.RemoveResource(Principal,uint64,string,string)(InventoryChange,error).
- [ ] Add real HTTP tests expecting200 unused removal,409 each reference class,
  404 wrong kind/missing,412 stale,428 absent base, strict400/auth401/403/CSRF403,
  audit-trigger503 rollback and direct SQLite source/facts equality.
- [ ] Run focused tests; expected FAIL missing endpoint404 before implementation.
- [ ] Extract shared source helper; guard references then full Replace; wire routes
  and strict schema. Add source test for explicit-empty replacement and unknown kind.
- [ ] Run focused Go tests/race, complete Go/vet/build; record local Unix limitation.
- [ ] Commit API and tests.

### Task 2: Real Resource removal console

**Files:** Resources.tsx; locale JSON; actual resource e2e module; bilingual docs.
**Interfaces:** consumes Task1 endpoint/receipt; uses existing inventory/CAS draft.
- [ ] Add real browser cases: unused create→cancel→remove; referenced409; stale412
  draft kept, rebase resets acknowledgement; malformed200 receipt blocks retry.
- [ ] Run native browser against pre-UI code; expected FAIL missing Remove Resource.
- [ ] Extend single removal mode with resource target, warning/details and safe
  receipt validation. Preserve Binding behavior; no cloned parallel draft state.
- [ ] Run locked npm ci/typecheck/lint/build and native complete browser; inspect
 375px/1440px screenshots. Expected all pass with no overflow.
- [ ] Commit UI/docs; run one whole-branch independent review and blocking fixes.
- [ ] Verify exact tree/head CI, merge with expected head, verify main CI and release.
