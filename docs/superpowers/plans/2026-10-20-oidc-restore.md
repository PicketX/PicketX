# Installed OIDC recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline. Steps use checkbox syntax.

**Goal:** Prove complete stopped OIDC state restores to a new installed service path.
**Architecture:** Existing native systemd flow +signed fixture +two offline path rewrites.
**Tech Stack:** Existing Node/Playwright/systemd/static Go, no new dependencies.
**Spec:** docs/superpowers/specs/2026-10-20-oidc-restore.md

## Global Constraints
- Exact identity/provider.json and identity/bindings.json;0700/0600 private modes.
- Retain issuer/client ID/secret/subject/account/generation and authority unchanged.
- Original paths unavailable; old cookie401 and actual fresh login required.
- Fixture NOT production IdP; no Agent/kernel/UI/runtime behavior changes.
- All prior Go normal/race/vet/build, locked Web and native gates remain required.

## Review Focus
- External/unexpected OIDC paths must fail before rewriting a document.
- Restored service cannot read original or backup paths to make login succeed.
- Old cookie/withdrawn binding must not authorize after restoration.
- Public reports/logs/images contain no secrets/private state.
- Fixture extraction preserves one-use PKCE/client/redirect checks and old cases.

### Task 1: Ship installed OIDC stopped restore acceptance
**Files:** Spec context map; existing CI runs the affected Node/native flows.
**Interfaces:**
- relocateInstanceConfig(config,old,target) -> clone including optional exact provider path.
- relocateOIDCProvider(provider,old,target) -> clone changing exact bindings path only.
- createOIDCProviderFixture({origin,clientID,secret,subject}) -> {issuer,exchanges,lastCallback,setMode(mode),close()}.
- service.restore.oidc -> boolean receipt with providerKind:'signed-protocol-fixture'.

- [ ] Write literal optional/nested path preservation and external path rejection tests. Run node --test scripts/controller-state-copy.test.mjs; expected RED unchanged path/missing helper.
- [ ] Implement minimal pure rewrites; run all bundle/state/service Node tests; expected GREEN.
- [ ] Extract existing issuer; retain all old OIDC assertions. Extend native service-only config/browser flow, exact changed-file checks and receipt.
- [ ] Run Node suite; npm ci/typecheck/lint/build; full Go normal/race/vet/build. Local inherited Unix EPERM is not green; full native3-job success/no skips required, including actual OIDC restored service.
- [ ] Synchronize EN/ZH docs, logical commits; ONE independent whole-branch review and ONE RED→GREEN blocker pass, defer minors explicitly.
- [ ] Check exact trees/raw CI/four actual archives/ELF/notices/images; expected-head merge and full main/source/archive recheck; final immutable receipt/release in PR. Deployed real IdP/HTTPS/Agent remains next.

Self-review: one integrated task, no shared task interfaces. All five review-focus
conditions assigned to local path tests or native flow. Inline execution selected;
routine plan/design/merge confirmations explicitly waived, not safety gates.
