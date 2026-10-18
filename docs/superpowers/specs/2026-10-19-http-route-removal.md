# HTTPRoute removal

## Intent and boundary
Operators need to retire a configured L7 entry without manually editing the entire
publication. Extend the existing create/list lifecycle, not the authorization model.
The user preauthorizes routine design/execution/push/merge; implement inline with a
fresh independent whole-branch review. No reapproval pause is required.

Choose UID-specific source-preserving removal through existing Replace/CAS/audit.
Alternative full-source editing remains available but is error-prone for this task;
cascading Resource/Grant deletion would change independent permissions and is rejected.
Resource/Endpoint/Binding editing, human identity/approval and combined Agent/kernel
acceptance remain next work, not completion claims for this increment.

## API and semantics
POST /api/v1/http-routes/{uid}/remove and /console/api/v1/http-routes/{uid}/remove
accept strict JSON16KiB {reason, acknowledge_removal:true}. Current publication:write,
strong If-Match and existing browser Origin/CSRF gates are required. Check revision
before UID lookup; absent/wrong-kind UID404, stale412. Initial unpublished revision0
uses the existing invalid_publication400 behavior; no silent no-op success.
Return200 {uid,revision} and the matching strong ETag. No caller-supplied revision,
actor or replacement UID. Preserve unrelated source bytes and all facts. Commit the
publication and audit atomically; failure leaves revision/source/audit unchanged.
Remove only the exact HTTPRoute; all Resources, Grants, Bindings and other routes
remain. The caller's runtime route allowlist remains unchanged but a missing route
must deny. Existing ingress configuration and TLS remain operator-owned. This is not
Grant revocation, a network firewall withdrawal or existing-stream termination.

## UI
Use the existing route page and one draft at a time: explicit Remove HTTP route
action opens a review with immutable UID/resource/endpoint/host/method/path details,
reason, acknowledgement and Cancel. Preserve a conflicting draft and target UID;
refresh never rebases it, explicit rebase resets acknowledgement and looks up the same
UID only. Missing targets stay blocked. Malformed/ambiguous success receipts retain
the draft and require rebase; follow-up read failure never repeats an accepted write.
Viewer has no mutation control. Loading/failure/empty/stale and bilingual labels
remain explicit. Retain existing left-aligned operational workbench, system sans
16px/1.5, palette #f5f7fa/#18283b/#145da0/#526174/#a52222; existing shared tokens and
44px controls, no decorative redesign. UI/UX search matched destructive confirmation
and explicit success feedback. Root verifies375px/1440px native screenshots.

## Acceptance
Real HTTP/SQLite tests: exact UID, source/facts durability, other objects unchanged,
missing/wrong kind, CAS, strict JSON, RBAC, CSRF and audit rollback. Shared source
helper retains its existing YAML-boundary regression suite. Real browser/Nginx test:
active source Grant permits the route; cancelled removal leaves access; accepted
removal denies new requests even though Grant and caller allowlist remain. Another
route to the same Endpoint still permits its declared request. Check stale/rebase,
malformed receipt, external disappearance, read failure, viewer and mobile states.
Full Go ordinary/race/vet/build and locked Web typecheck/lint/build, actual native
Unix/static smoke and real browser suite must pass before expected-head merge; verify
main CI. No skipped/mock kernel or fixture-only end-to-end claims.
