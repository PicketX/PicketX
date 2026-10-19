# Managed accounts: independent review and code acceptance

## Scope and immutable reviewed code

One fresh read-only whole-branch reviewer (gpt-6-astra/high, no history fork,
no delegated reviewers) inspected local6c5d28d5750ba2991382c9b97b8cd675ade1702b..
6c3b1496232ba552961b06013e7de3523793a7e2. This is an actual independent agent
review, not a fabricated GitHub approval. The executor verified the findings.

Reviewed local6c3b149, remoteab36da1d25a0edc2cd06abc2a5dda2c5a2af23d7 and actual
CI checkout5f604cff097ebdc6a8297745466538b7ea95056a share tree
3a14b5182881bab96be5b3ce5ed4f0bc478a6e67. Later acceptance-document changes do
not alter implementation; final exact head/tree/merge/main evidence belongs to
[PR33](https://github.com/PicketX/PicketX/pull/33) final metadata, avoiding a
self-referential CI/checkpoint commit loop. No full-product completion claim.

## Native and local verification

[CI39223445538](https://github.com/PicketX/PicketX/actions/runs/39223445538):
Go117963923063 and Web117963923350 SUCCESS. Raw Go241 top-level/591 total tests,
zero fail/skip/race; Unix socket100 repeats; pinned protobuf and OpenAPI reproduction;
vet/build; fully static SQLite Controller/CLI and real persisted managed publication,
gRPC, restart and SIGTERM checks pass. Locked npm ci/typecheck/lint/build and full
Chromium console suite pass (31 explicit PASS log lines, not31 kernel scenarios).

Real Accounts browser evidence covers create/audit/CAS, disable/re-enable old-cookie
401, reset and old-password rejection, requester403, preserved stale drafts, failed
reads,202/500/transport unknown receipts, cleared password state, held writer plus
older real GET across commit, custom directory-only rights, English/Chinese375/1440.
Existing real request/independent approval and Nginx/application cases also pass.
This does not assert new Agent/kernel confirmation: no Agent implementation changed.

Downloaded screenshot artifact12173041437 actual ZIP SHA256
ecd8a6f577a3961d12b1b4a825786aa59b7d3f334b880fdc0b18553c9fba350a matches GitHub
metadata. Four Accounts screenshots were visually inspected: field/error/acknowledgment
hierarchy, immutable identity, clear state, no horizontal overflow or visible secrets.
UI/UX skills retained the established React/shadcn/Tailwind palette and compositions;
focused error summaries, linked inline errors and explicit session invalidation were
used instead of inventing a new dashboard theme.

Local full normal/race runs encounter only host-restricted
TestAuthorizationRuntimeTCPUnixAndShutdown/unix socket EPERM. This is recorded,
not skipped or called a local pass; unrestricted exact-tree CI supplies the complete
green gate. Focused normal/race, vet/build and locked Web gates pass. The reviewer
independently ran uncached focused managementaccess/controller/runtimeconfig tests;
management has no standalone tests. Semantic OpenAPI comparison found no changes to
existing paths and inspected the four new account paths and authentication contracts.

## Observed RED → GREEN before independent review

| Failure / proof | Resolution |
| --- | --- |
| Missing account domain; HTTP404 and absent UsersMode | Implement framework-free directory, strict API and explicit typed assembly |
| Startup accepted accountVersion2 at directoryRevision1 | Validate generation against durable authority at startup as well as request reads |
| Native39221523972/Web117957690865: actual key login, missing Accounts control | Implement real console only after observing the native failure |
| Native39222401445/Web117960552920: browser encoded principal PUT400; screenshot ZIP hash34ab469e3af3b248630eea1156e1b4fa58e4a5ae5d8507c22c9dbe46cfe5e9b5 | Local encoded-path test RED/GREEN; decode once then retain exact domain ID validation |
| Native39222963850/Web117962368893: older real GET5 replaced committed account6 | Read-epoch ownership prevents obsolete successful reads from updating directory/CAS/freshness; same native regression now passes |

Existing publication stale UPDATE regression already exists as
TestStaleUpdateCannotReplacePersistedRevision; it was not duplicated.

## Reviewer verdict and executor rulings

Reviewer: Ready to merge **Yes**; Critical0, Important0, Minor1. No blocking
authority, session-revocation, authorization or transactional-integrity issue found.
No re-review was requested and no blocker fix pass was necessary after this review.

Minor1 — Accounts.tsx read rejection bypasses successful-response epoch check.
An older GET503 after newer reconciliation success can set readFailed/error.
The reviewer reproduced this ordering in an in-memory function harness, not as
native browser proof. The directory remains at the committed revision; the effect
is a conservative, obsolete warning, not false freshness or lost state.

Final: Ruling: retain severity Minor and defer to the next related UI change —
no data/CAS/authentication bypass; cost: an unnecessary stale warning until refresh.
Follow-up must guard rejected reads with the same ownership epoch and extend the
held-read native regression with an older failing response. Do not silently drop it.

The reviewer's complete Declined to judge list is resolved explicitly:

- Final: Ruling: OIDC/registration/MFA/mail recovery/multi-replica sessions remain
  excluded — local-account/SCS scope is explicit; cost: no federation or HA session
  service, and restart still requires login. This is not an OIDC-complete claim.
- Final: Ruling: new Agent/kernel behavior is outside this Core-only change — no
  enforcement code changed and Accounts says no network Grant/application claim;
  cost: existing Linux conjunction must be extended to the DB-user flow before
  claiming that new full chain, rather than treating these screenshots as kernel proof.
- Final: Ruling: PostgreSQL runtime compatibility is not established — actual
  accepted runtime/integration baseline is SQLite; cost: selected PostgreSQL support
  needs separate real integration before being advertised, not a silent driver swap.
- Final: Ruling: account delete/rename/recycling/arbitrary directory groups stay
  excluded — immutable identity, disable and four built-in roles preserve provenance;
  cost:128 total account ceiling and no username recycling/group directory editor.

## Integration and next scope

User standing authorization selects safe push/PR/expected-head merge into main
after exact native gates; no routine integration menu is required. Preserve the
host-owned isolated worktree/branch. Final merge/main CI and automatic lease release
are recorded in PR33 metadata. Extend the existing Linux human-approval conjunction
to database-account provisioning, then actual Gitea HTTPS/SSH and deployment/release
acceptance as required. Do not repeat PR19's accepted file-human/kernel work.

中文：本段实现真实数据库账户、独立 CAS/审计和中英文页面；旧会话不会因重新启用而恢复。
完整原生 Go/Web 通过，241顶层/591总测试，无失败/跳过/race，截图哈希及源码树已核对。
独立全分支评审无阻塞；保留一项 Minor：旧读取失败可能多显示过时警告，不覆盖数据或绕过权限。
其余 OIDC、PostgreSQL运行兼容、HA、账户删除/改名/复用均未宣称完成。新数据库人员流程的真实
Linux联合验收留作下一步；最终准确合并/mainCI及租约释放以 PR33 最终元数据为准。
