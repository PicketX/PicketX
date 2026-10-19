# Private packaged Nginx includes: independent review and verification

One fresh read-only whole-branch review, base cdee150201a5e33036cd3d73e3be5a8201000698
through local773e55ef323a9c3ff3a32d47a4f85a2fc857c1f4. Reviewer independently ran
the full CLI package, Node syntax and diff checks; no edits/subreview/re-review or
fabricated GitHub platform approval. Verdict **No**, one Important, one Minor,
no Critical. Root must fix blockers, verify exact native source/artifacts and
merge only with expected head. Source review is not real Linux acceptance.

## One root regression-first fix pass

1. Important: clean-parent validation followed by raw-path operations erased a
   symlink/.. component only from validation. Real-filesystem
   TestNginxInitRejectsSymlinkDotDotTraversal/{output,token} both RED (accepted),
   then canonical absolute-path rejection before all writes/read GREEN; full CLI
   suite green. Existing unrelated controller-init behavior not rewritten.
2. Native CI Important:39337074356/install118349132184 at22:47:01Z rejected real
   generated includes with `could not build variables_hash` (default64byte bucket).
   Complete SHA256 variable suffix caused the failure, not missing capabilities.
   Shorten namespace to128bits (first16digest bytes), keeping stock Nginx settings;
   collision causes duplicate internal-location rejection/failclosed. Real failing
   installed suite retained unchanged. Native GREEN and final exact evidence pending.

No skipped assertions, inflated Nginx hash setting, permissions/rate/auth weakening
or fake decision server. Earlier failed CI is NOT acceptance; skipped subsequent
service/archive steps cannot establish any new deployment proof.

## Deferred Minor

- Hostile-header fixture checks spoofed source but not independent forged method,
  host and URI, and its forged route equals selected route. Renderer overwrites
  each field correctly and wrong method/host/path/route requests are denied, but
  headerIsolation proof has narrower regression coverage than ideal. Deferred per
  one-pass policy, not a claim that all possible spoof combinations were tested.

## Rulings, including every behavior reviewer declined to judge

- Helper deliberately accepts only controller-init64lowercase-hex token subset,
  not every runtime-valid ASCII token: avoid unsafe Nginx complex-value escaping;
  runtime contract unchanged. Cost if wrong: other token users need reviewed
  manual ingress configuration, not automatic conversion.
- User's autonomous mandate waives routine design/plan/push/merge confirmation
  pauses; verification/review remain gates. Cost if wrong: revert a scoped PR.
- Independent installed-ingress-report.json accompanies existing install-report,
  preserving exact old install/service recovery assertions, not combining two
  different Controller fixtures. Cost if wrong: correlate two same-source reports.
- Acceptance fixture and CLI share one feature commit despite planned task split;
  consumer RED was actual prior shipped executable. Cost if wrong: coarser history.
- Production TLS, real-client-address trust and private upstream isolation remain
  existing ingress operator responsibilities. Reasonable users receive no automatic
  TLS/security claim. Cost if wrong: deployment may bypass source authorization.
- Surrounding rewrite/error_page/internal redirects/alternate paths must be reviewed
  in the entire operator configuration; explicitly documented bypasses unsupported.
  Cost if wrong: operator can expose application outside generated checks.
- Adversarial privileged filesystem races excluded by exclusive trusted-parent
  model; deterministic lexical discrepancy is included and fixed, not excluded.
  Cost if wrong: a hostile privileged operator could redirect/read credentials.
- Other runtime-valid token formats remain expressly unsupported by this helper;
  the first ruling applies, no runtime/API token restriction was introduced.
  Cost if wrong: extra manual integration work, not silently modified credentials.
- Agent/kernel/Gitea conjunction unchanged/deferred to the next installed increment.
  Cost if wrong: this test alone does not establish both network and L7 gates.
- Native execution/archive identity/post-merge receipts are external mandatory
  gates, never waived by source review. Cost if wrong: unverified artifacts would
  be mislabeled; do not merge/claim acceptance without actual proof.
- File syncing is present; power-loss durability of directory entries is not
  promised/tested. This generates disposable snippets, not authoritative database
  state. Cost if wrong: regenerate into a NEW directory after interruption.

中文：一次独立全分支评审发现1项重要路径问题、1项延后小项；原生执行另发现默认
Nginx变量名限制。根实现者保留真实RED并修复，不伪造评审批准。最终PR/main/原始日志/
四份实际归档证据和即时释放记录在PR38元数据；完整目标尚未完成。
