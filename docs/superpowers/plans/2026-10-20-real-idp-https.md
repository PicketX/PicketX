# Real Keycloak HTTPS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline. Steps use checkbox syntax.

**Goal:** Verify actual installed Controller OIDC against pinned real Keycloak over normal HTTPS.
**Architecture:** New independent native test using existing bundle, private init,
Nginx and exact existing account/session APIs; retain all original gates.
**Tech Stack:** Existing Node/Playwright/static Go; CI-only Keycloak26.8.1/JDK21.
**Spec:** docs/superpowers/specs/2026-10-20-real-idp-https.md

## Global Constraints
- Archive SHA2564e2740b181faa2ea623a1c2ecf00855935311e546046e2f857615a47ed24cbe8.
- Separate127.0.0.2 provider/127.0.0.1 Controller sites; no TLS verification bypass.
- Empty PATH installed binaries; explicit existing human/subject binding/local Casbin.
- Phase-only failure output; private state/logs never in public evidence.
- Real disposable provider does not prove production IdP/MFA/HA or Agent conjunction.
- Full old Go normal/race/vet/build, locked Web/native gates and4artifacts retained.

## Review Focus
- Unknown CA/wrong SAN must fail, not silently become trusted.
- Valid IdP identity without explicit binding must not gain a local session.
- HTTPS correlation and SCS cookies preserve Secure/HttpOnly/Lax-vs-Strict boundaries.
- Local role change/revoke must apply despite a still-live IdP SSO session.
- Failure/cleanup must neither expose private state nor leave trust/process changes.

### Task 1: Real-provider installed HTTPS acceptance
**Files:** Spec context map; one integrated deliverable, no shared task interfaces.
**Interfaces:** createTestCertificate(directory) -> {ca,cert,key};
requestHTTPS(url,{ca,method,headers,body}) -> {status,headers,body,json()};
real-idp.mjs BUNDLE KEYCLOAK_DIRECTORY -> real-idp-report.json +success PNG.

- [x] Add actual TLS server tests: explicit CA success, no CA failure, wrong SAN failure,
  HTTP rejection, bounded request timeout and fixed diagnostic without payload.
- [x] Observe missing helper RED, implement minimal strict TLS helper and rerun GREEN.
- [x] Implement real Keycloak realm/import/TLS lifecycle and Nginx/installed Controller;
  native browser asserts wrong password/unbound identity, exact bound principal,
  S256/nonce/cookies, publication/audit, role generation401/newviewer403/revoke/logout.
- [x] Add pinned SHA download/JDK21/certutil and mandatory native script step, preserve
  every original gate. Public allowlisted receipt/success-only screenshot/phase errors.
- [ ] Run complete Node, actual HTTP regressions, locked Web, full Go normal/race/vet/build;
  native exact-head full3jobs/no skips +real provider gate required. Synchronize docs.
- [x] ONE independent whole-branch review; ONE root RED→GREEN blocker pass, no new minors.
  Private provider staging +success/failure lifecycle regressions; Node32 pass.
- [ ] Final exact source/raw/four actual archives/static/notices/images, expected-head merge,
  fresh main full proof and immediate lease release in PR; not overall goal completion.

Self-review: spec coverage and5review-focus conditions map to actual tests/native
assertions; single test deployment task worth one final review, not split scaffolds.
Final integration is a separate gate, not a cyclic precondition to its own review.
