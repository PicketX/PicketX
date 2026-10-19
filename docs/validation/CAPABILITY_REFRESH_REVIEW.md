# Capability readiness: review and evidence

2026-10-19, PR37, owner codex/capability-refresh. Test-only repair: production
authorization, UI, API, schema and Linux enforcement unchanged.

## Root cause and tests first

Pinned Playwright1.64.1 decides predicate truthiness before awaiting async output:
Promise(false) terminates the old wait while normal30s Casbin reload is pending.
Native39305662932/web118241942266 at18:51:58Z deterministically observed creator
/me200/empty operations BEFORE real SQLite permission publication; pinned async
wait resolved false and old extracted one-read wait failed false!==true.
Local8f69f15/remote1e255c6/tree635aae93f6df0a812302053a431d45d0c355bcc9.
Failure archive12212654396 SHA256
`d4119e1526e55cf7db806e6061f07656325c52cbec80e7b455ed4a175deb8ec9`.
Earlier local200reload/12000checks, mixed-role race10 and diagnostic native passes
are characterization, NOT RED or repair proof. Fix explicitly awaits real HTTP;
only valid200/missing-operation polls. Retain normal policy reload,12 strict real
browser reload assertions and later request/review/revoke; no pass-after-failure.

## One independent whole-branch review

Reviewer capability_refresh_review reviewed85fa14e..1ee7d43/tree7521e9ae, all8files,
spec/plan/ledger/surrounding code; independently ran9/9 actual-HTTP tests. Verdict:
with fixes, no Critical, one Important, one Minor. No GitHub approval impersonation
or second review. Root handles the ONE correction pass.

Important: response.json SyntaxError can expose malformed body bytes. Root verified
two actual-HTTP RED regressions (200 and503 malformed bodies), then used ONE
self-contained reader in both real Playwright evaluations and HTTP tests. Non200
returns before parsing; parse errors become fixed body-free messages without
cause; projected fields type checked before assertions. Combined28 tests PASS,
zero fail/skip, including11 readiness tests. Native exact final-source/main gates
remain mandatory; this document does not infer them from local passes.

Minor deferred: browser callback cannot serialize Node cancellation signal; each
GET has15s timeout, outer readiness45s. Already-issued read-only GET may finish
after caller timeout. Node adapter honors signal; shared browser cancellation is
NOT proven. Cost: bounded in-flight diagnostic work after caller rejection.

Declined judgments: reviewer did not verify native CI/artifacts/merge/main;
12fixed-policy reloads are not atomicity during concurrent policy changes;
pre-existing console cleanup can replace browser exceptions (unchanged/deferred);
temporary response collector intentionally removed after deterministic RED.

## Rulings and costs

- Autonomous mandate waives routine design/plan/merge pauses; retain artifacts,
  inline TDD, one review. Cost: executor makes scoped judgments without interims.
- Reuse clean linked worktree/new exact-main branch; preserve old branches/user
  state. Cost: retained workspaces, no destructive cleanup.
- Native Chromium/nginx needed after local browser archive installation failed.
  Cost: native execution time; never simulate required browser evidence.
- Assert no NEW read starts after deadline, not packet withdrawal. Cost: a request
  issued before abort may arrive later, no fictitious cancellation guarantee.
- Task2 root-cause extension split during Task1; actual basee2c1816 retained rather
  than later task-start head. Cost: paperwork follows emergency diagnosis.
- UI skills preserve existing design/layout/status; no unrelated transition or
  cosmetic changes. Cost: no new production feature in this repair increment.
- Defer cancellation Minor and old cleanup concern. Cost: separate follow-up,
  accurately bounded diagnostics rather than unsupported guarantees.
- Remove temporary upstream characterization/collector after RED. Cost: later
  investigations re-instrument; permanent deterministic regressions remain.

Local full normal/race have ONLY inherited Unix listener EPERM. Vet/build and
locked npm ci/typecheck/lint/build pass. Require native full normal/race/vet/build,
Unix100/static Controller/CLI, locked Web/actual Chromium/Nginx, detached install,
actual unprivileged systemd crash restart and stopped complete restore. Exact final
head/tree/jobs/four archive digests/merge/main and immediate lease release belong
in PR37 final metadata, avoiding recursive documentation-only CI.

Next Agent service/provider-bound LKG and packaged ingress/application operational
acceptance. Controller service is not Agent service, host reboot or whole-goal
completion. Keep the current task enabled.

## 中文接续

本轮只修测试等待，确定性原生RED已保留，真实SQLite周期刷新与12次严格浏览器
断言不变。一次独立全分支评审的异常JSON泄露Important经真实HTTP两项RED→GREEN
修复，合计28测试通过；浏览器共享取消信号Minor及既有清理异常问题明确延后。
本地Go/race仅既有Unix权限限制，最终精确源码/main完整原生CI及四归档仍必须
通过。准确证据和立即释放租约写入PR37最终元数据。Agent服务/LKG和安装后联合
验收未完成，不宣称整体完成。
