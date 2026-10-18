# Structured source Grant management (alpha)

English | [简体中文](GRANT_MANAGEMENT.zh-CN.md)

The [operator console](BROWSER_CONSOLE.md) now has a Grants page backed by the
same sole database authority as complete publication. It is an administrator
workflow, not self-service approval or verified human identity. Create
Resources/Endpoints/Bindings on [Resources](RESOURCE_MANAGEMENT.md); configured host
facts and trusted HTTPRoutes still use Desired configuration. Resource/Binding
editing/removal, approval and fresh applied-node status remain separate work.

1. Open Grants and explicitly **Use latest Grant revision** after configuring
   inventory. Choose a unique Grant name, actual source IP/CIDR, pinned Resource
   and one or more named Endpoints, duration and audit reason. A CIDR grants every
   matching address, not a verified person; application credentials still apply.
2. Issue Grant. The server generates an independent random UID and fixes one
   absolute UTC expiry. The form offers15m/1h/8h/24h; the API permits60..86400s.
   It does not edit, renew or unrevoke existing Grants. A duplicate name fails.
3. Inspect desired validity **at the last successful read**, absolute deadline
   and UID. An active desired Grant is not proof of effective permission: explicit
   deny wins, independent Grants combine, and this page cannot confirm execution.
4. Clear/finish an issue draft before revoking. Select Revoke Grant, inspect its
   source/identity, enter a reason and confirm. Only that Grant is revoked. Other
   Grants can still allow access; NetworkDeny and unrelated objects are unchanged.

Reads and pagination do not silently advance a preserved operation's CAS base.
On412 inputs/reason remain and writes stop until explicit rebase. Rebase preserves
inputs: review the selected Resource UID and Endpoint scope before resubmitting.
Transport/5xx write failures may have committed; inspect the latest state before
trying again. No automatic mutation retry. Switching tabs preserves drafts;
unload/sign-out warns. The YAML editor keeps its independent revision base and
requires explicit reload after structured changes. Clear inputs are not saved in
localStorage/sessionStorage. Viewer roles can read but cannot issue/revoke, and
current trusted roles/credential revocation are checked on every backend request.

## API

Bearer routes below use independent management authentication and current
`publication:read`/`publication:write` RBAC. Identical `/console/api/v1` routes use
the existing SCS session, exact Origin and native CSRF protection. They are not
exposed on the local authorization listener. [Generated contract](api/swagger.json).

| Method/path | Contract |
| --- | --- |
| GET `/api/v1/grants?limit=50&after=UID` | Desired Grants ascending by UID, existing Resource/Endpoint catalog, authority/revision, server observation time, exclusive next cursor and has_more. Default50,max100. Continuation requires strong If-Match of the first page; stale412, missing428. Refresh without cursor returns first page. |
| POST `/api/v1/grants` | Strict JSON: name, source, resource_name, resource_uid, endpoint_names, duration_seconds, reason. Strong If-Match required.201 with uid/revision/expires_at and new ETag. |
| POST `/api/v1/grants/{uid}/revoke` | Strict JSON reason; strong If-Match checked before idempotency.200 uid/revision and ETag; missing or non-Grant UID404. Repeating current-state revoke does not create a revision/audit event. |

Requests reject unknown/duplicate/null fields; mutations are limited to16KiB.
Reason is nonblank,max512 bytes; new name max128 bytes; at most256 named Endpoints.
Errors400/401/403/404/412/413/415/428/429/503 follow the documented adapter contract.
All responses are no-store. The catalog remains within existing bounded
publication input limits (4MiB/4096 objects); rendered Grant pages have at most50
rows, and page continuation cannot mix revisions. Imported permanent, future,
expired, revoked, multi-scope and label/all-Endpoint dynamic Grants remain readable
and revocable; a dynamic selection is not relabeled as a fixed current list.

## Authority and evidence boundaries

The server transforms the accepted publication, preserves facts exactly, then
passes the complete candidate through existing shared validation/immutable-ID
checks and the same expected-revision transaction/audit/distribution owner. There
is no second Grant table or background policy-file writer. Accepted UTF-8 policy
is retained: issuance appends one compact JSON document (valid multi-document
YAML); revocation uses parser coordinates to edit only the Grant boolean. It
strictly re-decodes and compares the entire result against the exact intended
semantic change. This avoids indentation/escaping expansion of imported policy,
including large annotations. The new state must still fit the unchanged4MiB and
object bounds; adding an object/flag requires room. Unrelated object semantics,
labels, selectors and independent deadlines remain.
Existing file mode does not gain management writes. No business migration or
Agent/protocol/kernel change is introduced.

Tests include real SQLite audit rollback, stale/idempotent CAS, current RBAC,
strict DTOs and revision-bound pagination; real TCP management→gRPC snapshots and
durable independent IPv6 revoke/expiry preservation; actual Chromium form→SCS/CAS
→local decision→Nginx→test HTTP application with overlapping Grant/revoke,
conflict/draft, failure, viewer/mobile and audit checks. Exact tested head, tree,
raw CI and review acceptance are recorded in [the checkpoint](AUTONOMOUS_DELIVERY.md)
and PR17. This is not new native Agent execution or full product acceptance.
