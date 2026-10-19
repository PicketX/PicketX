# Access requests — independent whole-branch review

One fresh-context reviewer (`access_requests_review`, gpt-6-astra/high) reviewed
local00c97290..91682bc, source/tests/migrations/generated API/bilingual docs/spec/
plan/ledger. Read-only; no delegated review or checkout changes. It independently
ran `go test ./internal/publication ./internal/controller ./internal/controllerapp
./internal/managementaccess -run 'AccessRequest|RequestPolicy' -count=1`: all four
packages passed. Structural OpenAPI comparison found no changed pre-existing
paths/schemas. Browser/native artifact gates were not independently rerun.

Initial verdict: **not ready**, no Critical, four Important and one Minor. Root
accepted the four blockers and performed one fix/verification pass, no re-review.
This record does not substitute for final exact-head native CI evidence in PR32.

## Findings and coordinator rulings

| Finding | Ruling and regression | Cost if wrong |
|---|---|---|
| Important: read cleanup retains busy; re-entry GET can unlock pending POST | Track outstanding read/write controllers. Native39208177332/Web117915470482 RED: leaving canceled request read leaves logout disabled. Retained GET and POST/navigation tests must pass on final native head. | Duplicate durable requests or stranded session controls |
| Important: catalog requires creation even for independent approver; create-only requires history | Catalog permits create OR review+publication-write in both service/HTTP; UI requests only permitted reads. Real custom Casbin create-only/review-only/approve-only/write-only matrix; reviewer-with-create-deny RED403→GREEN200; browser custom roles use real persisted policy/reload. | Legitimate users blocked or unrelated data exposed |
| Important: same UID falsely validates historical approval after advanced edit | Compare source, pinned named scope, bounded deadline and selected network; report modified or scheduled without new scope leakage. Real advanced source/scope/expiry/NotBefore edits RED valid-desired→GREEN; persisted edit/restart tested. | Historical grant intent mistaken for current desired permission |
| Important: raw post-commit readback failure skips fail-stop/delivery | Remove fallible post-commit readback. Create resolves generated ID inside transaction; decisions return known committed record with stored ID. Actual SQLite Row callback fails outside-transaction read after event/publication write: create/approve/reject RED→GREEN, actual Agent service desired revision and reopen match. | Ready runtime diverges from durable publication |
| Minor: Chinese reason character limit vs UTF-8 byte budget | **Deferred** in this pass; server byte budget remains enforced, no broader input accepted. Follow up with explicit bilingual byte counting/field errors. | Some valid-looking Chinese form content gets generic rejection |

## Reviewed dispositions — root rulings, not reviewer silence

1. Accept self-rejection: attributable and grants nothing. Cost: users can end a
   pending own request, a documented behavior, not approval authority.
2. Accept catalogued Resource visibility as chosen alpha eligibility model;
   per-Resource eligibility remains deferred. Cost: catalog metadata visibility.
3. Accept numeric ID gaps without peer contents; cursor privacy redesign deferred.
   Cost: aggregate activity volume can be inferred.
4. Accept identical UID+network reuse as the existing identity model; new UID or
   changed shape conflicts. Cost: identical historical disappearance undetectable.
5. Accept facts/binding/label changes outside frozen network as deployment changes,
   not approved topology. Cost: operator must inspect actual execution placement.
6. Keep conservative revalidation of all originally requested names, including
   omitted approved names. Cost: unnecessary resubmission after unrelated change.
7. Accept final-fsync deadline crossing: unchanged absolute expiry creates no
   post-expiry permission. Cost: no strict wall-clock commit-boundary guarantee.
8. Accept once-only additive new built-in defaults, preserving existing rules and
   denies. Cost: operators must inspect documented new operation defaults.
9. Keep advanced publication writers' existing Grant-edit authority; fix observed
   provenance, not forbid their edits. Cost: workflow history is not immutable
   current configuration; explicit modified/scheduled state is necessary.
10. Keep existing request-time identity/periodic policy boundary for in-flight role
    changes. Cost: no instantaneous mid-operation revocation promise.
11. Defer retention/cancel/notifications/OIDC/HA/new privileged approval acceptance;
    do not call the full product complete. Cost: more delivery increments required.
12. Root owns exact native/browser/Unix/artifact proof. Old-suite label collision
    fixed with a distinct requested-source label, not weakened locator/guards.
    Cost: review alone never establishes deployability.

## Verification honesty

Initial native39206258903 had invalid `person:` browser fixture IDs and stale
schema3 smoke expectations;39207222350 hit hidden-view source-label ambiguity.
Neither was accepted. IDs corrected to existing `principal:` contract, separate
request label, schema4/new-table/RBACschema2 assertions retained alongside every
old static check. Local parallel race once hit the existing coalescing deadline
under heavy concurrent bcrypt/race work; complete serial race rerun only failed
the known local Unix socket EPERM, with no race report. Final native gates remain
mandatory, including all Unix tests, static artifacts and previous browser/Nginx.

中文：独立全分支评审发现四项阻塞，按真实失败回归修复并重新验证；不伪造二次评审或
原生执行结果。中文原因字节计数反馈作为 Minor 延后，完整证据以 PR32 为准。

## Accepted code verification — 2026-10-19T06:31Z

Local66fb33072f5cf0d2e7c6673c0fc2a2003162f374, remote
dc55a3b972d75f748dbd9644c461accc6f6f2978 and actual Go/Web CI checkout
fd8bb9c5f6819a16b978593cad4e6712f7aa8490 share tree
ee9b0bac97066d68ab911d8c5be34349337614d6. CI39209796875 Go117920419720 and
Web117920420197 are SUCCESS. Raw Go231 top-level/581 total, zero fail/skip/race;
all Unix100repeat, protocol/OpenAPI, vet/build, static Controller/CLI, real SQLite/
gRPC/restart/SIGTERM gates pass. All old Chromium/Nginx suites and new separate
human requester/admin workflow, held GET/POST navigation, custom create-only and
approve-only roles, approval/rejection/revoke assertions pass. Screenshots375/1440
visually inspected; artifact12166992084 downloaded ZIP independently matches
SHA2568b125e42978241fec4192276cb744e783d795f813f5391fa265ea10327e1a533.

Harness corrections did not weaken production: real429 consumes the same peer's
password burst after preceding tests; wait10.1s for configured0.1/s refill. Retained
request CAS requires explicit rebase before a later submission. Held route cleanup
waits for actual continuation before unregistering, avoiding double-handled routes.
Prior failed runs remain failures. Unchanged coalescing test alone passes race
16.489s; complete local normal/race failures stay documented. Final documentation
head still requires full exact-head CI, expected-head merge and green main.
