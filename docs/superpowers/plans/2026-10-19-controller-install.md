# Controller installation implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline, then one fresh independent whole-branch review.

**Goal:** Portable static Controller bundle with private initialization and actual first-use acceptance.
**Architecture:** Extend existing Cobra diagnostics with an opt-in exclusive init command;
package existing runtime and React without changing their authority semantics.
**Tech Stack:** existing Go/Cobra/Koanf/GORM SQLite/React/Vite/npm/Playwright.
**Spec:** docs/superpowers/specs/2026-10-19-controller-install.md

## Global Constraints
- Core298863f4e0cb1a72ef18e8abe77f8b8fb28464de; Agent233f081 accepted, unchanged.
- Three independent256-bit credentials,0600 files/new0700 directory, never overwrite.
- Loopback-only initial listeners; database source/users authority; no seeded Grants.
- Linux-amd64 fully-static artifacts, regular Web files, runtime without build tools.
- Existing isolated worktree; expected-head writes/merge; no force/no host installation.

## Review Focus
- Existing target/symlinked parent must not be overwritten or disclose credentials.
- Public origin and encoded paths/listeners/IDs must not produce a broken runtime.
- One key must not authenticate to another audience; fresh authority stays empty.
- Packaging must reject dynamic artifacts and exclude credentials/private state.
- Process restart must prove durable account/publication, not reuse a test fixture.

### Task 1: Private initialization command
**Files:** internal/cli/config.go, controller_init{,_test}.go.
**Interfaces:** consumes existing origin/config validation; produces Cobra
controller-init directory with controller.yaml and three key/registry pairs.
- [x] Write real command tests for successful initialization, generated config,
  audiences/modes/output, invalid input, preexisting paths, parent symlink and rerun.
- [x] Run `go test ./internal/cli -run ControllerInit -v`; expected missing command RED.
- [x] Implement command with validation before writes and exclusive target creation.
- [x] Run focused tests and full available Go normal/race/vet/build; expected focused
  GREEN; document any known local Unix socket EPERM, native full gate mandatory.
- [x] Commit tested command.

### Task 2: Bundle and actual first-use verification
**Files:** scripts/build-controller-bundle.sh, deploy/controller/*,
web/e2e/install.mjs, .github/workflows/ci.yml, delivery/review docs.
**Interfaces:** consumes Task1 command and actual built web; produces detached
bundle + SHA256/source metadata and public browser install evidence.
- [x] Write actual install runner requiring bundle executables; run against absent
  bundle, expected assertion RED; add invalid existing-output build invocation test.
- [x] Implement fail-closed static build/package script and bilingual service/runbook.
- [x] Actual runner verifies relocated PATH-empty init/start, audience rejection,
  browser bootstrap/human creation/publication/restart/readback and public screenshots.
- [x] Native full Go and locked Web plus mandatory install run; inspect raw source,
  report and downloaded archive bytes/hash; commit checkpoints.
- [ ] One independent whole-branch review, one Critical/Important TDD fix pass;
  defer Minors, no re-review. Exact final CI/head merge and main verification.

Final immutable PR/CI/head/tree/archive IDs and lease release are authoritative in
final PR metadata, avoiding self-referential commits. Continue existing automation.
中文：按TDD内联实现私有初始化及静态包，再独立全分支评审和准确head主线验收。
