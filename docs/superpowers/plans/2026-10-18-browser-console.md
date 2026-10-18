# Browser console: specification, plan and inline ledger

Baseline: Core main0ccfa8ff8522fb95d93b7898f8b0a1ed9dfcc65a (PR14), Agent main95a5f7f62dceb1c369d08b8e23ffda2f7b978ce4 (PR16). Both main CIs pass; no open PR at acquisition. Isolated linked worktree: `codex/browser-console`.

## Specification

Deliver a first operator console, not the completed product. Exchange an independent management access key for SCS2.9.0 server-side sessions. Never persist raw keys in browser storage or session data. Store only ID and credential fingerprint server-side; resolve current trusted roles and credential on every request. Removal/rotation revokes next request; ID reuse cannot revive sessions. Memory store: restart invalidates sessions. Absolute8h/idle30m expiry; renew on login, destroy on logout. Existing API accepts bearer only; console API accepts session only and rejects Authorization headers.

Browser settings explicitly opt in and require database authority and exact public origin. HTTPS except loopback HTTP development. Validate Host and unsafe Origin, ignoring forwarded identities/TLS headers. Native Echo CSRF protects login/logout/writes (modern Sec-Fetch-Site or legacy cookie/header); token comes from bootstrap response. Cookies: HttpOnly/SameSiteStrict, Secure for HTTPS. Login also rate limited. No CORS. Explicit built assets directory, confined filesystem reads and no API SPA fallback.

Reuse actual publication/audit business operations. Writes require strong ETag and reason. Save is not Agent application. Preserve drafts on errors, refresh and412; never silently retry a conflict. Viewer read-only UI and backend. Bounded audit cursor. Explicit loading/failure/empty/stale and last successful read; no sample records or invented applied metrics.

Visual direction: light operational workbench, slate canvas #f5f7fa, white #ffffff, ink #18283b, blue action #145da0, amber #8a4900, error #a52222. Offline system sans, monospace only for editable configuration. Left navigation/main editor, stacked mobile. Revision/desired-only strip is the characteristic element. Existing shadcn primitives stay business-free; i18next bilingual. UX searches matched operational freshness, controlled forms and accessible errors; unsuitable marketing/glass recommendations rejected.

Not delivered: password/OIDC identity, resource delegation, structured CRUD/approval, L7/ingress, Agent freshness UI or final product acceptance.

## Context map and tasks

1. `internal/management{,access}`: RED current-role/rotation/ID-reuse tests; implement ID+digest resolver; GREEN full Go. Produces fresh Principal or unauthorized/unavailable.
2. `internal/controller`, `runtimeconfig`, `controllerapp`: RED cookie/CSRF/origin/config tests; implement lifecycle-owned sessions, separate browser group, reused handlers and confined assets; GREEN Go/race/vet/build/static/OpenAPI.
3. `apps/web/src`, `vite.config.ts`, browser tests/CI: actual Chromium test RED before replacing scaffold; implement real session/publication/audit with preserved drafts/conflicts, loading/error/empty/stale; GREEN npm ci/typecheck/lint/build and Chromium→Go→SQLite, desktop/mobile screenshots.
4. Bilingual guide/checkpoint; one independent whole-branch review; fix blockers RED→GREEN. Compare trees, exact-head required CI, expected-head merge and main raw evidence. Keep remaining L7 task active.

Review focus: reused credential ID, rotation/stale roles, fixation/login CSRF, Host rebinding/forwarding, concurrent session requests, traversal/symlinks, authenticated cache, async draft loss, logout/revocation and no false applied/browser-mock acceptance.

## Ledger

Pre-flight: task1 resolver feeds task2 authentication; task2 DTO/ETag feeds task3. No conflicting interfaces. Existing bearer boundary remains intact.
Ruling: cloud skill helper resources unavailable (writing-good-tests read failed); use this committed inline ledger. Cost: manual consistency checked at final review.
Baseline ordinary full Go suite passed (`/tmp/picketx-console-baseline.log`).
