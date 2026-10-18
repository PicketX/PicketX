# Resource onboarding implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans; native execution, one fresh whole-branch review.

**Goal:** Real resource/Endpoint creation and explicit existing-context Binding creation through API and browser.
**Architecture:** Transform the accepted source, preserving unrelated bytes and facts; reuse publication CAS/audit/compiler/distribution. No second authority.
**Tech Stack:** Existing Go/Echo5/GORM/Casbin/SCS; React/TypeScript/Vite/shadcn/Tailwind/i18next/npm.
**Spec:** docs/superpowers/specs/2026-10-18-resource-onboarding-design.md

## Global Constraints
- Policy bounds 4MiB/4096 objects; mutation JSON16KiB; reason <=512 bytes.
- Names <=128; endpoints/address restrictions1..64; TCP/UDP, ports1..65535.
- Explicit pinned resource+named endpoints+one exact existing host context; no inferred applied state.
- No Agent/protocol/kernel changes or PostgreSQL/identity/approval support claims.
- User explicitly waived repeated routine design/plan/push/merge approvals.

## Context Map
| Files | Role/change |
| --- | --- |
| pkg/configuration/inventory.go, inventory_test.go | Detached inventory and source-preserving resource/binding addition |
| pkg/configuration/types.go, decode.go, compile.go | Existing private schema/strict decode/complete compiler, reference only |
| internal/management/inventory.go | Current access, CAS base, UID generation and existing Replace |
| internal/controller/inventory.go, inventory_test.go | Strict DTO/API/cookie wrappers, real SQLite tests |
| internal/controller/management.go, browser.go | Route registration |
| scripts/normalize-management-api.mjs, docs/api/swagger.json, internal/controller/management_test.go | Generated contract incl deferred endpoint-name constraints |
| apps/web/src/features/resources/Resources.tsx, App.tsx, locales, index.css | Real Resources flow, current styles/components |
| apps/web/e2e/console.mjs | Real browser/API/persistence/Grant acceptance |
| docs/RESOURCE_MANAGEMENT*.md, AUTONOMOUS_DELIVERY*, BASIC_FUNCTION_ROADMAP*, README* | Honest delivery/usage/remaining boundaries |

Dependencies: management.Service.Replace -> publication.Repository; configuration
shared decoder/compiler; console request/SCS/CSRF, existing Grants page and e2e.
Risk: additive public API; no DB migration/runtime config changes. Existing
Binding/Grant lifecycle remains compiler authority. Context map self-reviewed.

## Review Focus
1. Empty marker versus nonempty source; exact preservation of large annotations.
2. Context ID collision across agents; no user-supplied execution contract.
3. Duplicate/empty/null addresses or endpoint names must not widen scope.
4. Concurrent writer/audit failure: no partial resource, binding or facts change.
5. Draft conflict/ambiguous response/viewer: no silent CAS advance or unsafe retry.

### Task 1: Inventory and strict create API
**Files:** configuration/inventory*, management/inventory.go, controller/inventory*, route registrations, API normalization/contract.
**Interfaces:** consumes Snapshot/Replace and Bundle; produces Inventory(p) and CreateResource/CreateBinding(p,expected,reason,input), GET inventory and POST resources/bindings on bearer+cookie namespaces.
- [x] Write TestInventoryResourceAndBindingHTTP: current fixture -> create dual-stack resource -> bind exact existing context -> compiled projection, unchanged facts/audit; missing routes RED.
- [x] Write input/CAS/RBAC/atomic-audit/empty-marker/preservation/context-collision cases; ensure actual rejected writes leave revision/content unchanged.
- [x] Implement detached typed inventory and bounded compact source additions; existing complete Replace is mandatory.
- [x] Generate OpenAPI; regression asserts endpoint_names uniqueItems and item1..1024 on existing Grant API, strict new DTO bounds.
- [x] Run go test ./pkg/configuration ./internal/controller ./internal/management -count=1; expected PASS. Commit.

### Task 2: Real resource onboarding page
**Files:** Resources.tsx, App.tsx, locales/index.css, e2e/console.mjs.
**Interfaces:** consumes Task1 inventory/resource/binding JSON and ETag; produces Resources props csrf/canWrite/onUnauthorized/onDirty/onBusy.
- [x] Add browser create-resource/bind/context-details/Grant-selection flow before page implementation, run native CI and observe missing navigation RED.
- [x] Implement two forms, explicit rebase, preserved drafts, read/error/empty/stale, viewer, bounded lists and existing theme/a11y/i18n.
- [x] Add concurrent mutation412, failed read, viewer denial and375px screenshot/overflow acceptance.
- [x] Run locked npm ci/typecheck/lint/build plus actual Chromium CI; expected GREEN. Commit.

### Task 3: Acceptance and integration
**Files:** focused bilingual docs/checkpoints; all branch changes reviewed.
**Interfaces:** consumes exact tested source and raw CI evidence; produces accepted main/checkpoint.
- [x] Full go test ./..., race/vet/build; local Unix EPERM honestly recorded, required native CI no skips. Full locked Web checks.
- [x] Fresh independent whole-branch review; Important/Critical RED→GREEN fix pass, minors recorded.
- [ ] Verify local/remote/tested tree and final head, required CI logs; expected-head merge; verify main CI, release lease.

## Ledger
Base Core92b2d5f5e1f00c1ae32e219fa1cf9bd45deceee5/tree12e206a694ec648a91738cdbbce0cde69bb2ba72;
mainCI39137171776 GREEN161/423. Agent95a5f7f/CI39118687475 unchanged and GREEN.
No open PRs at acquisition. Existing host-owned isolated worktree reused.
Pre-flight: Task1 JSON/ETag and Task2 consumer match; Task3 exact-source/native gates consume both. No interface conflict.

Task1 local3eaa629: API404 and schema failures observed before corrections; focused
Go packages passed. Task2 local1d272a1/tree36a5b116: native browser39140082943
GREEN after missing-navigation RED39139004218; locked Web checks passed.
Full native Go167/429 zero fail/skip/race; local full runs only UnixEPERM.
Fresh whole-branch review92b2d5f..1d272a1: no Critical/Important; Unicode name
byte-limit feedback Minor deferred. Rulings, screenshot digest, exact heads/raw
CI and scope boundaries persisted in AUTONOMOUS_DELIVERY.md. Final docs-only
head must pass CI, expected-head merge and main CI before Task3 integration is
complete; final immutable evidence appended to PR18 after execution.
