# HTTP decision audit console

Intent: let an authorized operator trace actual HTTP authorization decisions in
the existing console, without confusing a decision with forwarding or execution.
Fresh entry Core81ec5bc/tree1b04ceb9/CI39405913826 and Agent665c9a4/tree5907b051/
CI39412700638 green, no open PR; Agent29 explicit11:18:35Z release verified.
Use the existing clean isolated worktree and new exact-main branch. User autonomy
waives routine design/plan/execution/push/verified-merge pauses, not evidence gates.

## Chosen design

Add an HTTP decisions subview beside existing Publication history under Audit.
Prefer a focused feature component plus small pure keyset parser/window helper.
Mixing rows into publication history obscures different evidence domains; a new
top-level dashboard and backend query/filter expansion are unnecessary.
No Go/API/DB/config/Agent/dependency changes. Existing audit:read and SCS apply.

Read the real /console/api/v1/http-decisions?after=N&limit=50 with expected200,
no cache. Initial N=0 (oldest first). Explicit Read next decisions advances the
exclusive cursor; Start from oldest resets only on success. Empty tail does not
erase earlier rows. Keep at most200 last-loaded events, clearly label a bounded
loaded window rather than total/all history. No fabricated counts or sample data.
No automatic feed polling or hidden-page reads. Navigation/unmount/identity-key
change aborts pending reads; no old response may update a new mount/session.

Display event ID/time, allow/deny text, reason, route/caller, IPv4/IPv6 source,
revision, contributing Grant/deny IDs. Omit nonpersisted next_change/cache_ttl as
historical lifetime evidence. Do not request/store/render URI/query/headers/body,
credentials or claim the recorded source identifies a person. Decisions precede
forwarding, are not proof of delivery, and are not current permission state.

Strictly validate successful JSON: plain page/event objects, safe positive IDs and
revisions, ascending IDs above requested cursor, next_after exactly finalID (or
requested cursor for empty), max50 events, bounded strings/ID arrays, real boolean
and parseable timestamp. Invalid/202/failed reads retain prior rows and cursor,
with fixed error and explicitly outdated status.401 clears data and invokes parent
sign-out;403 clears data/cursor and offers retry with a permission message.
Successful reads expose last-success time; >=60s (including request latency and
wall/monotonic suspension) marks history as potentially behind newer decisions.
Loading, initial empty, empty-tail, read-failure, permission-loss and age states
are distinct. Buttons serialize reads, cancel on cleanup, and preserve focus.
Mount only for current audit:read; include current identity/operations in its key.
Audit navigation itself must not incorrectly require publication:read.

## Visual brief and skill decisions

Keep repository-owned light operations-console identity and existing shadcn Button,
React/TypeScript/Vite/Tailwind/i18next/npm. Palette: background#f5f7fa, ink#18283b,
primary#145da0, muted#526174, line#c3cedb, warning#8a4900; existing semantic error/
success tokens, no new font/download/theme. System sans for text, monospace only
long IDs. Left-aligned decision ledger, clear outcome words and wrapping definition
lists, not equal decorative cards or charts. 375/768/1440 plus keyboard/reduced
motion; no horizontal overflow. Two clearly selected native buttons switch audit
subviews; one primary read action and secondary restart.44px controls retained.
Self-critique: data-backed traceability is the visual emphasis, not generic KPI
tiles. ui-ux-pro-max design-system search twice returned marketing hero patterns,
so no verified product-wide match is applied/persisted; use existing tokens plus
verified React cleanup/error-handling guidance and web accessibility defaults.

## Context Map (self-reviewed before implementation)

| File | Responsibility/change |
| --- | --- |
| apps/web/src/features/http-audit/model.ts (new) | Strict page parser and bounded200 keyset window, pure functions |
| apps/web/e2e/http-audit-model.test.mjs (new) | Real parser/window tests, malformed/cursor/unsafe/private-extra failures |
| apps/web/src/features/http-audit/HTTPAudit.tsx (new) | Actual API read lifecycle, states and bounded decision ledger |
| apps/web/src/App.tsx | Audit subview/mount and correct independent audit capability gate |
| apps/web/src/locales/en.json, zh-CN.json, src/index.css | Paired English/Chinese copy, scoped responsive layout |
| apps/web/e2e/http-audit.mjs (new), e2e/console.mjs | Actual empty and real Nginx/API decision records, pagination/failures/401 screenshots |
| .github/workflows/ci.yml | Run model tests as part of existing Web gate; retain every prior gate |
| docs/HTTP_DECISION_AUDIT*.md, AUTONOMOUS_DELIVERY*.md | User flow/boundaries/lease/exact receipts/next steps |

Dependencies: existing request/APIError, shadcnButton, Nodes observationAge helper,
HTTPAuditEvent/Response backend contract, current authenticated parent identity.
Reference: Nodes cleanup/freshness, existing publication history and real e2e
ingress methods. No public API, migration, configuration or topology change.
Tests must fail before implementation. Browser successes use actual Controller/
SQLite/Nginx, controlled failures explicitly labeled, never native success mocks.
Local absent Nginx/Chromium/Unix EPERM are not acceptance; Linux CI is mandatory.

Self-review: one integrated UI feature, explicit interface/state/privacy/cursor
bounds, no placeholders; user flow matches the requested usable audit page.
