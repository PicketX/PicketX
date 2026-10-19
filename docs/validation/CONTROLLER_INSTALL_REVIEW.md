# Controller installation review and evidence ledger

## Scope / gate
One fresh read-only whole-branch reviewer controller_install_review (gpt-6-astra/high)
reviewed298863f..2b98d6c, full spec/plan/map/ledger and runtime consumers. Focused
CLI and bundle preflight tests independently pass. No second review or fabricated
GitHub approval. Exact final native/raw/source/archive/merge/main gates belong to
the root executor and immutable final receipts/release to PR35 metadata.

## Findings and one root fix pass
- Important R1: accepted browser-noncanonical origins initialize successfully but
  client normalization then causes the console's literal Host/Origin check to403.
  Reviewer reproduced it with actual built binaries. Eight invalid-origin cases
  (DNS case, default/padded ports, noncompressed IPv6, legacy numeric IPv4, empty
  hostname) observed RED then GREEN. Install-only validation rejects before writing;
  canonical HTTPS, custom port and IPv6 configs still load through actual runtime
  config. Existing runtime/browser semantics unchanged, no lower security checks.
- Important R2: standalone bundle originally copied only project LICENSE, omitting
  repository THIRD_PARTY_NOTICES and actual linked/bundled dependency notices.
  Actual old-bundle regression observed missing-notice RED. Collection now retains
  upstream Go/npm texts, third-party files, Go runtime and explicit musl/compiler
  copyright texts with versions/provenance and complete SHA256 coverage. Controller
  builds with musl-gcc; CLI CGO0. Native original full3jobs39258523119 green does not
  overrule these review findings and is not acceptance of the fixed distribution.
  Actual native39260404237 reproduced a platform npm binding with no LICENSE in
  its published archive; verified same-version/repository/SPDX parent supplies
  original project/third-party text, never an SPDX-generated template.
  react-remove-scroll-bar2.3.8 also omits LICENSE; exact npm metadata declares MIT,
  published gitHead b3b1287 is unavailable through the public GitHub file endpoint.
  Retain the verbatim upstream project LICENSE at immutable8ca9ba5ea52de03308fe8ced94f7b159a44d28ff,
  SHA256a79aae0c0f21990d9d963bb3c5a79cdcea9a46f8523ba55c58d7fe776b6ebc84;
  supplement pins name/version/repository/SPDX/hash and records its actual provenance.
  Source: https://github.com/theKashey/react-remove-scroll-bar/blob/8ca9ba5ea52de03308fe8ced94f7b159a44d28ff/LICENSE .
  Final corrected code native39261533014 notice/archive gate passes; exact final
  documentation head and merge/main still require their own complete gates.
- Minor deferred: install report's independentAudiences flag exercises management
  and decision-listener cross-key rejection, not Agent gRPC audience. Do not claim
  this install runner newly certifies the latter; prior independent Agent protocol
  tests are separate evidence.
- Minor deferred: the runner resets logs at restart, so its final key-log assertion
  checks the second process, not both. No actual key leak observed; do not claim an
  exhaustive first-process log assertion. Public archive content is checked separately.

## Rulings made (in order, including declined boundaries)
1. Routine design/plan/push/merge waits waived under autonomous mandate; inline TDD,
   focused artifacts and one independent review retained. Cost: autonomous scope judgment.
2. Existing host-owned linked worktree retained, new branch from exact main, old
   branches preserved. Cost: extra branch, no discarded user files.
3. Linux-amd64 manual opt-in bundle only; no automatic root installation or public
   Release publication. Cost: operators explicitly provision user/unit/paths.
4. Local Unix socket EPERM cannot be bypassed. Native full Go normal/race mandatory.
   Cost: no all-local success claim; local tests name the one restricted case.
5. Preserve executable modes in tar.gz and re-extract before install acceptance.
   Cost: nested archive; independently verify both ZIP bytes and tar/content manifest.
6. Sole complete-branch review runs alongside CI; subsequent findings fixed by root
   TDD, no re-review. Cost: reviewer does not certify final/main native execution.
7. Hostile concurrent replacement of a trusted parent is outside the explicit
   exclusive-service-user/trusted-parent boundary. Reject ordinary symlink parents;
   do not promise defense against an administrator who controls ancestor directories.
   Cost: deployment must protect those parents and ownership.
8. Host systemd/boot/ownership/firewall/public TLS acceptance stays separate. The unit
   is a manual template, Type=simple not readiness. Cost: no boot/proxy setup guarantee.
9. Packaged Controller→Agent→kernel→ingress→application fresh-install acceptance is
   the next increment; existing PR21 actual Gitea evidence is not re-labeled packaged
   deployment evidence. Graceful SSH only denies new flows. Cost: goal not yet complete.
10. Non-Linux-amd64, HA, PostgreSQL and upgrade rollback not certified. Single-process
    SQLite restore requires stopped-state backup, never implicit rerun/import.
    Cost: platform expansion and database rollback need separate real verification.
11. Concrete notice omission fixed, not comprehensive legal clearance. Make the
    static C runtime explicit musl and retain its/compiler notices; preserve upstream
    licenses and pinned supplemental provenance. Cost: future dependency/distribution
    changes still need upstream-license review. Official build rationale:
    https://musl.libc.org/about.html . No Apache/GPL boundary change.
12. Trust configured build tools/dependency locks; no malicious-tool, concurrent
    source mutation or ignored-input hermetic/reproducible-build claim. Reject all
    tracked/untracked source dirt before packaging and record exact commit/tree.
    Cost: CI/toolchain supply-chain trust is still necessary.
13. Root must verify exact final raw mandatory native steps, actual checkout tree,
    independent archive downloads/hashes, merge expected head/parents and full main.
    Cost: any missing/failed gate blocks merge/release; initial green is insufficient.
14. Native npm bindings without notice files use only verified same-version/project/
    SPDX parent texts; the one version-pinned supplement uses immutable project text
    with explicit provenance, not an asserted inaccessible release commit. Cost:
    missing/changed packages fail closed and may require a reviewed new supplement.

## Verification chronology / limits
Baseline CLI passes. Missing controller-init command RED→GREEN private config and
key/hash/scope/mode/invalid/existing/symlink tests. Missing-bundle actual acceptance
RED; existing-target preflight RED127→GREEN2, untracked source RED127→GREEN2 before
any output. Local CLI normal/race, Go vet/build, locked Web ci/typecheck/lint/build
pass. Full normal/race both fail only TestAuthorizationRuntimeTCPUnixAndShutdown/unix:
Unix socket EPERM, not silently skipped. First local real static artifact/manifest
passes, PATH-empty init/start/health and tested management/decision key rejection
pass; Chromium installation returns truncated/non-ZIP0MiB repeatedly, so no local
browser claim. No permission bypass or fake execution. Initial native39258523119
real full normal/race/static/Web/detached browser/restart passes before review fixes.
Native39260404237 notice collection RED on actual npm binding; later candidate
39260945743 likewise stops on missing scroll-bar notice, neither is acceptance.
Local corrected collection reaches only missing musl copyright (toolchain not
installed locally); native Ubuntu musl toolchain and complete archive gates required.

Corrected code accepted: local8a2b3ee962220a37f4c1dd6ae4db310a11f237e9,
remote92554c25705bf8dc54e9d314fd5cdb1399d5eb91 and actual checkout
a01407f9c0c788a6fb89790b5f8e61926dd7397d share tree
72c34fe872d51836a9e81f8cdb8bb97b3e3a4d95. Native39261533014 all three jobs
and mandatory steps succeed: Go118090142639 (247 top-level/607 total, no failed,
skipped or race tests), Web118090142370, install118090142737. Full normal/race,
vet/build, Unix100/new-credential100, static SQLite Controller and locked Web
checks remain mandatory and passed. Raw installation log has one apparent error
annotation: `licenses/go/github.com/jackc/pgx/v5@v5.10.0/pgproto3/notice_response.go: OK`.
This is a successful checksum line classified by the runner matcher, not a failed
command; all other error annotations are rejected, checksum command and step exit0.
Independently downloaded ZIP digests match metadata:
- bundle12191916746 fa9e0bd95fc891e3ec7ea7344fdb57df675f7dd9ef8ab49db4c59185c23bbfb8
- install12191866949 b2e156b44ce31b4dcca4d1628a14463a838959a15fbdaae7aa119b69e0377aee
- Web12191722021 21c80ee3dd8fcd204aab4650d1567095785a5c1eabdb2eed3312404332ae601d
Inner tar33302859 bytes, all240 manifest-covered files, exact SOURCE tree/checkout,
both executable static ELF binaries (19677319/43755824 bytes), no unsafe archive
entries/private state, and exact public report independently verified. Actual
runtime uses empty PATH after relocation; browser creates one human account,
publishes revision2, restarts with clean SIGTERM, rejects old session and reads
durable account/publication/audit. Desktop1440/mobile375 images inspected without
horizontal overflow. Final doc-head and main receipts/release remain PR35 metadata.

中文：一次独立全分支评审，两个Important均按真实失败加回归：拒绝导致浏览器无法
登录的非规范origin，补齐安装包第三方/依赖/静态工具链许可原文与哈希。延后两项
非阻塞覆盖提示（未测Agent gRPC用途、首进程日志）。不宣称法律审查、主机开机或
打包Agent/入口/应用联合部署完成；最终准确head、主线证据与租约释放见PR35。
