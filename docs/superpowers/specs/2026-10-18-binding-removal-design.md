# Explicit Binding removal

Deliver one bounded lifecycle operation: remove an EnforcementBinding by immutable
UID, never a Resource or Grant. Removing protection is NOT revoking access. The
last matching Binding can leave services outside PicketX protection; independent
Bindings and all Grants remain. No endpoint edits, resource deletion, automatic
Grant revocation or new Agent/kernel behavior.

POST `/api/v1/bindings/{uid}/remove` and cookie equivalent accept strict <=16KiB
JSON `{reason, acknowledge_unprotection:true}`. Require publication:write, strong
If-Match, existing SCS/CSRF/origin controls. Missing/wrong-kind UID is404; stale
revision412 before lookup; missing acknowledgement400. Reason nonblank <=512
bytes. Existing complete compiler, CAS transaction, audit and distribution remain
the sole authority. Return200 UID/revision/ETag, never applied confirmation.

Transform source by parser document boundaries, retaining surviving documents
without reserialization/size expansion. Re-decode and compare ALL resulting
objects against precisely one removed Binding before returning. Keep existing
4MiB/4096 limits; unsupported/ambiguous presentation fails without publication.
Removing the only object produces explicit empty Configuration. Facts unchanged.

The Resources page offers an explicit removal draft, with selected name/UID and
full selection/enforcement details, danger warning, reason, acknowledgement,
confirm and cancel. Share the existing single-draft/busy/dirty/revision rules:
switching discards only with confirmation, refresh never rebases, explicit rebase
preserves reason but resets acknowledgement and uses current UID details. Missing
target blocks confirmation.412/ambiguous write blocks retry; successful write
is not repeated after refresh failure. Viewer sees no action. English/Chinese.

Use existing system-font16px, background #f5f7fa, foreground #18283b, action
#145da0, white surface and existing danger tokens. Left-aligned, single-column
review panel within Resources; no modal/animation/new design system.44px controls,
focusable error summary and mobile375px without overflow. UI skill confirmation
guidance applies to protection withdrawal, not a misleading access-revoke button.

Acceptance: real SQLite API rejects unsafe input/auth/CAS/audit failure atomically;
two overlapping Bindings preserve remaining protection; last removal withdraws
protection but retains Grants. Source tests cover first/middle/last documents,
Unicode line breaks, JSON, comments and large annotation. Real browser/API test
covers warning/cancel/acknowledgement, conflict/rebase, success/retained Grant,
viewer denial/audit/mobile. Native Linux CI remains required for full existing
Unix/static and browser/Nginx suites; no new kernel application claim.
