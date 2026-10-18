# Resource network editing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Correct unreferenced Resource network parameters through API and live console.
**Architecture:** Shared configuration lifecycle guard, existing managed transaction,
thin Echo adapters; extend existing single Resource draft.
**Tech Stack:** Existing Go/Echo5/GORM/Casbin/SCS and React/TS/Vite/shadcn/Tailwind/i18next/npm.
**Spec:** docs/superpowers/specs/2026-10-19-resource-edit.md

## Global Constraints

- Keep Resource UID/name/metadata and exact Endpoint names, type and labels.
- Only network values change; input reordering must not reorder Endpoints.
- Any Grant, NetworkDeny, EnforcementBinding or HTTPRoute reference blocks.
- Preserve facts/unrelated source; selected document normalizes.
- Exact200, matching UID, increasing safe revision and matching ETag required.
- No dependency/schema/kernel change. Publication is not application.

## Review Focus

- Imported metadata/Endpoint labels and reversed input order must survive.
- Revoked, expired and zero-match references still prevent retargeting.
- Concurrent references/same-name replacement must not bypass CAS or UID.
- Incomplete/malformed responses must preserve draft and block retry.
- Rebase exposes latest values, retains intent and resets acknowledgement.

## Context Map (reviewed)

| Files to modify/create | Purpose |
| --- | --- |
| pkg/configuration/resource_edit.go + tests | Guard reuse; network-only edit |
| internal/management/inventory.go | Atomic grantBase/Replace operation |
| internal/controller/resource_edit.go + tests | Strict DTO/auth/CAS/status |
| internal/controller/{management,browser}.go | Register bearer/session routes |
| docs/api and existing OpenAPI generation | Document explicit contract |
| apps/web/src/features/resources/Resources.tsx, locales | Live draft/recovery |
| apps/web/e2e/resource-edit.mjs, console.mjs | Browser acceptance |

Dependencies: InventoryEndpoint/InventoryChange, RemoveResource/appendObject,
Replace, decodeGrantJSON, request(status). Patterns: Resource create/remove and
resources.mjs. Risks: additive API; no migration/config change; reference guard
intentionally narrows editing rather than retarget existing authorizations.

### Task 1: Safe network edit API

**Files:** configuration/resource_edit.go+test; management/inventory.go;
controller/resource_edit.go+test, management.go, browser.go; OpenAPI generator/output.
**Interfaces:** Bundle.UpdateResourceNetwork(uid string,entries []InventoryEndpoint)
([]byte,error); Service.UpdateResourceNetwork(p Principal,expected uint64,reason,uid
string,entries []configuration.InventoryEndpoint)(InventoryChange,error).
- [ ] Write TestResourceNetworkEditHTTP:200/revision/UID, direct SQLite network/facts/
  metadata retention. Table-test references409, missing404, stale412, viewer403,
  CSRF403, invalid400, audit rollback503. Imported labels/reordering retained.
- [ ] Run `go test ./internal/controller -run TestResourceNetworkEdit -count=1`;
  expected missing-route404 RED.
- [ ] Implement transformation, service, strict adapters/registration. Reject changed
  Endpoint membership. Generate/validate pinned OpenAPI and test endpoint security.
- [ ] Run `go test ./pkg/configuration ./internal/controller ./internal/management`;
  expected PASS. Commit feat(management): edit unreferenced Resource networks.

### Task 2: Live edit workflow and integration

**Files:** Resources.tsx, locales, e2e/resource-edit.mjs/console.mjs, bilingual docs.
**Interfaces:** Consume Task1 PUT network endpoint/200 InventoryChange/ETag.
- [ ] Write browser tests: edit actual Resource, cancel unchanged, save verified API/
  audit; referenced409; stale412 then rebase preserving fields/reset acknowledgement;
  invalid202/receipt recovery; missing UID; viewer;375px/1440px screenshots. Run native
  before UI, expected missing edit control RED. Keep real Nginx tests mandatory.
- [ ] Add edit mode, read-only identities, latest preview, acknowledgement, exact
  receipt and missing-target retry lock. No new visual system/dependencies.
- [ ] Run npm ci/typecheck/lint/build and native full Go/race/vet/build/browser CI;
  expected PASS without skip. Commit feat(web): edit Resource network parameters.
- [ ] Fresh independent whole-branch review; blockers RED→GREEN, exact head/tree/raw
  CI verification, expected-head merge/main CI. Record handoff and release lease.

中文：分 API 安全编辑和真实页面两段，先观察失败测试，再实现并验证；最后独立
全分支评审、准确源码树/CI 核验和安全合并。保留引用限制及后续未完成项。
