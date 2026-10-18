# Structured resource onboarding

## Intent and boundary
The operator should create a protected service with named TCP/UDP entries and
place explicit entries on an existing, operator-declared host context without
writing policy YAML. This is desired configuration, never proof of live Agent
application. User authorized routine design, implementation and integration.
Retain the selected Go/Echo/GORM/Casbin/SCS and React/TypeScript/Vite/shadcn/
Tailwind/i18next/npm stack; no dependency or Agent/protocol change.

## Approach
Use the existing sole database publication, immutable identity validation,
expected-revision transaction, audit and distribution. A separate resource table
would duplicate authority; a generic object editor would expose too much lifecycle
surface. Instead add two strict create operations and a complete bounded inventory.
Existing resource mutation/deletion, selector editing, context enrollment and
HTTPRoute editing stay in the advanced publication editor for now.

GET /api/v1/inventory returns authority, revision, observed_at, resource entries
including destination restrictions, existing Binding scopes/selections/settings,
and configured execution contexts. No online/applied inference from facts.
POST /api/v1/resources accepts name, endpoints (name/protocol/port and optional
nonempty addresses) and reason. Server generates resource UID. No rename/replace.
POST /api/v1/bindings accepts name, pinned resource name/UID, nonempty unique
endpoint_names, agent_name, context_id and reason. Resolve that exact context in
current facts; copy its profile/class/stage/destination/handling contract, only
host-local/original-destination. Never accept caller-authored execution settings.
Use one Resource and one explicit Agent per new Binding; broader imported scopes
remain faithfully readable. Existing full compiler rejects overlap/capability/
reference failures. No automatic grant is created by resource or binding creation.

Mutations require current publication:write, strong If-Match before transformation,
nonblank reason <=512 bytes, strict JSON <=16KiB, no unknown/duplicate/null fields.
Names <=128, 1..64 endpoints, TCP/UDP, port1..65535, address list omitted or1..64
valid IPv4/IPv6 IP/CIDR, endpoint names <=128. Binding lists max64, duplicates fail.
Append compact JSON documents without normalizing existing source; replace the
sole explicit-empty marker when first resource is added. Preserve facts exactly.
No policy byte/object-limit increase (4MiB/4096). Read arrays are complete within
existing publication/facts bounds; UI renders at most50 records per section/page.
Equivalent /console/api/v1 routes retain session/CSRF/Origin/current role checks.

## UI
Add Resources navigation with resource and placement lists, source scope/context
details and two labeled forms. Resource form supports add/remove endpoint rows;
destination field explicitly distinguishes omitted (binding scope, both families)
from nonempty restrictions. Binding selects existing resource, named endpoints and
context; show exact agent, profile, destinations and handling before confirmation.
Existing selectors are displayed as dynamic, never converted to named selection.
Reuse current component/theme/a11y/i18n rules; no sample data. Loading/error/empty/
last-read/stale states; read refresh never advances a preserved mutation CAS base.
Explicit rebase preserves drafts, 412/ambiguous errors block retry until rebase;
logout/unload warns, switching tabs preserves drafts, cancellation ignores stale
responses. Viewer can read but backend denies writes. No automatic mutation retry.

## Acceptance
Real API/SQLite create, context mapping, facts preservation, compile rejection,
CAS/RBAC/CSRF, audit rollback and restart; source representations stay unchanged.
Real Chromium creates resource then binding, sees it in Grant selection, issues
and revokes a Grant; verify persisted configuration/audit and projected target.
Existing real Nginx HTTP acceptance stays mandatory. Conflict/read-failure/viewer
and375px layout cases execute. Full ordinary/race/vet/build and locked Web checks,
one independent whole-branch review, exact-source CI and expected-head merge.
Local Unix EPERM is an environment limitation, not a skip or a passing suite;
native Linux CI must execute it. Full product/combined kernel acceptance remains.
