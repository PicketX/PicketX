# Real-provider HTTPS review

One independent read-only whole-branch review, real_idp_review, inspected
0b3d8c1..a7df1cd, all ten changed files, spec/plan/ledger and AGENTS.md.
Verdict **With fixes**: 0 Critical, 1 Important, 0 Minor. Reviewer independently
ran all 30 focused Node tests, syntax/diff checks, inspected the successful actual
1440px screenshot and public provider receipt. Native CI/archive verification was
supplied by root, not independently rerun or a platform approval.

## One root blocker-fix pass

Important retained: --db=dev-file ran directly from the caller's extracted
Keycloak distribution. Deleting only its realm import and the unrelated private
Controller/CA directory left provider database/runtime state outside the owned
private tree. This is a CI private-state lifecycle defect, not evidence of
artifact leakage or a production compromise; upload paths exclude that directory.

Root first extracted the original home-selection behavior with the existing seven
TLS tests green. Two actual filesystem lifecycle regressions, success and failure,
then failed because a runtime file survived owned-tree deletion. The minimal fix
copies the verified pristine input into a unique subtree under the owned private
0700 directory, preserves executable files, and starts/imports only that copy.
The input is never run, mutated or recursively deleted. Owned processes stop before
the whole tree is removed; success checks absence before emitting its public
receipt, and finally removes owned state on failure. An initial cp-to-existing
directory EEXIST was corrected by copying into an absent child of the unique private
container; no overwrite option was enabled. Both lifecycle tests now pass; combined
Node tests: 32 passed, zero skipped. No second review of root's fix is invented.

## Root rulings on all declined judgments

| Reviewer boundary | Root ruling | Cost if wrong |
| --- | --- | --- |
| Production Keycloak hardening, DB, HA, MFA and availability | Disposable CI provider only; no production suitability or deployment instructions inferred | False readiness or unsafe production deployment |
| Other IdPs, versions and custom themes | Exact Keycloak26.8.1/default theme only; OIDC product selection remains configurable | Unsupported compatibility claim |
| Provider logout/account revocation/SSO termination | Test current local account roles, binding withdrawal and local SCS logout; provider SSO intentionally remains live | Local revocation confused with IdP-wide termination |
| Installed Agent/kernel/network/application enforcement | No Agent changes; preserve existing separate real gates and require final installed OIDC conjunction next | Login/publication mistaken for enforcement |
| Full UI accessibility/responsiveness | No product UI changes; inspect actual 1440px public screenshot using UI/UX guidance, no universal claim | Unseen usability/accessibility defects |
| Final head, expected-head merge and fresh main | Require new full native CI/raw/four downloaded archives/source/static/notices/images on corrected head and merged main | Stale or unvalidated release evidence |

Other root rulings/costs: explicit autonomy waives routine confirmation, not review
or validation (wrong: unreviewed trust); reuse isolated host worktree without
overwriting user state (wrong: lost work); pinned upstream archive/JDK21 is CI-only,
not a product dependency (wrong: unintended platform/vendor requirement); private
disposable CA/NSS label with normal CA/SAN verification is test setup, not product
mTLS/internal CA (wrong: trust residue); default v2 theme field+aria assertions
replace the obsolete global selector, not password-denial assertions (wrong:
fabricated authentication proof); task completion precedes final integration gates
without satisfying them (wrong: premature completion). All five review-focus areas
were checked: TLS, explicit binding, cookie boundaries, current local roles with
live SSO, and private lifecycle. No new Minor was reported; inherited deferred
items remain in their original review records, not silently closed here.

Initial native39376573835 and diagnostic39377032991 were RED at actual provider
password handling. Upstream v26.8.1 keycloak.v2 login/field templates established
input-error-username plus aria-invalid=true. Candidate39377577216, head da59ccb /
tree6a956acf, passed all three jobs/every step and actual real-provider HTTPS flow;
Go260 top-level/660 total no fail/skip/race, Node30/HTTP11, old Web/installed/service/
restore/ingress gates and all four actual archives verified. This is **pre-review
fix candidate evidence, not corrected-head acceptance**.

Local fresh full normal/race still hit inherited Unix socket EPERM; vet/build,
locked Web checks and focused tests are separately recorded, never called a full
local green run. Final immutable source/PR/CI/archive/main receipt and immediate
lease release belong in PR41 metadata, superseding source pending paragraphs.
Overall goal remains active until installed OIDC/Agent/ingress/application closure.
