# Resource onboarding (alpha)

English | [简体中文](RESOURCE_MANAGEMENT.zh-CN.md)

The [operator console](BROWSER_CONSOLE.md) Resources page reads the real database
publication and creates Resources/Endpoints and explicit host Bindings. Creating
a Resource does **not** grant access. A Binding is desired protection placement,
not proof of Agent application. [Grant issue/revoke](GRANT_MANAGEMENT.md) remains
a separate operation; [HTTPRoutes](HTTP_AUTHORIZATION.md) have a structured console
page. This increment introduces no second authority or tables.

## Operator flow

1. Configure trusted Agent capabilities and host execution facts in Desired
   configuration. This page does not enroll Agents or infer facts from heartbeats.
2. Open Resources, explicitly use the latest resource revision, then New Resource.
   Enter a unique name, 1–64 named TCP/UDP single-port Endpoints and audit reason.
   Each port is 1–65535. Optional address restrictions support IPv4/IPv6 IP/CIDR,
   newline/comma separated, at most64 per Endpoint. Blank means omitted: use the
   Binding's destination scope, not deny-all or an invented global address scope.
3. Create Resource, then New Binding. Choose the pinned Resource UID, 1–64 explicit
   named Endpoints and an existing `(agent_name, context_id)`. Review its profile,
   class, stage, destination boundary and handling before submitting with reason.
   Only host-local/original-destination is supported by this form. The complete
   compiler still validates Agent capabilities, physical overlap and references.
4. On Grants, explicitly use the latest Grant revision. The new Resource and its
   Endpoints are selectable for an independent absolute-duration Grant. Issue and
   revoke without editing YAML; inspect audit reasons and current desired state.
5. To retire a protection placement, select **Remove Binding** on that Binding.
   Review its UID, resource/Agent selections and execution settings, enter a reason
   and acknowledge that services may become unprotected. Confirm or cancel.
   This is **not Grant revocation**: other Bindings and existing Grants remain.
   Last-Binding removal withdraws PicketX protection, not just its allow rules.
   Accepted removal is desired state, not confirmation of Agent application.

The form has one active Resource, Binding or removal draft. Switching form type/clearing
asks before discarding; switching console tabs preserves it. Refresh updates only
the observed inventory, not the draft's CAS base or selection catalog. Explicit
rebase retains inputs and refreshes choices: inspect them before retrying.412 and
ambiguous transport/5xx writes block resubmission until rebase; an error is not
proof of rollback. No automatic retry. Unload/sign-out warns about dirty drafts.
Viewer reads are supported, with backend write denial. Nothing is stored in
browser localStorage/sessionStorage. The advanced editor and Grants keep separate
revision bases; reload/rebase them explicitly after another page changes policy.
Removal rebase resets acknowledgement and displays the latest details for the same
UID; a missing target blocks confirmation. Cancellation explicitly discards that
removal draft. A successful removal is never retried after a failed follow-up read.

Inventory shows complete stored Binding selections (including imported dynamic
all/selector scopes), not a fabricated resolved list. Each page renders at most50
Resources,50 Bindings and50 configured contexts. Loading, error, empty and stale
states are explicit; timestamps describe a read, not execution freshness.

## API and invariants

Bearer `/api/v1` and session/Origin/CSRF `/console/api/v1` share current
`publication:read`/`publication:write` checks. See [OpenAPI](api/swagger.json).

| Method/path | Contract |
| --- | --- |
| GET `/api/v1/inventory` | Complete bounded Resource/Endpoint, Binding and context arrays, authority/revision/observed_at and strong ETag; query parameters rejected. |
| POST `/api/v1/resources` | name, endpoints[{name,protocol,port,addresses?}], reason;201 uid/revision and ETag. Server generates UID. |
| POST `/api/v1/bindings` | name, resource_name/resource_uid, endpoint_names, agent_name/context_id, reason;201 uid/revision and ETag. Server generates UID. |
| POST `/api/v1/bindings/{uid}/remove` | reason, acknowledge_unprotection:true;200 uid/revision and ETag. Missing/wrong-kind UID404; stale revision412 before lookup. |

All mutations require strong current If-Match (missing428, stale412), strict
JSON16KiB, no duplicate/unknown/null fields, nonblank reason<=512 bytes and names
<=128 bytes. Binding references preserve accepted existing names up to1024.
Address arrays are optional, never null/empty; named endpoint selections must be
nonempty and unique. All responses are no-store. Complete accepted policy retains
its unchanged4MiB/4096-object bounds; full inventory is bounded by existing input
limits, not silently truncated. Creation/read and Binding removal are available,
**not full CRUD**.

The transformation preserves existing policy bytes and facts, appending compact
JSON (valid YAML); the sole explicit-empty Configuration marker is replaced for
the first Resource. The exact context is looked up in current facts, and its
contract is copied; the caller cannot supply or widen execution settings. The
compiler already requires unique context profile per Agent. Existing full
Replace performs immutable-identity validation, revision CAS, transactional audit
and distribution. Audit failure rolls back all changes; SQLite reopen restores
the same source, projection and audit. No policy-file poller writes this database
authority, and file mode does not gain these management writes.
Removal uses parser document boundaries, retains surviving source bytes, and
strictly re-decodes/compares every resulting object against exactly one removed
Binding. Unsupported presentation is rejected without modifying the publication.

## Evidence and remaining work

API tests cover dual-stack placement/no automatic grant, unchanged source/facts,
strict validation, context collisions, RBAC/CSRF, CAS, audit rollback and durable
reopen. Browser acceptance covers actual resource→binding→Grant/revoke,
conflict/draft, failed reads, viewer denial and375px layout, alongside the existing
real Nginx/application authorization suite. Exact pass/fail, tested trees, review
and CI are recorded in [the checkpoint](AUTONOMOUS_DELIVERY.md) and PR18.

Resource/Endpoint editing/deletion, Binding editing, context enrollment, human
identity/requests/approval and final combined Linux workflow remain. HTTPRoute
creation and desired/applied node observations are already delivered. No new
kernel execution evidence or completed product is
claimed by this increment.
