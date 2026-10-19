# Binding Endpoint editing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Safely edit existing named host Binding protection in the live console.
**Architecture:** Configuration transform, managed transaction, thin Echo adapter,
additive inventory projection and existing single-draft Resources UI.
**Tech Stack:** Existing Go/Echo5/GORM/Casbin/SCS; React/TS/Vite/shadcn/Tailwind/i18next/npm.
**Spec:** docs/superpowers/specs/2026-10-19-binding-endpoint-edit.md

## Global Constraints

- Keep identities, Resource/Agent/enforcement scope and independent authorizations.
- Explicit names only; no dynamic or multi-target convenience edits.
- Whole compile, authenticated CAS, atomic audit; publication is not application.
- Preserve unrelated source and facts; no new dependencies/schema/kernel changes.
- Retain draft on failure; exact200 receipt; rebase must not silently move scope.

## Review Focus

- Metadata, Grants/Denies and IPv4/IPv6 scope survive the transform.
- Unsupported selectors cannot be narrowed accidentally by the convenience API.
- Stale revision cannot change the latest target; invalid compile/audit roll back.
- Rebase resets acknowledgement and blocks changed scope/missing targets.
- Ambiguous receipts never clear a draft or enable unsafe automatic retry.

## Context Map (self-reviewed)

| Files | Role |
| --- | --- |
| pkg/configuration/binding_edit.go, inventory.go | Named-only transform/projection |
| internal/management/inventory.go | grantBase and transactional Replace |
| internal/controller/binding_edit.go, management.go, browser.go | Strict DTO/routes/errors |
| controller tests, docs/api, normalize-management-api.mjs | Persistence/auth/contract |
| Resources.tsx, locales | Single draft and scope-safe rebase |
| e2e/binding-edit.mjs, console.mjs | Real browser/API/audit acceptance |

Reuse removeDocument/appendObject and InventoryChange. Binding raw selectors and
enforcement remain unchanged. No migration or Agent code. Risks: removing
protection differs from Grant revocation; UI must explicitly explain this.

### Task 1: Binding Endpoint edit API

**Interfaces:** Bundle.UpdateBindingEndpoints(uid string,names []string)([]byte,error);
Service.UpdateBindingEndpoints(p Principal,expected uint64,reason,uid string,names
[]string)(InventoryChange,error); optional InventoryBinding.endpoint_edit projection.
- [ ] HTTP/persistence tests first: correct receipt, canonical names, retained
  metadata/facts/Grants/Denies, stale412, unsupported409, missing404, invalid400,
  auth/CSRF403 and audit rollback. Run focused tests; expect missing route RED.
- [ ] Implement transform/service/strict handlers/OpenAPI. Run configuration,
  controller and management suites; expect PASS except documented local Unix EPERM.
- [ ] Commit API implementation and generated contract.

### Task 2: Live console workflow and delivery

**Interfaces:** Consume Task1 projection/PUT200 receipt; no new API contract.
- [ ] Add browser acceptance before UI: real named binding, cancellation,
  acknowledged save, SQLite/API/audit, stale/rebase, changed-scope lock,
  ambiguous receipt, viewer and375/1440 screenshots. Native RED expected.
- [ ] Implement fixed-scope editor, reason/acknowledgement, guarded rebase and
  precise receipt handling in existing visual system. Run locked Web checks.
- [ ] Full Go normal/race/vet/build and native Linux/Web/browser/Nginx gates.
- [ ] One fresh independent whole-branch review; fix blockers RED→GREEN; verify
  exact local/remote trees and raw CI, expected-head merge, main CI and release.

中文：先 API 安全变更，后真实页面；保留完整作用域和授权语义，必须先失败后通过，
再独立评审和安全合并。整体管理端/身份审批/联合 Linux 验收未完成，不宣称产品完成。
