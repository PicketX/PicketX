# Explicit HTTP path-segment prefixes

## Purpose and authority

The exact-only HTTP alpha cannot express a repository subtree or static asset
directory for the planned real Gitea acceptance. Add an explicit, bounded selector;
do not bypass ingress authorization or infer application identity. Autonomous
implementation, push and verified exact-head merge are already authorized.

## Contract

- HTTPRoute `spec.pathPrefixes` and management `path_prefixes` are optional lists.
  `paths` remains an exact-match list. At least one selector across the two lists
  is required. Omitted lists mean no contribution; explicit empty/null lists are
  invalid. Existing exact-only documents and requests retain their behavior.
- Each list is duplicate-free; the combined count is at most128. The same string
  in different lists is allowed: exact and prefix have different meanings.
- A prefix `/repo` matches `/repo`, `/repo/`, and `/repo/child`, not `/repository`.
  Explicit `/` matches every otherwise-valid canonical path. Other trailing-slash
  prefixes are rejected, not silently normalized. No wildcard/glob/regex syntax.
- Existing strict ASCII/canonical path validation applies before matching; percent
  escapes, backslashes, duplicate slashes and dot segments fail closed. Queries
  do not participate in matching and are not persisted in decision audit.
- Exact/prefix selectors form a union within one route. Caller route scope, exact
  Host, methods, pinned TCP Resource/Endpoint and current source Grant/deny/time
  requirements remain conjunctive. Request-time evaluation remains uncached.
- Decode/re-encode, immutable compiled state, inventory and SQLite publication
  restart preserve the new selectors. No SQL migration, Agent wire or GPL changes.

## Console

Keep the established white/navy/blue console, system sans, left-aligned forms,
palette #f5f7fa/#18283b/#145da0/#526174/#ffffff. Separate exact and prefix textareas
and detail sections; clearly explain segment boundary and broad `/` selection.
At least one field is needed, not both. Existing loading/empty/failure/stale/CAS,
unknown-write and explicit rebase behavior remains intact. EN/ZH,375/1440 layouts
must show the real API-backed selector and inline errors without overflow.

## Acceptance

Tests-first Go coverage must catch a raw string-prefix widening, missing canonical
validation, lost cloned prefixes, empty/null broadening and lost durable restart.
Real Chromium creates a prefix-only route through the console. Real Nginx→local
AuthZ→HTTP application proves nested allow, boundary/method/Host/source denies,
malformed-path rejection and Grant revocation on a reused connection. Existing
exact-route suites remain mandatory. Full Go normal/race/vet/build and locked Web
typecheck/lint/build plus native CI, one independent whole-branch review, exact
source-tree/raw-log/archive evidence and postmerge main CI gate delivery.

Actual Gitea HTTPS/SSH, deployment/release acceptance and whole-goal completion
remain pending. This increment is not proof of those scenarios.
