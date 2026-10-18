# Structured Grant management

## Focused design and context map

Accepted baseline: Core main113461ba99bf070139ba7b8858d7eba996c5ce7f,
tree0a864577ad7819c00125a75ca021c4eeabd8699b, main CI39131202785 green.
Agent main95a5f7f/CI39118687475 remains unchanged. Both repositories had no
open PR at acquisition. The PR16 merged+green lease has released.

User-approved autonomous execution: proceed inline without routine approval
waits. This increment makes one real admin workflow usable: select an existing
Resource and explicit named Endpoints, issue an independent time-limited source
Grant, observe its desired validity and revoke it with a reason. Approval,
resource/binding CRUD, verified human identity and applied-node status are not
supplied by this increment. Active desired permission is never applied evidence.

Choose server-owned structured transformations through the existing complete
publication CAS/audit transaction. A second Grant database/publisher duplicates
authority; client YAML mutation puts trusted domain logic in the browser. Reject
both alternatives. The database remains the sole managed authority; preserve
facts and every unrelated Grant/NetworkDeny/HTTPRoute/Binding and immutable UID.
Canonical JSON documents separated by YAML document markers retain JSON field
names and validated omission semantics. Full shared decode/compiler validation
still gates the complete candidate.

New Grants pin a Resource name+UID and nonempty explicit Endpoint names, get a
cryptographically independent server UID and server-calculated absolute expiry.
Offer durations60..86400 seconds (a narrow admin API bound, not a policy-wide
lifetime change). No renew, edit, unrevoke or automatic CAS retry. Imported
permanent/future/multi-scope/dynamic Grants remain readable and revocable;
display dynamic scope as such, not as a fixed current selection. Revoke only
Grant, never deny; preserve other independent contributions. Check expected
revision before idempotency. Keep expiry unchanged on retry/read/restart.

GET grants uses bounded UID keyset pages (default50,max100); page continuation
requires the first page's strong revision. POST issue/revoke require strong
If-Match, trusted current write RBAC, strict bounded JSON and nonblank reason.
Reuse both independent bearer and SCS/CSRF adapters, transactional audit and
post-commit distribution. Read-only catalog includes existing Resources and
Endpoints; catalog is bounded by existing4MiB/4096-object publication limits.

Context map: pkg/configuration/{types,decode,compile,candidate}.go owns policy
semantics; internal/management owns trusted use cases; internal/controller
{management,browser}.go owns strict DTO/ETag/session adapters; publication.Store
and controllerapp publisher own atomic authority/distribution; apps/web/src
features/grants owns UI, lib/console.ts transport, App.tsx navigation, locales
and index.css shared presentation; apps/web/e2e/{console,ingress}.mjs owns real
Chromium/API/Nginx/application acceptance. No Agent/kernel changes.

Visual/component plan (frontend-design + ui-ux-pro-max): retain existing light
operations workbench, system fonts/offline assets, slate surfaces, blue primary,
existing Button primitive and 44px controls. Left-aligned issue form followed by
bounded selectable/copyable Grant rows, explicit absolute dates, wrapping IPv6
and UID. No hero, ornamental statistics or sample data. Existing sidebar gets
one Grants destination. Native labeled selects/checks and a focused error summary
with inline linked errors; scoped revoke confirmation with cancel/reason.
Loading/error/empty/stale/read-only states are explicit. Refresh must not silently
advance a preserved draft's CAS base;412 retains input and requires explicit
reload. No background write, no credentials in storage. Pagination replaces rows.
Keep existing YAML editor CAS base unchanged by structured writes. Dirty form
navigation must warn. Server observation time/validity is not live enforcement.

UX search retry did not yield a verified operations page pattern (both generic
landing layouts); retain repository layout, not the off-topic recommendation.
Verified error-summary/aria feedback search and React form guidance apply.
Self-critique: this is a narrow workflow, not a complete resource manager; long
scope strings, stale drafts and accidental revoke are the main interaction risks.
Bounded pages, explicit revision/reload and two-step confirmation address them.

## Execution plan

### Task 1: configuration inventory/add/revoke and management/HTTP adapters

  RED missing API/domain tests first; validate preservation, absolute deadlines,
  dynamic scopes, duplicate name, stale/no-op, RBAC/strict DTO/audit and pagination.

### Task 2: real structured Grant page and browser→API→authorization→Nginx
  application acceptance. Missing-page RED in real CI; loading/error/empty/stale,
  conflict draft/reason, read-only keyboard/mobile and independent revoke covered.

### Task 3: bilingual docs/checkpoint, full verification, one fresh independent
  whole-branch review, fix Important/Critical in one RED→GREEN/full-check pass,
  exact remote/source/tree/required raw CI, expected-head merge and main CI.

## Validation and ledger

Local baseline complete ordinary suite executed: one existing Unix-socket runtime
case fails EPERM in this host; all other packages pass. Keep it mandatory; actual
Linux CI must run it without skip. Run complete ordinary/race/vet/build and locked
Web npm ci/typecheck/lint/build. Native CI must also run pinned generation/local
OpenAPI, actual static SQLite Controller/CLI and actual Chromium/Nginx flow.
Review and CI evidence will be appended before integration; no completion claim
from mocked handlers or simulated execution. Final combined Agent/browser and
human/approval workflows remain required before pausing autonomous delivery.

### Increment ledger (2026-10-18T14:06Z)

Pre-flight: Task1 catalog/issue/revoke produces the JSON field/ETag contract used
by Task2; names and single database authority match. Task1 retains existing
repository.Replace CAS/audit/distribution, so no second persistence owner is
introduced. Domain/API missing-symbol/404 regressions were observed RED before
implementation. Task1 commit68df1ed passed configuration/controller tests; its
runtime/rollback/OpenAPI supplements are in6b50aec and selector/HTTP preservation
supplement in a0d87d2. New actual TCP→gRPC/restart test first exposed its own
duplicate-stream replacement assumption, corrected to observe the same stream;
the production single-active-stream contract was retained.

Task2 implementation6b50aec, summary markup fix247dc56. Real Web CI39133088621
job117692062184 was intentional missing-Grants-button RED. CI39133841930
job117694246371 reached the actual application but its exact error-summary text
assertion could not match the summary's combined field-list text. Separate summary
paragraph and native focus observation fixed it (not a removed assertion).
CI39134161791 Web117695168450 GREEN: actual structured forms→SCS/CAS/SQLite→
local decision→Nginx1.24→real HTTP application, independent union/revoke on the
same ingress connection, immutable expiry, concurrent-writer412/input/reason,
read failure, viewer backend403/mobile and existing browser/HTTP cases all pass.
Local247dc562758aa09bf0566fdef88b1d1cd2e39ed7 and fetched actual CI checkout
460684aa290d31a13d7c6788239081ae079af715 share treeda7926c837beb7aa8dc24c9b978fed5b75126b65.
Remote headdde54dbcd2a2cff5ae506f9265873827023ad274 has that exact tree.
Screenshot artifact12139072700 independently downloaded; SHA256
aedb507fdab14912cdfe6fbc0632b2d024c547fbcbfddfa07e498b6e800d124f matches API.
Desktop and375px viewer screenshots visually inspected; no horizontal overflow.

Complete local ordinary/race runs retained the mandatory Unix EPERM failure.
Parallel pressure initially exhausted an existing200ms HTTP boundary fixture and
the new runtime helper's1s HTTP client. Boundary fixture now leaves2s actual
compile/fsync headroom while still asserting a pre-boundary query, real mutex/
SQLite blocking and crossed absolute expiry/deny activation. Serial complete
ordinary/race then fail only the known Unix permission case; new runtime and all
four existing boundary cases pass. No race warning. Local vet/build and locked
npm ci/typecheck/oxlint/build plus local-only OpenAPI validation pass. Native
CI39134161791 Go117695168118 race/build/generation are green; static gates still
running at this ledger entry. Exact final whole-branch CI remains mandatory.

Task2: Ruling: task-start bookkeeping ran after its combined implementation
commit; use actual68df1ed..247dc56 range and the observed native RED→GREEN CI,
not the late helper's6b50aec start — provenance follows verified git/CI — cost if
wrong is misleading task attribution, not an untested production change.

One fresh whole-branch review, findings/rulings, final source/CI acceptance and
expected-head merge/main verification remain pending. Do not infer acceptance
from this ledger's intermediate code evidence.
