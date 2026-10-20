# OIDC sign-in review and verification

## One independent whole-branch review
Reviewer oidc_review read adfa1b1..31b6285, read-only, no reviewer delegation or
platform approval. Verdict **With fixes**:0Critical/2Important, no additional code
Minors. Independently ran focused four-package race successfully and inspected
pinned SCS/Echo/go-oidc. This is not full native/archive approval.

1. Important: cross-site callback omits Strict SCS cookie, so replacement must
   revoke originating token before navigation. Root verified actual RED:
   TestBrowserOIDCProtectedStartCorrelationAndSafeCallback captured old cookie
   remains200 after start. ONE root fix pass destroys originating SCS under the
   existing start request token lock, before returning authorization URL. Strict
   stays unchanged. New actual browser assertion seeds a real key session, checks
   its old cookie401 before cross-site callback and after eventual logout.
2. Important: invalid browser publication fixture mixed explicit-empty Configuration
   with Resource. Native39356233386/Web118411541723 and actual SHA-verified PNG
   demonstrate correct invalid_publication. Root replaces initial candidate with
   literal Resource; preserves decoder rejection, adds exact revision2/readback
   and real human audit attribution. Full new native green is mandatory.

Both fixes belong to the same root blocker pass; no second review. Exact final
suite/native/source/archive/main results and immediate lease release go in PR39
metadata. Pending statements here are not acceptance. Review head differs from
later corrected head; source identity must be checked, never inferred.

## Deferred minor verification breadth
Reviewer recommended direct fault-injection tests for oversized discovery/token/
JWKS bodies, redirects, slow responses and untrusted TLS. Source uses bounded1MiB,
5s, redirect refusal and normal verification, but dedicated adversarial transport
regressions are not in this slice. Deferred test breadth, not a known bypass or
claim that such fault injection ran; retain this in next provider acceptance.

## Root rulings on every declined judgment
| Reviewer boundary | Root ruling | Cost if wrong |
| --- | --- | --- |
| Full normal/race not independently run | Local normal/race only inherited Unix EPERM; root requires fresh full native Go/raw evidence | Hidden test/race failure |
| Complete native browser failing | Block merge until corrected full real Chromium scenario and prior suites pass | Unexecuted session/revoke claims |
| Later heads not reviewed | Root regression-first fix pass and exact full final checks; no reviewer approval attributed to later head | Unreviewed regression |
| Real IdP/MFA/consent/outage policy | Explicitly pending deployed-provider acceptance; protocol fixture is not production identity | Provider incompatibility/false readiness |
| Real HTTPS/Secure-cookie topology | Existing HTTPS ingress evidence not extended to OIDC; new deployed-IdP scenario still required | Cookie or proxy deployment failure |
| Other engines/assistive-tech/layout | Root inspects actual EN/ZH375/1440 screenshots; only Chromium execution claimed, not universal a11y | Unseen browser/accessibility issue |
| Unexecuted adversarial transport | Boundaries source-reviewed, direct test breadth deferred as above | Provider-induced disclosure/DoS |
| Installed OIDC/Agent/app/restore | Keep pending and document nested private paths/restore review; existing non-OIDC proof not extrapolated | Broken restore or incomplete access conjunction |
| External IdP revoke/logout | Intentionally unsupported; local revoke/ID-token absolute deadline documented | Unexpected external-session persistence |
| Identical file re-add without rotation | Explicit generation rotation required; database account durable version tested without intervening request | Old file-mode cookie resurrection |
| Archives/notices/remote/merge/main | Root independently checks actual bytes/static ELF/full manifest/source/head/raw, expected_sha merge and main | Wrong/stale or incomplete release |
| Dependency legal/security/HA/kernel | No general clearance/HA/kernel compatibility claim; no Agent/kernel code changed | Unsupported assurance beyond evidence |

Other execution rulings: routine design/plan pauses waived by explicit mandate,
not review/verification gates (wrong: unreviewed trust boundary). Local Chromium
download is a corrupted0MiB ZIP; use native mandatory browser gate (wrong: no real
UI proof). Test handler injection must precede route registration, matching runtime
(wrong: testing fictitious live config reload). Empty managed marker replacement
is fixture repair, never relaxed production validation (wrong: masked parser bug).

UI skills retained established palette/components and hierarchy, added truthful
organization action/help/loading/failure, EN/ZH and secret clearing; no new visual
stack, sample data or primitive changes. Actual screenshot review is root-owned.
Whole basic goal remains active; next deployed IdP/installed full conjunction and
OIDC stopped-state restore, not repeated old provider/packaging foundation.
