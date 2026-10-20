# HTTP Decision Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline; one fresh whole-branch reviewer. Steps use checkbox syntax.

**Goal:** Actual HTTP decision records readable in the authorized console.
**Architecture:** Focused component/pure keyset parser, existing API and auth,
actual browser/Controller/SQLite/Nginx tests; no backend changes.
**Tech Stack:** Existing React/TypeScript/Vite/shadcn/Tailwind/i18next/npm/Go.
**Spec:** docs/superpowers/specs/2026-10-20-http-decision-audit.md

## Global Constraints
- Existing audit:read/SCS, exact200/no-store, after=N&limit=50; bounded200 rows.
- No product Go/API/DB/config/Agent/dependency/stack changes.
-401 clears/signs out;403 clears; failures retain cursor/data but show outdated.
-60s latency/suspension-inclusive age, no hidden reads or stale response update.
- Existing light tokens/44px controls, English/Chinese,375/768/1440/keyboard.
- Keep every Go/Web/install/real-provider/recovery/CLI/native gate, one review,
  expected-head publication/merge and actual archives/raw/main revalidation.

## Review Focus
- Late/aborted responses after logout, permission change or tab switches must not repopulate old identity data.
- Duplicate, backwards, unsafe or inconsistent keyset pages must not skip/repeat records or advance cursor on failure.
-403/401 after a successful read must not retain private decision history.
- Slow/suspended/failed reads must not label old loaded history as complete/current authorization or forwarding proof.
- Long IPv6/IDs/reasons, empty-tail and200-row retention must remain usable by keyboard/mobile without unbounded DOM growth.

### Task 1: Deliver real HTTP decision audit console
**Files:** Spec Context Map.
**Interfaces:** parseHTTPAuditPage(input:unknown,after:number):HTTPAuditPage;
appendHTTPAudit(previous:HTTPAuditEvent[],page:HTTPAuditPage):HTTPAuditEvent[]
retains last200; HTTPAudit({onUnauthorized:()=>void}) component.
- [ ] Write model tests first: exact page/cursor,50limit,200window, emptytail,
  booleans/time/unsafe/reordered/duplicate/privateextras; run
  node --test apps/web/e2e/http-audit-model.test.mjs; Expected missing-model RED.
- [ ] Write actual browser initial-empty/missing-subview test and publish test-first
  draft; actual CI must show missing HTTP decisions button before UI implementation.
- [ ] Implement minimal parser/window, same Node command Expected all PASS.
- [ ] Implement component and audit subview/capability mount; scoped pairedcopy/CSS.
  Browser tests use actual persisted allow/deny/IPv6/Grant/deny IDs and205 actual
  API decisions for pagination/window. Test loading/failed503/202/403/late unmount/
  stale clock plus actual credential removal401, desktop/mobile screenshots.
- [ ] Run model fullNode, lockedWeb ci/typecheck/lint/build and complete Go normal/
  race/vet/build; explicit inherited local UnixEPERM not nativegreen. Full real
  browser/Linux/install/provider gates in CI mandatory. Expected clean supported
  local checks and actual native success, no ignored errors or skipped scenarios.
- [ ] Update bilingual docs/checkpoint and commit. Whole-task command:
  node --test apps/web/e2e/http-audit-model.test.mjs apps/web/e2e/wait-for-operation.test.mjs
- [ ] ONE independent full-branch review; root ONE blocker RED→GREEN pass; defer
  minors and persist exhaustive rulings/costs. No extra reviewer/gate bypass.
- [ ] Verify final head/tree/fullCI/raw/fouractualarchives/static/notices/screenshots;
  expected-head merge, main repeat, immediate release and concrete next step.

Self-review: one task/no shared interfaces; all five focus classes pinned by model
or actual browser cases. Explicit user autonomy waives routine handoff pauses only.
