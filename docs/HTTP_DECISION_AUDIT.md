# HTTP decision audit console

Sign in with current `audit:read` permission, open **Audit → HTTP decisions**.
This view reads the existing authenticated `/console/api/v1/http-decisions` API.
It is read-only and independent of permission to edit/publish configuration.
Publication history remains a separate Audit subview.

Records are oldest-first in exclusive event-ID order,50 per API page. **Read next
decisions** continues from the last successful cursor; **Start from oldest** resets
only after a successful read. An empty tail preserves earlier records. The screen
retains at most the200 last-loaded records and labels both this window and cursor;
it is never a total count, all history or an automatic live tail. Use next again
to check for newer records. No hidden-page reads or automatic feed polling occur.

Each record shows its ID/time, allow/deny, decision reason, route/caller, source IP,
revision and contributing Grant/deny IDs. IPv4/IPv6 and long IDs wrap. A decision
is not proof of forwarding, application delivery or current authorization; a source
address does not identify a person. Nonpersisted lifetime/cache fields are not
presented as historical facts. Invalid/unauthenticated callers may fail before
decision recording. URI/query/headers/body and credentials are not displayed.

Loading, initial empty, empty tail, unavailable/invalid read, permission loss and
outdated history are separate. Failed/schema-invalid/unexpected202 reads retain
previous history/cursor, never substitute samples or advance silently.401 clears
history and signs out;403 clears it and permits retry. Last-read status becomes
outdated after60s, including request latency and browser/device suspension; this
is collection freshness, not a lease or current permission assertion. Requests are
serialized and aborted on subview/session/identity cleanup.

English/Chinese, existing light console tokens/shadcn controls, keyboard focus,
reduced motion and375/768/1440 layouts are retained. JSON validation rejects unsafe
IDs/revisions, backwards/duplicate/inconsistent pages, unknown fields, malformed
booleans/timestamps, strings over256characters or contributing arrays over4096/
duplicate IDs. These are display safety limits, not authorization policy limits.
Oversize records fail visibly without modifying permissions; advanced server-side
filtering/latest-page search/export and retention/capacity guarantees are deferred.

## Verification boundary

Test-first PR44 initially fails in actual Linux Chromium at the missing HTTP
decisions button (CI39416795817/web118607673736 raw log), not a local mock.
Parser/window tests first failed missing implementation, then duplicate-ID
regression failed `Missing expected exception` and was fixed. Local Go normal/
race baseline Unix authorization socket EPERM is not native success. Nginx/
Chromium absence is not a skip accepted as proof; full Linux CI must execute.

Actual browser success uses real Controller/SQLite/Nginx allow/deny/IPv6/Grant/
deny-ID records;205 additional real authenticated decision calls test pagination/
200-window (not205 forwarded requests). Controlled503/202/schema/403 and capability
projection cases test UI failure boundaries only. Actual separate credential
removal tests401 clearing. All old gates remain. Exact final head/CI/archives/main
and review/rulings are in the delivery checkpoint/PR, not inferred here.
