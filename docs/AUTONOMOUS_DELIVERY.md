# Autonomous delivery checkpoint

## Node observations (active)

Owner codex/node-observations; heartbeat2026-10-18T17:45Z; lease until19:15Z or
merged+green main/explicit handoff. Existing isolated branch codex/resource-onboarding.
Core PR20/main819d343dd107bf1a32a8ba1ad769fb933964b068 accepted, tree184b1176c1a2aa4abde3a0863665bcad1f21756a,
CI39148069494 GREEN172/434 and real browser/Unix/static gates. PR20 lease released.
Agent95a5f7f/CI39118687475 unchanged; neither repo has an open PR at acquisition.
Plan: docs/superpowers/plans/2026-10-19-node-observations.md. Implement atomic
credential-scope report/current-session/freshness projection, authenticated API
and real Nodes page. No kernel proof or protocol heartbeat claim. Preserve PR20
deferred minors and remaining lifecycle/identity/approval/combined Linux acceptance.
Do not compete while this delivery is active. Overall task remains incomplete.

PR21 browser RED CI39151001045 / Web117744459046 failed exactly on the missing
Nodes navigation after all existing real browser/Nginx scenarios passed. Its Go
job caught unformatted new Swag annotations; gofmt corrected them. Remote upload
initially lost the generator executable bit; f6f50d85 restored it and matched
local tree70a5f5346d221db3dbf68ed4fddb20024a64bb45. No forced ref update.
API/service tests cover current-session freshness, unknown/stale/disconnected,
decimal revisions, credential removal, live auth and actual gRPC→HTTP reports.
UI uses existing visual tokens and read-only cards, no applied-access success
claim; freshness helper missing-module RED→GREEN covers boundary/clock/suspend.
Browser credential fixture corrected to the existing agent_ref JSON contract.
Local complete ordinary/race tests encounter the known Unix socket EPERM;
native CI must pass that scenario. Full review, final CI and merge remain pending.
Usage and limits: [Node observations](NODE_OBSERVATIONS.md).

## HTTPRoute console (active)

Heartbeat2026-10-18T16:42Z; owner `codex/http-route-console`, existing isolated
branch `codex/resource-onboarding`; lease until18:15Z or merged+green main/explicit
handoff. PR19 accepted main38e4f93c02a1c1fb72131b612a315c2fd6207a44,
treeed5135eff3f2c1ad09980f13846f55729786a84c; mainCI39142252279 Go117718888580/
Web117718888927 GREEN167/429, zero fail/skip/race, actual Unix plus100 repetitions,
static SQLite/managed runtime and real Chromium/Nginx/application checks passed.
PR18/19 leases released; older active headings below are historical. Agent
95a5f7f/CI39118687475 unchanged GREEN. No open PRs at acquisition.
Plan/context map: docs/superpowers/plans/2026-10-19-http-route-console.md.
Structured exact HTTPRoute create/read API+page only; preserve authority/CAS/audit,
facts and explicit caller/ingress scope. Actual browser→API→Nginx→application
must pass. Local full Go baseline only Unix socket EPERM; native gate mandatory.
Remaining: lifecycle, enrollment, identity/approval, fresh desired/applied status
and combined privileged Linux product acceptance. Do not pause the overall task.

[PR20](https://github.com/PicketX/PicketX/pull/20) code evidence:
local84588c8b93a60a5893aca67b3a10145f43fd26f8, remote
f43fcdf564b1de60f86b6916a30399fe8ba7b0b7 and actual CI checkout
23c3ab184585330c40da0866bb872c581089870d share
tree708ae952da90843ce3980e671876ff5c96d60f3c. CI39147016947 Go117732810809/
Web117732810602 GREEN:172 top-level/434 total, zero fail/skip/race; actual Unix
and100 repetitions, full vet/build, pinned protocol/OpenAPI, static SQLite file/
managed Controller/CAS/audit/restart/SIGTERM and CLI checks passed. Local complete
ordinary/race ran with only known Unix socketEPERM; never claimed local full green.
Locked npm ci/typecheck/lint/build passed locally and natively.

Real Chromium route creation→explicit generated UID caller allowlist/Nginx mapping
→source deny/Grant/application/revoke passed, plus exact Host/method/path/source
isolation, CAS412/retained draft, failed reads, Viewer and375px layouts. Existing
Grant/Resource/Binding and Nginx IPv6/union/deny/deadline cases remain green.
API404/OpenAPI and browser missing-navigation RED→GREEN (browser RED
CI39146482257/Web117731241046). Artifact12143592283 downloaded, SHA256
a9e4bdcac115e28fd914ade466a8a186ca5a9efc435cb6b9b92cfc7f6c4195c0 matches API;
desktop, mobile create/error and Viewer images inspected.

One fresh independent whole-branch review94d615e..2840135 is ready to merge,
no Critical/Important findings. Reviewer independently passed HTTPRoute controller
and full configuration/publication tests, inspected mobile image and test source;
remote raw full CI/browser were independently verified by executor, not rerun by
reviewer. No implementation changes after review; remaining changes are docs.

Deferred minors (nonblocking):
1. Tab activation does not refresh route observations; ordinary refresh does not
   update an initialized target catalog/base even with pristine draft. Explicit
   Use latest route revision works and CAS is safe. Follow-up: refresh on entry
   and auto-rebase only pristine/unambiguous drafts, with create-Resource→route test.
2. Add browser regressions for committed POST/lost response and successful POST/
   immediate inventory-read failure; assert no duplicate POST, correct retained
   versus cleared draft and explicit rebase. Current code is conservative, but
   those exact failure branches are not directly covered by browser faults.

Rulings (review scope exclusions and costs):
- Keep caller/ingress credential/configuration separate trusted deployment authority;
  cost: initial setup still needs explicit operator configuration.
- Route edit/delete remains advanced configuration; cost: incomplete structured lifecycle.
- Verified person/OIDC/approval remains separate from source routing; cost: no user self-service.
- Live applied status/kernel/TLS/active-stream termination remain unchanged and
  unclaimed; cost: no combined privileged Linux or complete production acceptance.

Final documentation head CI, expected-head merge and main CI still required.
Immutable final SHA/tree/CI evidence is appended to PR20 after execution; this
lease releases automatically only after merged+green main. Next prioritize fresh
desired/applied node status (with explicit stale/disconnected/unknown states),
then lifecycle/enrollment/identity/approval and combined Linux acceptance.

## PR18 main CI repair (active)

Heartbeat2026-10-18T15:43Z; owner `codex/resource-onboarding`; lease extended
until16:30Z or repair merged+green main/explicit handoff. PR18 merged as
2de0d6a8c9e52ed40cd68b6486b7bd5fccd29bad, tree9c59b871298cb5a86757357642e7bddab2def5e9,
after exact-head CI39140789757 passed Go167/429 and real Web/Nginx checks.
MainCI39141222771 Web117715886102 passed, but Go117715886276 failed the real
Unix permission assertion: the test observes the raw Listen callback before
runtime chmod0600 finishes. This is not local EPERM and not green acceptance.
Repair plan/context map: docs/superpowers/plans/2026-10-18-unix-startup-assertion.md.
Keep the assertion, synchronize on actual HTTP serving, no production changes.
Full native CI/stress/independent repair review and main verification required.
Original lease stays active; do not start competing feature work. Final repair
SHA/tree/PR/CI evidence will be appended to PR18 and the repair PR.

## Resource onboarding (active)

Heartbeat2026-10-18T15:28Z; owner `codex/resource-onboarding`, lease until16:00Z
or merged+green main/explicit handoff. Core PR17 accepted main92b2d5f5e1f00c1ae32e219fa1cf9bd45deceee5,
tree12e206a694ec648a91738cdbbce0cde69bb2ba72, mainCI39137171776 GREEN161/423,
real browser/Nginx/Unix/static gates passed; its lease released. Agent95a5f7f/
CI39118687475 unchanged GREEN. No open PRs at acquisition. Design and plan:
docs/superpowers/specs/2026-10-18-resource-onboarding-design.md and
docs/superpowers/plans/2026-10-18-resource-onboarding.md. Implement real structured
Resource/Endpoint and explicit-context Binding creation plus live React inventory.
Preserve current authority/CAS/audit and facts. Existing object mutation, enrollment,
approval/identity, truthful node status and combined Linux acceptance remain.
Local baseline full Go fails only TestAuthorizationRuntimeTCPUnixAndShutdown/unix
with socket EPERM; native CI must execute without skip. Do not compete with this lease.

[PR18](https://github.com/PicketX/PicketX/pull/18) code acceptance:
local1d272a14d495d970924cbb1c8a2e1d04df718046,
remote f040e70aea47a3a0205c726aa3b30ec981c46fa7 and actual CI checkout
52f11cd70064a24b4c022af3f78dad1b29fe7cf9 share
tree36a5b1161e30dbb32912e1b5fd8abd3407697c3d. CI39140082943
Go117712526496/Web117712526381 GREEN. Raw167 top-level/429 total, zero
fail/skip/race; actual Unix socket, full vet/build, pinned protocol/OpenAPI
reproduction/local validation and fully static SQLite/managed Controller/CLI
checks passed. Local full ordinary/race-count1 ran and only Unix EPERM failed;
this is not a local full-suite success and no required scenario was skipped.
Locked npm ci/typecheck/lint/build passed locally and natively.

Actual Chromium Resource/dual-stack Endpoint→exact host Binding→Grant/revoke,
412/preserved draft, failed reads, Viewer denial,375px create/read layouts,
unchanged facts and persisted audit passed, along with all existing real
Nginx/application source isolation/union/deny/revoke/deadline/IPv6 cases.
Artifact12140969029 downloaded; SHA256
1637544b82b004e55fecb0d52c3c3ba9484a6b3c230f6d2f9b23a83498676f7f
matches API. Desktop, Resource/Binding mobile forms and Viewer mobile inspected.
Missing Resources navigation was observed RED in CI39139004218/Web117709345450
before UI implementation. API route404 and Grant item/optional-address schema
tests also observed RED→GREEN. Test-only integration corrections: exact reason
label selection (CI39139499263); concurrent resource test must change semantic
content, not send an idempotent publication (CI39139761084).

One fresh independent whole-branch review of92b2d5f..1d272a1 found no Critical
or Important issue. Reviewer independently passed TestInventory and related
configuration/controller/management/publication/networkplan race suites,
diff check, and inspected Binding mobile. Full browser/native/remote CI evidence
was verified by the executor, not independently rerun by the reviewer.
Deferred Minor: form/DTO character count versus domain128-UTF8-byte name limit
can reject43 CJK characters with generic feedback. Backend safely rejects;
follow-up should add byte-specific inline validation/help and boundary regression.
PR17's deferred Grant endpoint_names schema constraints are fixed here.

Rulings and their costs:
- Serialize Resource/Binding drafts: prevents hidden CAS-base advance; costs extra
  form navigation for batch work.
- Reuse existing console tokens/components after no verified matching UI pattern:
  verified error-summary guidance applied; cost is less tailored layout polish.
- Defer editing/removal: accurate create/read scope and advanced editor remain;
  lifecycle changes still need advanced configuration.
- Defer Agent enrollment and structured HTTPRoutes: current trusted configuration
  remains available; initial deployment still needs advanced configuration.
- Defer human identity/request/approval: administrator operation only; no requester
  self-service yet.
- Do not claim live application/combined privileged Linux completion: no Agent
  change and desired state explicitly labeled; product acceptance remains pending.
- Do not claim PostgreSQL/multi-Controller readiness: single-Controller SQLite
  remains the supported persistence scope; deployment choices remain limited.

Only final acceptance documentation changed after review. Final accurate-head CI,
expected-head merge and main CI must still be verified; their immutable SHA/tree/
run results are appended to PR18 after execution. Lease releases automatically
only after merged+green main, not at an earlier code-only acceptance. Next:
structured HTTPRoute and fresh desired/applied status management, then identity/
approval/lifecycle gaps and combined browser→API→L4/L7→application Linux acceptance.
Do not pause the overall task at this partial milestone.

## Structured Grant increment (active)

Heartbeat2026-10-18T13:42Z; owner `codex/grant-console`, lease until15:00Z or
merged+green main/explicit handoff. Core PR16 is accepted: main113461ba99bf070139ba7b8858d7eba996c5ce7f,
tree0a864577ad7819c00125a75ca021c4eeabd8699b, main CI39131202785 Go/Web green.
Its old lease is released. Agent main95a5f7f/CI39118687475 unchanged. No open PR
in either repository at acquisition. [PR17](https://github.com/PicketX/PicketX/pull/17)
is the active increment. Heartbeat2026-10-18T14:26Z; lease remains until15:00Z or
merged+green main/explicit handoff. Plan/context map/design:
docs/superpowers/plans/2026-10-18-grant-console.md. Implement structured source
Grant issue/revoke and real React management/ingress acceptance through existing
database authority/CAS/RBAC/transaction audit. No competing implementation while
this lease is live. Remaining: resource/binding/approval/status workflows,
verified human identity and final combined Linux/browser acceptance. Do not pause
the automation at this partial milestone.

Heartbeat2026-10-18T14:34Z; lease until15:00Z or merged+green main/explicit handoff.
Fixed code acceptance: local2d321d5f991d83290da570dfa167f759f534b573,
remote72d970d51d745655252897272963b2f2561cc796 and fetched actual CI checkout
2d5eadd4d62e42fcb251b76d05f09ce30256e118 share tree1d5cf8e66823156b985d17c5ca6a5ecaecd5ece6.
CI39136155132 Go117700970764/Web117700970967 both GREEN. Raw161 top-level/423
total tests, zero fail/skip/race; actual Unix socket, complete vet/build/pinned
generation/local OpenAPI and fully-static SQLite Controller/CLI checks passed.
Actual structured browser→SCS/CAS→Nginx/application independent union/revoke,
absolute deadline, audit, concurrent412/draft, failed read, IPv6/viewer-mobile and
all prior HTTP scenarios passed. Artifact12139816302 independently downloaded;
SHA25651458611104a13a53b7c161a102580645ef90e2142831f90bd442b5bd6708350 matches
API; desktop/mobile visually inspected. Independent review's one Important
serialization-expansion blocker fixed in one RED→GREEN pass: large escaping/dense
source plus real SQLite active revoke/reopen, source-local semantic-checked edits,
unchanged byte/object limits. No Critical or unresolved Important. Minor deferred:
generated endpoint-name uniqueness/item-length schema; runtime remains strict.
Local full suites still honestly fail Unix EPERM only; native CI executes it.

This final metadata-only commit still needs complete exact-head CI before
expected-head merge/main verification. Exact final head/tree/CI and main results
are appended to PR17 after checking. Once merged with green main, this lease
automatically releases: do not wait15:00Z or redo Grant management. Next:
structured Resource/Endpoint/Binding editing and truthful Agent/decision/audit
pages, basic request/approval workflow and verified identity, then final combined
browser/HTTP/Agent/Linux acceptance. Keep database authority and strict CAS.

## Trusted HTTP increment (active)

Heartbeat2026-10-18T13:15Z; owner `codex/http-authorization`, [Core PR16](https://github.com/PicketX/PicketX/pull/16), lease until14:15Z or merged+green main/explicit handoff. Core PR15 baseline74575a4/CI39127783270 and Agent main95a5f7f/CI39118687475 remain accepted. Plan/spec/context map: docs/superpowers/plans/2026-10-18-http-authorization.md. Source-backed HTTPRoute, scoped local TCP/Unix decisions, durable minimal audit and actual Nginx/browser/application now implemented, code acceptance below. Final metadata head still needs complete CI/expected-head merge/main verification. Do not compete with this live lease. Overall goal remains incomplete.

Code acceptance: local8cb7ece287d358fad053009ca50c1ad95289a7de, remote fe43e0772de1ec54b40cfa10fb0e9cf0188d1d08, actual CI checkout55f510aad6db8f03a297aa0d1a6d1429d270fd63 all share treea2932abdfd22ac650d9a7abc562bee4706569779 (fetched exact source; no diff). CI39130425080 Go117684282808/Web117684283026 both green. Raw Go150 top-level/405 total, zero fail/skip/race; real IPv4/IPv6 TCP and Unix valid scoped decisions, socket0600/shutdown/cleanup, schema upgrade/reopen/audit fault, complete race/vet/build/pinned generation/local OpenAPI and actual static SQLite Controller/CLI checks pass. Raw Web locked npm checks and actual Chromium→console→SCS/CAS publication→HTTP authorization→Nginx1.24.0→real HTTP application pass; reused connection target/Host/method/path/source spoof isolation, explicit deny, source revoke/absolute expiry, IPv6 context, cookie audit and caller revocation/unavailable are covered. Artifact12137651861 independently downloaded; SHA256168e64403b3081383f3f9f94ece5dbde4ad6dfe15ae202ef38ec847650ac4abc matches API; application/mobile screenshots visually inspected.

One fresh whole-branch reviewer found no Critical/Minor and one Important stale-clock decision across queue/durable-audit waits. Four real expiry/deny-activation × mutex/SQLite-connection cases RED→GREEN; now refresh UTC after lock and re-evaluate/re-audit if a boundary crossed before return, bounded2 then503. Complete fixed-source CI above is GREEN. No unresolved or deferred finding. Local full suite still honestly fails Unix EPERM; real Linux CI supplies required creation/execution/cleanup (no skips). Detailed review scope/rulings/limits in plan. Agent/kernel source unchanged, no new combined Linux or production HTTPS acceptance claimed.

After merged+green main (final results appended to PR16), this lease releases automatically; do not wait14:15Z or redo the HTTP adapter. Next: structured Resource/Endpoint/Binding/Grant create/revoke/status and approval APIs plus real React pages using ui-ux-pro-max/frontend-design, then human/OIDC and final combined browser→approval→Agent/native network and HTTP application workflow. Keep one database authority and current strict CAS/authentication; policy file polling must not overwrite managed data. Exact-path/source-only HTTP alpha is not generic verified-user/prefix/query-sensitive authorization or production capacity. Do not pause autonomous delivery before the overall goal is actually validated.

## Browser console handoff (current)

Heartbeat 2026-10-18T12:27:00Z. Owner `codex/browser-console`, [PR15](https://github.com/PicketX/PicketX/pull/15); lease until2026-10-18T13:30:00Z, auto-release only after merged/green main or explicit handoff. Prior PR14 lease released. Accepted Core main0ccfa8ff8522fb95d93b7898f8b0a1ed9dfcc65a/tree3f49d6f51cc0cba29ada9f3a65a7d845a849e06d; main CI39121855312 green128/357, zero fail/skip/race; Go/Web/static SQLite-managed-runtime checks passed. Agent remains95a5f7f/CI39118687475. Neither repository had an open PR at acquisition.

Plan/spec/ledger: `docs/superpowers/plans/2026-10-18-browser-console.md`. Scope: actual SCS operator session/CSRF and live publication/audit React console. Remaining: structured management/approval, L7 and final browser→proxy→application acceptance. Do not pause this task after this increment.

Browser implementation evidence (not yet merge acceptance): remote head65096587c3769b3037e1b78486370955d08304f8/treec11e009074d7856a0a37599ff74f3ef9cfc5241e matches local376742e and fetched CI checkout9754a7594697c0f5097c9b36b821614ba9fa23d8. CI39125526147 Go/Web passed. Raw Go job117669729855 confirms133 top-level/362 total tests, zero fail/skip/race, vet/build, pinned generation, local OpenAPI and static Controller/CLI gates. Raw Web job117669729678 confirms actual Chromium→React→SCS/CSRF→Go→SQLite publish/audit, external-writer conflict/draft, injected read failure, viewer backend denial/mobile and credential revocation. Screenshot artifact12135557659 downloaded, SHA256c658e43251ca395e77f20f698cf684ea71aa92cd36af40a1c4145fe5ab3002d7 verified; desktop/mobile visually inspected. Local browser CDN zip was empty; no local browser success is claimed. Initial CI39124774237 was intentional missing-login RED, not acceptance.

Pre-review supplement adds asset confinement, explicit expiry/fixation/absolute deadline tests and RED→GREEN fixes for duplicate Origin and unsupported Bearer challenge on cookie routes. Complete local race136/365 passed. Final exact-head CI, fresh whole-branch review and expected-head merge/main verification remain mandatory.

Post-review code acceptance: remote headecddce298d5b18933680808593191e469fc68d22/treeab3b87c3f0cb440189257d2adab69d60a7e0dcfc equals local7b27b24 and actual CI checkout4fa673c6de7e51ed4582514a13fad8c9ec55aef6. CI39127108548 Go/Web both green; raw Go job117674442703 confirms139 top-level/368 total tests, zero fail/skip/race, vet/build, pinned protocol/OpenAPI/local validation, actual fully-static SQLite/gRPC/managed API/audit/restart/SIGTERM and CLI deadline/withdrawal gates. Raw Web job117674442819 confirms locked build plus actual Chromium flow, including enabled/readOnly focus/selection/native keyboard scrolling/no-edit and viewer backend403. Artifact12135394967 SHA2560bcded1a0a9fd54a2072f83d2153660812cc14eee1ddcc2bee7fb8ae029097fd downloaded/verified and corrected mobile screenshot visually inspected.

Independent fresh whole-branch review0ccfa8f..d934b64 found no Critical and two actionable findings: unknown-cookie stripe-lock availability and viewer keyboard access (regraded Important). Single fix pass: both stalled unknown/existing-other-session regressions RED→GREEN; exact valid-session locks, cancellation/ref cleanup and logout-resurrection race passed. Real viewer CI39126597617 was RED on disabled focus; fix run39126806452 exposed immediate scroll-state timing, corrected to bounded observation of native scrolling without JS scroll assignment. CI39127108548 is the complete GREEN acceptance. No unresolved findings/deferred minors; reviewer independently ran full ordinary/focused race, not browser/remote/static/Web. Detailed root rulings/limits are in the plan ledger.

This final metadata-only commit still requires full exact-head PR CI before expected-head merge and main verification. Exact final head/tree/CI and merged-main acceptance are appended durably to PR15 after checking, as with PR14. Once PR15 is merged with green main, its lease is released automatically: do not wait until13:30Z or reimplement sessions/publication pages. Next required slice: basic trusted HTTP request authorization plus scoped local decision interface/real ingress, followed by structured resource/Grant issue-revoke/status and approval pages. Complete browser→authorization→proxy→application and Linux evidence remain required before pausing delivery. Agent code is unchanged here; PR16 remains its native baseline, not a new kernel claim.

The historical management section below is superseded by the accepted PR14/main above.

Updated: 2026-10-18T11:00:00Z. User explicitly authorized autonomous development, tests, push, PRs and merges until a usable management application and basic Layer 7 authorization exist. Routine design/plan and integration decisions do not require another approval round. Preserve existing security boundaries and required checks.

## Current lease
Owner: managed-publication increment. Branch: `codex/management-authority`. Lease expires: 2026-10-18T12:30:00Z; automatically releases after its PR merges and main CI passes. Check the latest PR/head/heartbeat before competing work. Provider PR16 is merged with green main CI and its prior lease is released.

Current accepted Core main0b2c5e919626f281750295cf6cb119ea7241ed99/tree094555cc0a493cddbe855b4dac9fd38715da54de, CI39117067190 green (115/344). Agent main95a5f7f62dceb1c369d08b8e23ffda2f7b978ce4/treea53a743e22254c0511b430e5b9e2ed0664180629, CI39118687475 green (190/482,112 native/9 service cases). PR11 nonempty-target and PR12 stale UPDATE regressions are complete; do not repeat. Both repositories had no open PR at lease acquisition.

Plan: docs/superpowers/plans/2026-10-18-management-publication.md. Scope is an actual independent-Token/Casbin management API, explicit database authority, CAS and transactional audit. Human login, L7 and live pages remain next. Baseline `go test ./...` passed. Native inline execution; one fresh whole-branch reviewer at integration.

## Management increment ledger

Plan identity: docs/superpowers/plans/2026-10-18-management-publication.md.
Pre-flight: Task1 Store interfaces are Task3 runtime inputs; Task2 Access interfaces are Task3 identity inputs. No conflicting contracts.
Task1 complete: commit1482a3d, missing managed API RED→GREEN; complete `go test -count=1 ./...` passed. Source binding/v1 migration, stale semantic no-op, two concurrent writers, raw input roundtrip/reopen and transactional audit rollback covered.
Ruling: use this committed inline ledger because cloud skill helper scripts are not exposed as executable filesystem resources; cost is manual bookkeeping, preserved in the repository.
Task2 active: independent private principal file and official Casbin adapter/enforcer; focused tests initially failed on missing APIs. Dependencies verified against actual pinned module source; selected official PostgreSQL transitive versions are pinned but not runtime-supported here.
Task2 complete: private credentials/revocation, combined deny and reload failure/recovery tests passed; `go test -count=1 ./...` passed after dependencies/Go sums resolved. Upstream-retracted mimetype1.4.15 was replaced with1.4.16. The adapter imports multiple drivers, but only the existing official SQLite connection is used. Native Casbin AutoLoad discards errors; a cancellable periodic owner calls LoadPolicy and records fail-closed availability instead, with no watcher or automatic interval changes.
Task3 active: strict HTTP tests GREEN; actual TCP API→SQLite→gRPC→restart test initially failed with no managed initial state, then exposed duplicate initial publication (service rejects non-increasing revisions). Initialization now only commits; common assembly publishes exactly once. Integration and full suite GREEN. Complete race/static/Web/docs and independent review pending.
Task3 evidence: complete ordinary Go tests/vet/build, static file/managed Controller artifact and static CLI smoke passed. Web npm ci/typecheck/lint/build passed. Static migration smoke first caught its old schema1 assertion, updated to schema2 and source binding. First full race passed; final new-case race pending. Post-commit distribution failure/recovery and invalid credential readiness covered.
Task3 complete: local commit2557b398c0c0787826ad942057d8061eb6a53ff5, exact remote code head525497cf051d8f4937df020b53f5e20b52e311c9/tree2576f0fd53aff92f59babf16c6a81296f4473b04 verified by fetch/diff. Final full race127 top-level/356 total tests, zero fail/skip/race. Full independent branch review (including bilingual README/roadmap status corrections) found no Critical/Important/Minor issues or declined judgments; independently ran complete ordinary and focused race suites. Reviewer did not rerun static/Web/native or inspect remote CI. PR14 code CI39120860780 pending; merge remains gated on exact-head raw evidence.
CI39120860780 failed before required tests: setup-node used nonexistent apps/web/.node-version (raw job117656009194). Regression TestDocumentationCINodeVersionFilesExist failed on that exact path; fixed to repository-root .node-version. This is a workflow/test-only correction beyond the reviewed production tree; no production behavior changed. Failed/skipped CI is not acceptance; full checks must run again.
Code acceptance after correction: remote headf8536e6b447b9da5454dadbb9f235b492dd27e39, tree27daea21709eb7682f8d7f0dc9b551e72a4abbb8, exact fetched CI checkout96108c071c0c009b4b95b54c9899307df07c376b has the same tree. CI39121161170 passed both required Go/Web jobs. Raw Go job117656881965 confirms128 top-level/357 total pass, zero fail/skip/race, vet/build, pinned protocol/OpenAPI regeneration, local-only OpenAPI3.1 schema/reference validation, actual fully-static file SQLite/gRPC and managed Token/Casbin/HTTP CAS/audit/restart/SIGTERM smoke, plus static CLI deadline/withdrawal checks. Raw Web job117656882175 confirms npm ci/typecheck/lint/build. No Agent kernel source changed; prior PR16 native evidence remains the Agent baseline, not a new native claim.
Final metadata/docs-only head must still pass complete CI before expected-head merge/main verification. Integration outcome and exact final/main CI are appended to PR14 after verification; the lease auto-releases only on merged/green-main. Independent review is recorded above; no unaddressed findings or deferred minors.
Ruling: Redocly CLI validation triggered an unapproved external telemetry request and was stopped; replaced with locked Swagger Parser13.1.0/Ajv2020 local-document validation, with all external/file/HTTP resolution disabled. Cost: different lint recommendations, while full OpenAPI3.1 schema/reference validation and project contract checks remain required. No telemetry permission was assumed.
Remaining executable slices: SCS browser login/CSRF and trusted membership freshness; usable object/Grant/approval APIs and real React pages; basic trusted HTTP ingress authorization and scoped TCP/Unix decisions; actual browser→API→authorization→proxy→application plus Linux acceptance. Do not pause the task until those pass. Swagger UI rendering and per-resource delegated management are not supplied by this API increment.

## Historical baseline (superseded by accepted heads above)
- Core PR11/main793ac15faadd6565c64c713959c59990c2bf3a1b, tree f6f98df86d7e0e69c0ba7ac41cf563c4d865572a. Main CI39109301425 passed94 top-level/311 total cases, real gRPC/TCP/TLS. Embeddable stream service only.
- Agent PR15/main1949d6352da6afb0bf6796d97e45eabe6d197ee8. Static host L4, independent deadlines, scoped strict cleanup, durable restart and systemd readiness accepted; Controller provider pending.

## Historical work sequence (steps1/2 and both deferred regressions are complete)
1. Running Controller: strict Koanf configuration, SQLite durable publication, credential reload, optional standard server TLS, bounded HTTP/gRPC lifecycle and static Linux artifact smoke. Implemented in PR12 and locally validated; independent review/PR CI/main CI determine acceptance. After PR12 is merged with green main CI, proceed to step2 without repeating this implementation.
2. PicketXD Controller Provider: exactly one authoritative provider, scoped validation, independent deadlines during outage, provider-bound remote LKG and actual native apply reports. Reuse the typed protocol without importing GPL Agent code into Core.
3. Management APIs/persistence/login/RBAC and basic HTTP request authorization through a trusted existing ingress plus scoped local decisions. Keep approval, desired state and applied assertions distinct.
4. Real React administration pages (resources/bindings, Agent freshness, grant issue/revoke, authorization rules and records) against live APIs; basic portal/approval workflows as documented. Use the selected shadcn/Tailwind/i18next stack, UX/UI skills and explicit loading/error/empty/stale states.
5. Actual browser/API/proxy/application walkthrough and Linux enforcement evidence. At completion pause the hourly task and report precise acceptance evidence.

PR11's deferred nonempty target after credential registration is complete in PR12. Every increment needs uncached relevant full tests, fresh whole-branch review, checked remote source tree/head, passing required PR CI, expected-head merge and main CI verification. No forced pushes or bypassed checks; no fixture/UI-only completion claims. Actual implementation is required when an execution environment is available; otherwise report a real blocker.

PR12's deferred stale UPDATE CAS regression is complete in PR13; do not repeat it.

## Historical Provider handoff (PR16 now merged/green; lease released)
Core PR12 is accepted: main5e2254efec90f2a5448450a2974c587c37f1dd18,
tree d78219ef42fb515e73c0144e40e7ea85a3896cc8, main CI39113728426 success.
The PR11 newly credentialed nonempty-target regression was completed in PR12.

Agent PR16 (`codex/controller-provider`) is now merged with green main CI; the
Provider increment lease has released, as recorded above and in its PR evidence.
Historical Core PR13 only supplemented the deferred stale UPDATE CAS regression: seed revision1,
open two stores at revision1, persist a valid revision2 and reject the stale distinct update.
The new test caught deliberate removal of the revision predicate; production code is unchanged.
Local complete race115 top-level/344 total tests, vet/build and Web lockfile checks passed.
