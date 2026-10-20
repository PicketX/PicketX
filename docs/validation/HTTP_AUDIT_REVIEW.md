# HTTP audit review and correction

2026-10-20: one fresh independent whole-branch review, base81ec5bc through
local6b6d810 (published18a45f8, identical tree d6102576). No nested reviewer,
second review or platform approval. Verdict With fixes,0Critical/3Important/
1Minor. Root confirmed all three important effects and made ONE correction pass.

- Observed capability removal now commits immediately after epoch-checked /me,
  before dependent publication/audit reads. The actual parent load regression
  failed undefined identity, then passed despite publication503. Browser adds a
  held old HTTP response/observed removal/dependent503/unmount regression.
- Authorized selected views survive refresh. Both audit+request and audit+account
  no-publication regressions failed Requests/Accounts versus Audit, then passed.
  Initial unauthorized view still selects a valid fallback. Browser checks both
  mixed-capability navigation and refresh (explicit UI projections, not RBAC proof).
- A memoized ledger is independent of freshness ticks. Collapsed details create
  no contribution DOM; only one event opens, each Grant/deny list pages50 IDs with
  explicit positions/total/previous/next and no silent truncation. The new renderer
  was first absent/RED;200 records ×4096+4096 IDs now render zero collapsed li,
  opened lists50 each. Actual old6b6d810 JSX: a single closed4096+4096 record
  produced8192li, assertion expected0 RED; corrected200-record renderer GREEN0li.
  Browser controlled valid4096+4096 stress walks to last ID at
 375px. This display stress is not actual persisted-decision acceptance.

Local combined44Node pass/no skip, locked Web typecheck/lint/build pass. Original
native candidate39418402856 reached actual populated audit; web118612963297 failed
at controlled61s freshness assertion, NOT accepted. Clock stress now advances wall
time without firing the independent15s transport timeout and explicitly checks
read completion/no error before freshness. Final corrected native/full raw/four
actual archives/source/static/notices/screenshots/exacthead/main remain mandatory.

## Deferred minor

The keyboard test only focuses an idle button. Full Tab/Enter/Space, disclosure
interaction and focus preservation through busy/success/failure remain unverified.
Native semantic controls/focus styles remain, but this is not an accessibility
conformance claim. Do not represent that narrow assertion as complete keyboard proof.

## Rulings and costs

The reviewer explicitly considered and set aside each behavior below; root rules:

1. Unchanged backend evaluation stands: no engine changes here; existing actual
   authorization gates remain. Cost if wrong: a backend policy defect may remain.
2. Unchanged DB durability/migrations/retention stands: no persistence change;
   actual restart/restore gates retained. Cost: no new retention/capacity guarantee.
3. Forwarding/application/Agent correctness stays separately proven by native
   gates, never inferred from a row. Cost: a row alone cannot diagnose delivery.
4. Server filtering/newest/search/export stay deferred; explicit oldest-first
   manual keyset flow is usable basic scope. Cost: large histories require paging.
5. No polling/unobserved immediate revocation stands; every newly observed /me
   removal now applies before dependent reads. Cost: idle screens need a new read.
6. No BroadcastChannel/cross-window synchronization added. Cost: another window
   can retain historical data until it observes session/permission change.
7. Unsafe numeric IDs fail visibly without cursor advance. Cost: exceptionally
   large histories need future API string-ID migration, not rounding or skipping.
8. Over-display-limit records fail visibly, not skipped. Cost: paging can stop at
   such a record until a future operator API/export path exists.
9. Bounded parseable timestamps stand, not strict calendar/RFC3339 validation.
   Cost: parseable noncanonical timestamps might normalize in local rendering.
10. Arbitrarily hostile pre-JSON response bytes remain the shared transport's
    threat boundary, not changed here. Accepted-data DOM is now independently
    bounded. Cost: no new streaming-response byte cap against hostile servers.
11. Existing publication pagination/unrelated editor/request/account lifecycle
    stays unchanged; actual new HTTP integration defects were fixed. Cost:
    unrelated pre-existing lifecycle defects are not excluded by this review.
12. CI/archive/source/screenshots/expectedhead/main remain executor gates, not
    reviewer-established facts. Cost: delivery remains pending until they pass.
13. Controlled wall-clock suspension replaces advancing all browser timers during
    a held request; keep the actual15s timeout and require completed real response.
    Cost: this simulates clock suspension, not a real61s network-wait acceptance.

Final exact receipts and explicit lease release live in PR44 metadata; source
pending text does not supersede later immutable evidence.
