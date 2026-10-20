# Installed OIDC restore review

One independent read-only whole-branch review, oidc_restore_review, inspected
72ab5d0..ef55027 (11 files), spec/plan/ledger and inherited service/CI plumbing.
Verdict **With fixes**: 0 Critical, 1 Important, 1 Minor. Reviewer independently
ran 21 Node tests, module syntax checks, diff check and synthetic disclosure
reproduction; inspected extracted service report/image. Root supplied native CI,
raw logs, actual archive/static/source evidence, not reviewer platform approval.

## One root blocker-fix pass

Important: provider-object and binding-byte equality assertions attach complete
private actual/expected documents to public failure stacks. Literal credential
redaction cannot protect unknown generation/subject/other private fields.
Root retained the finding: it is an evidence-confidentiality defect, not a known
production authorization bypass. After behavior-preserving extraction (21 green),
two synthetic mismatch regressions were actually RED because private markers
survived in the stack. The helper now deep-compares to a boolean with a fixed
diagnostic. Both mismatches still reject; equal independent values pass; neither
document's subject/principal/generation/issuer/client/secret appears in the stack.
Combined Node suite: 23 passed, zero skipped. No second review or approval of the
later root fix is attributed to the independent reviewer.

Minor deferred: installed-oidc-restored-1440.png is actually 1280 pixels wide.
Report its real dimensions; do not claim a 1440 viewport. No UI layout changed.

## Root rulings on every declined judgment

| Reviewer boundary | Root ruling | Cost if wrong |
| --- | --- | --- |
| Production IdP/MFA and HTTPS/Secure cookies | Keep deployed-provider/HTTPS acceptance pending; signed fixture is not production identity | False deployment readiness |
| Installed Agent/kernel/application conjunction | No Agent changes; retain separate real conjunction gate | Identity mistaken for enforcement |
| Full Go/native systemd not independently rerun | Root requires fresh final native normal/race/vet/build and all install gates; local Unix EPERM is not green | Hidden failure or race |
| Four archives/provenance/static/notices | Root verifies downloaded bytes, complete manifests, source and static ELF, not artifact names alone | Wrong release evidence |
| General backup/external paths/concurrent mutation | Intentionally unsupported; exact complete stopped instance and controlled parent only | Unsafe recovery outside contract |
| Exhaustive inherited OIDC verifier review | Extraction equivalence and new assertions reviewed; no general security clearance claimed | Unsupported assurance |
| Unobserved binding removal/re-addition | This scenario proves observed withdrawal only; inherited generation boundary remains documented | Old cookie resurrection assumption |
| Universal visual/accessibility quality | Actual restored/mobile/Chinese screenshots inspected, no UI change or universal claim | Unseen accessibility/layout defects |
| Final merge/main/post-merge artifacts | Expected-head merge and fresh exact main raw/archives required before release | Stale or unvalidated merge |
| Absolute historical log confidentiality | Fix this reproduced failure path; no blanket historical-log claim | Undetected disclosure elsewhere |

Other rulings: routine approval pauses waived by explicit mandate, not verification
(wrong: unreviewed trust boundary). Plan final integration is not a prerequisite
for starting its own review (wrong: premature completion). Existing account UI
creates requester plus checked admin; exact literal role union corrected after
native RED39371306721, not production validation weakened (wrong: privilege error).
Local extracted issuer smoke failed through configured proxy; no bypass or local
success invented (wrong: false protocol proof). Existing host-owned worktree and
other work preserved (wrong: loss of user changes).

Candidate39371678322, remote2909449/tree9cd46b, has all3 jobs/steps green and raw
260 top-level/660 total Go tests, no fail/skip/race. Root checked all4 actual ZIP
hashes/sizes, 244-file manifest/static binaries/notices, 10 private restore files,
four uid999/caps0/NNP1 processes and strict old/new service/ingress receipts. This
is pre-review-fix candidate evidence, NOT final corrected-head acceptance.
Final immutable commit/PR/CI/source/archive/main receipts and immediate lease
release belong in PR40 metadata, superseding pending source paragraphs. Whole
goal remains active: real deployed IdP/HTTPS and installed full conjunction next.
