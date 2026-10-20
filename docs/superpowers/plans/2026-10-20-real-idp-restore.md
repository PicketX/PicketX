# Real IdP Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline. Steps use checkbox syntax.

**Goal:** Real Keycloak HTTPS login survives complete stopped Controller relocation
with exact durable identity/policy state and invalid old sessions.
**Architecture:** Extend the existing installed real-provider runner; a small
private-copy helper reuses existing strict state utilities and a strict public gate.
**Tech Stack:** Existing Node/Playwright/Go static bundle/Nginx/Keycloak26.8.1/JDK21.
**Spec:** docs/superpowers/specs/2026-10-20-real-idp-restore.md

## Global Constraints
- Core base64c8cb8e6c92669873031ca45cff47dbb38d0224/tree4637c029a34c71ec04b66fcc205c288f1280dfb9.
- Actual Keycloak26.8.1 SHA2564e2740b181faa2ea623a1c2ecf00855935311e546046e2f857615a47ed24cbe8.
- Keep all existing3jobs/fourartifacts/real-provider/signed-restore gates and HTTPS verification.
- No product Go/API/UI/kernel/dependency change; no IdP DB or online backup claim.
- Private0700directories/0600files, actual clean stop, no original fallback, no raw secrets/errors.

## Review Focus
- Partial restore or caller paths must not overwrite retained state or leak private diagnostics.
- Process must be stopped before copying, provider remains actually alive, original instance unavailable.
- Restarted session/local identity cannot be satisfied by old browser or recovery-key sessions.
- Durable views and absolute deadlines must not change through restore or observation normalization.
- Failure/cleanup must not publish success or leave owned provider/trust state.

### Task 1: Complete real-provider recovery acceptance
**Files:** Spec context map.
**Interfaces:** restoreRealIdpInstance(instance,backup,target,retired) ->
{instance,configuration,provider,proof}; validateRealIdpRestoreEvidence(receipt,source)
-> throws on malformed/private/missing/false evidence. Existing TLS/copy APIs unchanged.

- [ ] Write real filesystem copy/relocate/preserve/unsafe-target/error-redaction tests
  plus strict synthetic public gate mutations before helper. Run node --test
  scripts/real-idp-restore.test.mjs; Expected missing required functions RED.
- [ ] Implement minimum helper using existing state utilities; same command GREEN.
- [ ] Integrate nested identity paths and actual clean stop/relocate/start into existing
  real browser flow; preserve originals, fresh exact ID/roles and PKCE/nonce/session,
  durable views, human publication/audit and binding withdrawal; only public report.
- [ ] Add mandatory CI tests, synchronized scope docs/checkpoint. Full Node suite,
  Go normal/race/vet/build, locked Web ci/typecheck/lint/build; inherited local socket
  EPERM recorded, never called native green. Commit coherent implementation.
- [ ] ONE independent whole-branch review; root ONE blocker RED→GREEN correction,
  defer minors, record declined scopes/costs. Exact-head fullCI/raw/fouractualarchives/
  manifests/static/source/notices/images required before expected-head merge.
- [ ] Fresh merged main all3jobs/raw/fourarchives/source verified; immediate release,
  preserve outstanding CLI status/doctor goal, no overall completion claim.

Self-review: one integrated acceptance task, no inter-task interfaces; each review
focus covered by helper tests or mandatory actual runtime assertions.
