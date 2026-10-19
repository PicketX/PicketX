# Access requests and independent approval (alpha)

> English | [简体中文](ACCESS_REQUESTS.zh-CN.md)

Use database authority, the existing [human login](HUMAN_LOGIN.md) and browser
configuration. Provision separate stable `principal:` IDs: `role:requester` for a
requester and `role:admin` for an approver. Roles are trusted credential settings,
not form fields. A shared account cannot supply independent approval. Existing key
and human authentication work with the same backend checks.

1. Sign in as requester. Access requests shows a limited Resource/Endpoint catalog,
   not policy/facts, Agent credentials, other requests or administrative audit.
2. Enter the explicit source IP/CIDR, choose one pinned Resource and named entries,
   duration60..86400seconds and reason. The server normalizes IPv4/IPv6 and fixes one
   absolute requested deadline. Submission creates no permission.
3. A different authorized admin opens Access requests, refreshes the inbox and
   reviews the saved source, pinned target/UID, selected entries and deadline.
   Approval can only choose a subset and an earlier/equal absolute deadline. Its
   confirmation states shared reachability and that publication is not application.
   Self-approval is forbidden, including for admins. Rejection requires a reason
   and remains possible after target removal or expiry; self-rejection is allowed.
4. Approval atomically commits a bounded independent Grant, approval record,
   append-only workflow event and publication audit/revision. The normal complete
   candidate compiler and Agent preflight run before commit. Stream delivery follows
   commit; delivery/persistence failures retain existing fail-stop/reopen semantics.
5. Requesters see original and approved scopes/deadlines, actors, decision reason,
   Grant UID/revision and current desired Grant state (valid-desired, revoked,
   expired, withdrawn, modified, scheduled, unknown). Same-UID source/scope/deadline
   or target-network changes do not validate the historical approval; future start
   is scheduled, not active. Replacement scope is not disclosed to requesters.
   Approval history does not become 'revoked' when the
   independent Grant is revoked. No displayed state is independent execution proof.
   Administrators use Grants to revoke that contribution and Nodes to inspect actual
   reported target/stream/freshness. Applications retain their own authentication.

The portal source is not automatically the service's enforcement source. Proxy,
VPN, SNAT, routing and IPv4/IPv6 may differ. Browser forwarding headers are not
trusted mapping evidence. Explicit CIDRs can cover other people/devices; selecting
them does not prove ownership, create a tunnel, or grant application credentials.
This first workflow permits requests for catalogued Resources; resource-specific
eligibility policies and automatic self-service approval are not implemented.

## Consistency and safety

New APIs exist under both `/api/v1` (independent Bearer) and `/console/api/v1`
(SCS cookie, exact Origin/Host and Echo CSRF):

| Operation | Contract |
|---|---|
| GET `/access-request-catalog` | Limited current catalog plus strong ETag |
| POST `/access-requests` | Strict16KiB JSON, pinned names/UID, explicit names, source, duration, reason; strong catalog If-Match required |
| GET `/access-requests?after=0&limit=50` | Own-only unless request:review; exclusive record ID, max100; mutable point-in-time pages, refresh to see changed earlier records |
| POST `/access-requests/{uid}/decision` | Pending version CAS; action/reason; approve also needs current publication If-Match, subset and RFC3339 expires_at |

Requested network shape (including addresses) is frozen. Retarget/removal/recreation
causes409 instead of approving a silently changed target. Expired/extended/broader
approval is refused; even a deadline crossed during preflight cannot commit.
Repeated/concurrent decisions yield one contribution, not duplicate Grants. No
automatic retry or renewal. An unknown response may have committed: refresh history
and inspect actors/revision before explicitly discarding/rebasing drafts. Plain
refresh never silently advances a retained request/review CAS base.

SQLite publication schema3→4 is a versioned transaction adding request/event
tables and owner index; no policy migration or file-authority takeover. Casbin
business schema1→2 adds only new defaults once: admin create/read-own/review,
requester create/read-own, blocked denies. Existing/custom rules are retained;
an existing rule for a built-in role/new operation is not replaced. Approval also
requires publication:write. Catalog access requires request:create OR both
request:review and publication:write. Creation does not require history-read;
review-only roles can load the inbox and reject without catalog/creation access.
Custom roles/policies must explicitly receive the corresponding operations.
Policy synchronization remains periodic and credentials
are reread per request; audit IDs cannot establish packet/user attribution.

## Verification and remaining scope

Real HTTP/Casbin/SQLite tests cover strict inputs, own-only visibility, deny
precedence, self-approval, narrowing/absolute deadlines, retargeting, migrations,
transaction rollback, concurrent decisions and reopen. A runtime test verifies the
exact approved IPv6 contribution in real gRPC and unchanged recovery after restart.
Chromium exercises actual separate human sessions, live API/catalog/form, approval,
rejection, Grant/audit readback and revocation. Final exact-source CI/artifact proof
and independent review are recorded in PR32 and AUTONOMOUS_DELIVERY, not inferred
from this description. Existing Agent kernel and old browser/Nginx gates remain.

No OIDC, account CRUD, cancellation, notifications, multi-step approval, policy
eligibility, PostgreSQL/HA or new privileged approval→Agent acceptance is claimed.
One Controller/SQLite only; requests/events currently have no retention/size budget.
Keep database backups private because source/scope/reasons/identity are sensitive.
The current browser has manual refresh and explicit age/failure states; it is not
a real-time workflow notification system. Continue the roadmap, not a full release.
