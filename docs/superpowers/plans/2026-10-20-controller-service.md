# Controller service implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline; one independent whole-branch review at end.

**Goal:** Real packaged systemd Controller lifecycle and stopped complete restore.
**Architecture:** Existing Chromium install flow plus opt-in systemd fixture and
strict private filesystem helper; retain detached acceptance separately.
**Tech Stack:** Existing Go static/SQLite, Node/Playwright and Linux systemd.
**Spec:** docs/superpowers/specs/2026-10-20-controller-service.md

## Global Constraints
- Core93024b229c072ce1db8512323c3d7fa35e3df74e/tree64ed0dcc4b6a3232e093150d7fc100cc9a852e43.
- Agent5373595 unchanged; Apache/GPL boundary; no production/UI/kernel change.
- Opted-in disposable Linux; unique private resources; empty runtime PATH.
- No reboot, online backup or migration rollback claim; no skipped native gate.

## Review Focus
- Symlink/special/loose state cannot copy or overwrite operator paths.
- Restore must use new state, not still-read original SQLite/registry.
- Actual process UID/capabilities and shipped sandbox remain effective.
- Crash restart/clean stop refer to current actual unit/PID, not stale HTTP peer.
- Cleanup touches only created resources; errors cannot expose credentials.

### Task 1: Strict stopped-state filesystem helper
**Files:** scripts/controller-state-copy{,.test}.mjs.
**Interfaces:** snapshotPrivateTree(dir) returns sorted relative-path SHA256 map;
copyPrivateState(source,newTarget) returns snapshot; relocateInstanceConfig(config,old,new)
changes only four known private file paths.
- [ ] Real FS tests: full bytes, existing target, symlink, loose mode, special file,
  malformed relocation reject. Node run RED against missing behavior.
- [ ] Minimal helper; full scripts/*.test.mjs GREEN; commit.

### Task 2: Real service and browser acceptance
**Files:** fixture, install runner, workflow, bilingual docs/checkpoints.
**Interfaces:** createControllerService({temp,bundle,stateParent}) returns identity,
start(instance), stop(), crashRestart(), logs(), close(), public proof.
- [ ] Mandatory service-mode proof gate: actual native RED before implementation.
- [ ] Implement real systemd lifecycle and stopped new-target restore; make original
  path unavailable; exact durable views and fresh human/browser readbacks.
- [ ] Full Go/race/vet/build/locked Web/Node and actual native install/service gates.
- [ ] One independent review, one Important RED→GREEN fix pass, defer Minors;
  exact artifacts/source/expected-head merge/main receipt and immediate release.

Self-review: all five Review Focus inputs map to real FS/native assertions; shared
helper/fixture interfaces explicit. Final immutable evidence belongs in PR metadata.
