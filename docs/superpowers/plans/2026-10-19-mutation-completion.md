# Management mutation completion implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Never announce completion for an unexpected management-write status.
**Architecture:** Optional explicit expected success status in shared transport;
callers supply endpoint contracts, preserving existing domain receipt checks.
**Tech Stack:** Existing React/TypeScript/Vite/i18next/npm; Go/Echo5 unchanged.
**Spec:** docs/superpowers/specs/2026-10-19-mutation-completion.md

## Global Constraints

No dependency/schema/auth/kernel changes; preserve single authority, absolute
expiry, IPv4/IPv6 and desired-versus-applied distinction. Retain current UI style.
User preauthorizes design/implementation/push/merge; no routine approval pause.

## Review Focus

- Valid-looking 202 body must not clear draft or show success.
- 204 must fail before undefined data reaches a write consumer expecting JSON.
- Existing non-2xx APIError codes, especially 401/412, retain semantics.
- Publication unknown/5xx outcomes cannot permit blind repeat with old CAS base.
- Expected success and existing removal UID/revision/ETag checks remain enforced.

### Task 1: Exact completion status and real form recovery

**Files:** All paths in the spec context map; delivery checkpoint.
**Interfaces:** Consumes existing endpoint statuses; produces
`request<T>(path: string, init: RequestInit = {}, expectedStatus?: number)` with
unchanged result `{data: T, etag: string, status: number}`.

- [ ] Write `e2e/request.mjs`: unexpected success statuses reject ordinary Error;
  exact200/201 preserve data/ETag/status;204 explicit success remains valid;
  non-2xx APIError status/code and GET behavior unchanged. Run Node: expected
  RED missing rejection before production changes.
- [ ] Add browser fault assertions before existing real successful writes:
  publication, Resource/Binding create/remove, Grant issue/revoke, route create.
  Inject202 valid-looking receipts; assert retained intent, no success and retry
  lock. Explicit rebase restores eligibility; actual existing API flow completes.
- [ ] Publish tests on leased draft PR; native browser should fail before fix.
- [ ] Implement exact status guard and explicit management callsite arguments;
  publication unknown/5xx catch locks retry. Keep existing body checks.
- [ ] Run local Node regression and locked npm ci/typecheck/lint/build: expected
  GREEN. Run full native CI including Go race/vet/build and real browser suite.
- [ ] Commit; run task-done using `node apps/web/e2e/request.mjs` after complete
  native suite evidence is read. Independent whole-branch review; fix blockers
  RED→GREEN, verify exact tree/head/checks, merge safely and verify main CI.
