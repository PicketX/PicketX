# Durable access requests and independent approval

Deliver one live workflow, not a generic workflow engine: a trusted requester asks
for one explicit source CIDR, one pinned Resource and named Endpoints for60..86400
seconds. The server fixes the requested absolute deadline once. Requests create no
permission. A different authorized actor rejects or approves a subset of those
names with an absolute deadline no later than requested. Approval publishes one
independent bounded Grant; it never asserts application. Requested content stays
immutable alongside the accepted content, actor, reason, Grant UID and revision.

Reuse the existing SQLite authority, GORM, Casbin, SCS/CSRF and publication pipeline.
Schema4 adds durable requests and append-only workflow events. Approval's pending
CAS, workflow event, Grant publication and publication audit commit in one database
transaction, after full candidate compilation/preflight; distribute only after
commit. A failed commit leaves no approval or Grant. Duplicate/concurrent decisions
cannot create a second contribution. Persistence/delivery failure retains existing
fail-stop behavior. No file-provider writes, Agent/protocol changes or new packages.

`role:requester` receives request:create/read-own and a limited resource catalog,
not policy/facts, other users' requests, audits or publication writes. Admins gain
review/read-all; approving also requires publication:write. Every operation uses
current trusted RBAC, never client requester/roles. Existing viewer/key/human paths
remain. Default policy migration is versioned and never replaces custom rules;
explicit denies still win. No user-directory mirroring into Casbin.

Strict JSON16KiB, explicit field/collection limits and bounded paginated queries.
Submission/approval require strong publication If-Match; decisions include the
pending record's version. Selected Resource/Endpoint network shape is frozen at
submission and checked before approval; removed/recreated/retargeted entries cannot
silently receive approval. Reject remains possible after target removal/expiry.
Source is explicit requested input, never inferred from portal/forwarded headers;
the form explains shared reachability, routes and application-auth boundaries.

UI: existing EN/ZH console shell, Requests view, real limited catalog, named
checkboxes, explicit source/duration/reason, own history, separate admin review.
Show requester, source, requested scope/deadline, decision/approved scope/deadline,
Grant/revision and 'not confirmed applied'. Loading/empty/error/age states, refresh,
pagination, keyboard labels/focus, mobile375/desktop1440. Unknown write outcomes
retain intent and require reconciliation before retry, never automatic resubmission.
Request-only users must not trigger privileged background inventory requests.

Validation: RED→GREEN HTTP/business/storage tests for separate identities, own-only
visibility, both families, strict inputs, self-approval denial, narrowing, expiration,
scope replacement, pending CAS, migration/restart, commit rollback and existing
revision1 stale UPDATE regression. Actual browser two-person submit/review/approve/
reject/revoke/audit with real SQLite/Controller; keep old native gates. Full Go
test/race/vet/build and locked Web typecheck/lint/build; one independent whole-branch
review, exact source/tree/CI proof, expected-head merge and main CI verification.

Boundaries: local file-provisioned users remain; OIDC/account lifecycle, request
cancellation/notifications, eligibility by resource, HA/PostgreSQL and combined
new approval→privileged Agent acceptance are subsequent increments. No production
readiness or full-goal completion claim follows. Fixed named scope only; no dynamic
selectors/permanent grants. User standing autonomy covers design/plan execution.

中文：新增真实持久申请及独立审批，不将申请人当作来源，不将审批当作已应用；
同一事务提交审批、Grant、审计和发布，禁止自批及扩大范围/期限，真实浏览器验收。
