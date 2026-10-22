# Capability Refresh Investigation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans inline;
> one independent whole-branch review before integration.

**Goal:** Identify the actual /me-to-browser boundary losing custom capability.
**Architecture:** Retain real Controller/SQLite/Casbin/Chromium; capture allowlisted
public response evidence and assert every repeated reload, never retry a failure.
**Tech Stack:** Existing Go, React19/TypeScript/Vite/i18next/npm/Playwright.
**Spec:** docs/superpowers/specs/2026-10-20-capability-refresh.md

## Global Constraints
- BaseCore85fa14ef/tree522c105; Agent5373595 unchanged; no secrets in diagnostics.
- No policy widening, rate-limit/reset bypass, fake response, skipped assertion,
  dependency change, source/config schema change or UI restyling.
- Root-cause evidence must precede a production repair and its RED→GREEN proof.
- Native full Go/race/vet/build, locked Web, browser/install/service remain gates.

## Review Focus
- A 200 login response may precede the UI's own completion load.
- A permission change between /me observations must not be claimed a stable view.
- Diagnostic response bodies can contain secrets; project allowlisted fields only.
- Repeated reloads are assertions, not retries that hide a failing iteration.
- Failure diagnostics/cleanup must not replace the original browser exception.

### Task 1: Real boundary evidence and repeat assertions
**Files:** web/e2e/access-requests.mjs; bilingual checkpoints/spec.
**Interfaces:** Existing accessRequests({...}) unchanged; real /me response listener
stores at most16 records of status/id/roles/operations (no arbitrary payload);
every actual reload must show Requests and exact custom capability.
- [ ] Compare unchanged real Access/SQLite under200reloads/12000checks, then mixed
  identities under race; passing characterization is not a repair or TDD RED.
- [ ] In existing real browser flow add bounded public response evidence and12
  strict reload assertions; output the allowlisted history on failure, rethrow.
- [ ] Run native full CI, compare exact actual responses to final UI. If a specific
  culprit is identified, extend this plan with the exact regression/fix before code.
- [ ] Complete one independent review, final exact-source CI/merge/main evidence.

Self-review: all5focuses map to listener bounds/allowlist, strict per-iteration UI
and capability assertions, original-error rethrow and unchanged production policy.
The plan is intentionally investigation-only until evidence identifies the fix;
no unknown production implementation is authorized by an unspecified placeholder.

### Task 2: Explicit awaited readiness (root-cause extension18:48Z)
Pinned Playwright coreBundle.js25672..25736 tests predicate()'s Promise for truth,
then adopts its false result without repolling. This explains initial30s Casbin
propagation racing the old async wait and reload; production policy unchanged.
**Files:** e2e/wait-for-operation{,.test}.mjs, access-requests.mjs, CI.
**Interfaces:** waitForOperation(readIdentity, operation, {timeoutMs=45000,
pollMs=100}={}) returns true ONLY after awaited200/operations contains operation.
readIdentity(signal) is asynchronous; bounded deadline aborts pending I/O. Missing
operation in a valid200 is the sole polling case. Non200/malformed/read failures
reject immediately without response-body disclosure. No background polling after
success/failure; original rejection preserved.
- [ ] Extract old single-read predicate behavior, observe real-HTTP false!==true RED.
- [ ] Native deterministic fixture: creator rule absent; pinned async wait resolves
  false; actual first /me200[] THEN real SQLite INSERT. Old single-read readiness
  must fail its true assertion (not random timeouts). Record exact native RED.
- [ ] Implement deadline/abort/awaited polling; real-HTTP tests readiness, stop,
  timeout, pending I/O,202/401/429/503 and malformed data RED→GREEN.
- [ ] Remove temporary upstream characterization, retain actual delayed rule and
  all12 strict reload assertions plus complete request/approval workflow.
- [ ] Full locked Web/Go/native gates; one independent review and exact merge/main.
Self-review: behavior only in test utilities; no production APIs/UI/auth changes,
failure predicate preserved; valid200 absent operation is synchronization not retry.
