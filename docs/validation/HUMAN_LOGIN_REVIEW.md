# Local human login review — 2026-10-19

One fresh-context independent read-only AI reviewer inspected the complete local
range9122dee8c1da15016551d461669f4bdfb69c9e1c..03f7fab523ef6a08da0608d7e3d65b149ae855d2,
tree311f66726dbd16b1494873e40f0a621cdfe35632, against AGENTS/domain docs, spec/plan
and ledger. No additional reviewer or delegated subreview was used. This is not a
fabricated GitHub/human approval. Reviewer independently ran uncached Go tests for
managementaccess/controller/cli/password and inspected UI/accessibility semantics.

The only Important finding was the new browser fixture's comment-only change:
canonical policy intentionally made it a no-op, so revision/audit proof was absent.
Native39196659364 Web117880796684 reproduced51vs52. Correction03f7fab adds a real
unbound Resource, preserving prior execution scope, and requires exact persisted
policy, revision advance and human audit actor. Reviewer inspected this correction
within the same review. No unresolved production Critical/Important or deferred
Minor finding. Corrected exact-head native CI must still be independently verified
before merge; the review explicitly withheld merge readiness until then.

## Executor rulings and costs

1. Retain the host-owned isolated linked worktree, including its branch identity;
   existing isolation is sufficient. Cost if wrong: workspace ownership conflict.
2. Use the existing private credential file for users, not a competing database
   identity authority. Cost: manual provisioning/rotation until account management.
3. Fixed bcrypt12, four concurrent checks and local peer limits bound this process.
   Cost: distributed abuse and reverse-proxy sharing still need ingress controls.
4. Review and final native CI may overlap, but neither substitutes for the other.
   Cost if wrong: native defects could be merged; therefore any failed gate blocks.
5. Use a semantic unbound Resource in the acceptance fixture, not a YAML comment.
   Cost if wrong: compiler/API/audit must reject it; do not weaken no-op/CAS behavior.
6. Public TLS deployment correctness remains out of this source-only increment:
   document trusted HTTPS ingress and direct-access restrictions without claiming
   supplied certificate/production evidence. Cost: operators must validate deployment.
7. Distributed quotas/shared sessions/per-account lockout remain excluded; local
   bounds were reviewed. Cost: single-process controls are insufficient for HA or
   global brute-force defense; do not deploy as though those capabilities exist.
8. OIDC/MFA/registration/reset/account CRUD/login-event audit remain explicit future
   capabilities, not silently implemented. Cost: this is not a complete identity
   platform. Publication audit still records real human accepted writes.
9. Native autocomplete/paste/label semantics are verified, not particular third-party
   password-manager products. Cost: vendor-specific compatibility remains untested.
10. No Agent/kernel implementation changed. Existing native integration evidence is
    retained; no new human-login kernel application claim is inferred. Cost if
    overstated: desired publication would be confused with independently applied rules.

Credential-type confusion, malformed hashes/CPU, enumeration bounds, role/rotation/
deletion races, mixed auth, CSRF/forwarded peers, Unicode byte limits, stale method
discovery and identical-record restoration were reviewed rather than excluded.
The native browser report and final immutable source/CI/artifact evidence belong in
AUTONOMOUS_DELIVERY.md and PR30; final metadata may supersede pending status here.

Post-review native39197009801 passed real semantic publication/readback/human audit,
then exposed an ambiguous test locator: retained hidden feature views carry the
same read-only message. The test is now scoped to the visible publication editor
and checks the actual textarea.readOnly property, matching existing viewer gates.
This is a test-only correction in the same verification/fix pass, not an additional
production feature or a new review verdict; complete native gates remain mandatory.

中文：独立全分支评审无遗留阻塞或延后小项；已复现并修正语义无效的浏览器夹具。
原生完整 CI 仍为合并前硬门槛；上述部署/账户平台/第三方兼容及内核声明边界保留。
