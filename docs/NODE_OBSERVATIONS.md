# Node observations

The Nodes console and GET `/api/v1/agent-observations` (Bearer) or
`/console/api/v1/agent-observations` (SCS cookie) require current
`publication:read` permission. Responses are no-store; query parameters are rejected.

Only currently credentialed Agent scopes appear, deduplicated and sorted. This is
not enrollment inventory or a count of all declared Agents. An empty list means
no credentialed scopes, not successful deployment. No credentials are returned.

Each row separates distributed desired revision/hash, current stream presence,
and the latest Agent assertion (RECEIVED/APPLIED/FAILED). Revisions are decimal
strings, including values above JavaScript's safe integer range. The report's
observation time is Agent-supplied; receive time is measured by the Controller.
Report freshness lasts strictly less than 60 seconds from local receipt and only
while that same reporting session remains active. `matches_desired` separately
compares revision and hash. New publication or reconnection cannot freshen history.
APPLIED means **Agent reported applied**, never an independent kernel observation.
Even a fresh matching report is a point-in-time assertion, not guaranteed current
reachability, verified revocation or continuous health. There is no heartbeat yet.

The UI requests on entry and every 15 seconds without overlap, pauses in hidden
tabs, refreshes on visibility resume and cancels on exit. Manual refresh is
available. Read failures retain clearly historical data; browser/report age does
not reset merely by refreshing the page. Hashes are expandable; mobile cards wrap.

Observation state is process-local and disappears on Controller restart; removed
credential scopes disappear. Readiness/delivery failure returns503 instead of an
apparently current old delivery view. There is no historical status database,
cross-Controller aggregation, automatic Agent enrollment or kernel change here.
Source policies, absolute expiry, LKG and existing Agent enforcement are unchanged.
Actual current packet/kernel behavior still requires privileged Linux acceptance.

Validation and exact PR/main evidence: [delivery checkpoint](AUTONOMOUS_DELIVERY.md).
