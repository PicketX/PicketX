# Safe Resource definition removal

Remove an unused Resource by immutable UID through the real management API and
Resources console. This cleans up a definition, not a running service. Reject any
incoming Grant, Deny, EnforcementBinding or HTTPRoute reference, including revoked,
expired and dynamic Endpoint selections. Never cascade, rewrite references or
infer that an inactive contribution can be discarded. Existing alpha references
are pinned Resource name/UID; future unknown object types fail closed.

POST /api/v1/resources/{uid}/remove and /console/api/v1/resources/{uid}/remove
accept strict JSON16KiB `{reason, acknowledge_removal:true}`. Preserve current
RBAC/session/Origin/CSRF and strong If-Match: missing428, stale412 before lookup,
missing/wrong-kind404 resource_not_found, referenced409 resource_in_use. Success200
returns uid/revision and strong ETag. Full compilation, identity checking, CAS and
transactional audit remain the only publication path. Facts/unrelated source bytes
survive; sole Resource becomes explicit-empty Configuration. Unsupported source
encoding rejects without mutation. No schema migration, dependencies or Agent change.

Use the existing one-draft Resources form, not a new screen. Resource row action
opens named UID/Endpoint review, reason, explicit confirmation and cancel. Preserve
draft on errors/tab changes; read refresh never rebases. Explicit rebase refreshes
same-UID details and resets acknowledgement. Missing target blocks confirmation.
Reference conflict explains which object classes must first be reviewed manually;
do not promise an exhaustive reference list. Match UID, advancing safe revision and
strong ETag before clearing any removal draft. Ambiguous success/transport/5xx,
404 and412 block retries until rebase; no automatic mutation retry.

Retain system sans16px, background#f5f7fa, foreground#18283b, action#145da0,
existing semantic danger/error tokens,24px form gaps and44px controls. Left-aligned
operational workbench; Endpoint details convey deletion scope, not decorative cards.
Frontend-design review: reuse established palette/layout instead of inventing a
new visual identity for one action. UI/UX search verified confirmation and success
feedback guidance; retain focusable errors, keyboard labels and375px responsiveness.
English/Chinese copy must distinguish Resource cleanup from Binding unprotection.

Acceptance: real SQLite HTTP removal/reference guards, rollback/CAS/auth/CSRF,
source preservation with direct DB query (also closes PR24's cached-only assertion),
real Chromium create/remove/cancel/conflict/rebase and referenced refusal. Keep
existing real Nginx/application acceptance. Native full Go/race/vet/build, locked
Web checks and one independent whole-branch review before expected-head merge.
Local Unix EPERM is not a passing suite; native Linux must exercise it. Resource
editing, reference lifecycle, human identity/approval and combined kernel/browser
product acceptance remain outside this slice.
