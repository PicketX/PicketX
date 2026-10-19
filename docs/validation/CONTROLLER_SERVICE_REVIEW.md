# Packaged Controller service / stopped restore review record

## One independent whole-branch review
Fresh read-only controller_service_review (gpt-6-astra/high) reviewed full
93024b229c072ce1db8512323c3d7fa35e3df74e..e54e8cd9d01324b7071954bd5ca49488a4ebd4d7,
all eleven files, AGENTS/spec/plan/ledger and relevant production contracts.
Independently ran12/12 safe Node tests, both module syntax checks and diff check;
did not mutate checkout/index/HEAD/branches/system manager or spawn another reviewer.
No fabricated GitHub approval. Local PID1 supervisord, native review evidence absent;
executor must establish full actual CI/raw/download/source/main gates separately.

## Important findings / one root fix pass
1. WorkingDirectory does not unquote its directive value. Reviewer and root actual
   unprivileged systemd255 parser reproduced fatal quoted-space path rejection.
   Extract existing renderer without behavioral change; regression RED against old
   serialization, then GREEN with directive-specific unquoted escaped path. ExecStart
   and ReadWritePaths still use word-list quoting. No shipped unit setting changed.
2. Accounts() sets observedAt=time.Now().UTC() on each read (accounts.go268); exact
   full response comparison cannot establish stopped restore. Two actual value
   regressions RED: fresh metadata differs, invalid metadata accepted. Exclude ONLY
   top-level accounts.observedAt after RFC3339 validity and actual request-time
   freshness checks. Mode/revision/full accounts/events/event timestamps remain
   exact; mutation assertions prove durable differences are not normalized away.

Three new regressions RED→GREEN, then complete applicable Node15/15 no fail/skip.
Actual final native verification still required; no second reviewer/re-review.

## CI-confirmed blocker regrade in the same open fix pass
The sole review's potential Type=simple /proc race was initially a deferred Minor.
Corrected native39298124906/install118215880462 then actually failed17:56:55Z on
manual restart: observed UID0 vs expected999 before systemd dropped identity/exec.
This run is NOT acceptance. Regrade to Important by actual effect; the first fix
pass was not complete because native remained red. Two real controlled child-process
tests RED with immediate inspection: delayed exec rejected and persistent mismatch
did not receive bounded retries. GREEN after bounded actual identity observation;
all UID/capability/executable/manager-PID checks remain mandatory, and actual HTTP
readiness must confirm the SAME inspected current PID. Full applicable Node17/17.
No second review; no deferred Minor remains in this increment.

## Rulings I made (ordered, costs)
1. Autonomous mandate waives routine artifact/execution/push/merge pauses; retain
   focused self-reviewed design/plan/inline TDD/one review. Cost: executor makes
   scoped judgments without intermediate human review.
2. Fresh exact-main clones + ignored linked worktree after workspace maintenance.
   Cost: retained clones/branches, no user state overwritten or cleaned.
3. Real manager instead of rebooting runner or mocking systemctl. Cost: no machine
   reboot/boot-enable or Agent Controller-mode service claim.
4. Unique temporary user/unit/root-owned payload on opted-in disposable CI only.
   Cost: privileged CI needed, not an automatic production installer.
5. Complete stopped-instance copy, not online SQLite backup. Cost: downtime and
   backups must be protected as credentials; no concurrent writer guarantee.
6. Notices tests require actual bundle. Initial scripts/*.test.mjs wildcard failed
   that explicit precondition; run applicable unit suite locally and real notices
   in native install job. Cost: no all-scripts local pass claim.
7. Reviewer-declined concurrent mutation/online backup remains excluded, not silently
   supported. Cost: caller must stop service and trust/stabilize parent paths.
8. Reviewer-declined actual reboot/boot-time enablement remains excluded. Cost:
   service lifecycle is not machine reboot proof.
9. Reviewer-declined Agent/Grant/kernel enforcement unchanged; IPv6 TCP check is
   listener proof only. Cost: no new network enforcement claim from Controller run.
10. Reviewer-declined migration rollback/HA/multi-writer restore excluded. Cost:
    same-binary stopped copy only; deployment migration needs separate evidence.
11. Reviewer-declined production installer/proxy deployment excluded. Cost: operator
    follows reviewed manual instructions, not a universal bootstrap mechanism.

## Verification chronology / final immutable receipt
Core baseline/full changed-tree Go normal/race only inherited
TestAuthorizationRuntimeTCPUnixAndShutdown/unix socket EPERM; full vet/build0.
Locked npm ci/typecheck/lint/build pass, no runtime/Go/UI changes. Ten real FS tests
RED against empty contracts then12 applicable tests GREEN. Initial fixture expected
SHA was independently corrected using sha256sum, not helper-derived.

First tests-first39296675649 failed workflow parsing before any jobs: unsupported
runner.temp context at job.env. NOT intended RED. Changed only browser cache context
to the already-used github.workspace pattern and ignored node_modules path.
Corrected tests-first39297129941/install118212435393 observed actual intended RED
17:49:51Z 'actual systemd and stopped restore proof required' AFTER real detached
Chromium acceptance passes. No old process-spawn run is service success.

Final exact local/remote/actual checkout/tree/CI/jobs, public archive SHA values,
static ELF/content/source checks, screenshots, expected-head merge/parents and full
main acceptance + immediate lease release belong in PR36 metadata. All gates remain
mandatory after fixes; this record does not itself assert native success.

中文：一次独立全分支评审，2项阻塞及随后CI证实并升级的时序阻塞均在同一未完成
修复阶段用真实回归RED→GREEN解决，17项适用Node通过；无新遗留Minor。取舍/代价
如上；不冒称整机重启、在线
备份、Agent执行或生产自动安装。最终准确CI/下载产物/main及立即释放以PR36为准。
