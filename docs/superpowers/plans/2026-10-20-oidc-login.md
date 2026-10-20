# OIDC browser sign-in Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan inline. Steps use checkbox syntax for tracking.

**Goal:** Real opt-in OIDC sign-in for existing trusted human accounts.
**Architecture:** A focused protocol adapter verifies provider identity; explicit
subject bindings select current local human accounts. Existing SCS/Casbin own
sessions/operations and retain key/password recovery; no Agent or Grant mutation.
**Tech Stack:** Existing Go/Echo5/SCS/Casbin/Koanf, React/i18next/npm; pinned
go-oidc v3.21.0 and oauth2 v0.37.0, no hand-written JWT verifier.
**Spec:** docs/superpowers/specs/2026-10-20-oidc-login.md

## Global Constraints
- One provider, restart-only provider config; fresh private bindings and human roles.
- Pending max128, absolute5min; calls timeout5s; no redirect; max1MiB bodies.
- Existing Strict SCS cookie stays; callback correlation Lax/HttpOnly/path-scoped.
- All state/nonce/correlation/PKCE are independent256bit; single consume before I/O.
- No email/claim-role auto-link, key principal mapping or external token persistence.
- All prior CI/security/standalone/IPv4+IPv6 gates retained; no Agent changes.

## Review Focus
- Two simultaneous callbacks must not exchange the same transaction twice.
- Remove/retarget binding or disable/re-enable managed account must not revive cookie.
- Issuer discovery/network failures must disclose no secret/provider payload.
- Multi-audience/empty subject/future issued-at must not slip through library defaults.
- Cross-site callback must work without weakening ordinary login CSRF or SCS SameSite.

### Task 1: Ship the integrated OIDC browser workflow

**Files:** Context map in spec, exact new package files and adjacent tests.
**Interfaces:**
- `oidclogin.New(ctx, configPath, origin) (*Client,error)` explicit construction.
- `Client.Start() (authorizationURL,state,correlation string,error)`;
  `Client.Complete(ctx,state,correlation,code) (Identity,error)` one-use proof.
- `Client.Resolve(subject,fingerprint) (principalID,bindingFingerprint string,error)`;
  `Identity` contains Subject/PrincipalID/Fingerprint/ExpiresAt.
- `Access.AuthenticateHuman(id) (Principal,string,error)` and
  `ResolveHumanSession(id,fingerprint) (Principal,error)` never resolve key principals.
- `Browser.SetOIDC(*oidclogin.Client)` startup-only injection; live session resolution
  uses external binding then current local fingerprint, never claim roles.

- [ ] Write RED tests first: existing runtime loader rejects `browser.oidc_config`;
  human resolver cannot exist; new integration expects protected start/callback and
  signed issuer proof. Assert hostile nonce/iss/aud/signature/expiry/iat/azp/empty sub,
  missing/duplicate/cross-browser state/code, replay/concurrent callbacks, limits,
  binding removal/rotation/retarget/current account revocation, private config errors.
- [ ] Run focused tests; expected RED because config/flow/resolver are absent, not
  fixture errors. Install pinned production dependencies only after spec/plan.
- [ ] Implement minimal client/private binding adapter/human resolver/runtime assembly
  and browser flow exactly as spec. Keep new files focused, old auth paths unchanged.
- [ ] Run focused Go normal/race; expected GREEN. Capture actual RED/GREEN evidence.
- [ ] Read UI/UX and frontend-design skills, retain existing visual rules. Add real
  browser protocol acceptance before UI; expected RED missing organization button.
- [ ] Implement additive EN/ZH organization button, loading/failure/empty-config
  behavior; actual signed protocol fixture validates PKCE + real Controller identity,
  trusted viewer/admin deny, account changes, local logout/cookie replay, no app data
  substitution. Run locked Web typecheck/lint/build and actual browser gate GREEN.
- [ ] Update operator docs/Chinese counterpart, dependency notices and durable proof;
  run full Go normal/race/vet/build and all prior Web/native gates; local inherited
  Unix EPERM is explicitly not success and requires real Linux CI.
- [ ] Commit logical changes, one fresh whole-branch review, fix blockers once using
  RED→GREEN; exact-tree/native raw/four actual archives and expected-head merge,
  main CI/source/archive recheck. No merge before all gates. Record release/remaining.

Self-review: all spec requirements map to this single integrated deliverable;
Review Focus cases are owned by its protocol/browser/account tests; signatures and
limits agree. The user waived routine stage pauses; inline execution selected to
avoid competing implementers. No shared cross-task interfaces (one task).
