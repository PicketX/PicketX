# Installed Controller service and stopped restore

## Intent
Prove the existing static package runs under its unprivileged systemd template,
and a stopped complete private instance restores without reinitialization or loss
of authority/accounts/publication/audit. Architectural acceptance harness only:
no production authorization/API/UI/schema/kernel change. Agent5373595 unchanged.

## Design
Retain detached Chromium install acceptance; repeat through a real system-manager
fixture on opted-in disposable Linux CI. Create only one unique temporary user,
unit and /run directory, and clean only those resources. Retain shipped sandbox/
restart values; change user/group, binary/working/config/state paths and empty PATH.
Check actual UID, CapEff=0, NoNewPrivs=1 and executable digest. Type=simple active
is not ready: require real HTTP readiness. Kill actual PID, verify automatic new
PID restart, clean SIGTERM, session invalidation and durable browser readbacks.

After clean stop, snapshot every private regular file including SQLite/WAL,
registries, keys and runtime config. Reject symlinks/special files/loose modes or
existing destinations. Copy complete tree to NEW0700 backup and NEW0700 restore.
Compare every file digest before rewriting only the four known private absolute
paths in restored JSON-syntax controller.yaml. Preserve every other field including
authority/origin/assets. Restore ownership, make original path unavailable and
switch unit to restored state. No init on restore. Read back exact publication,
accounts/audit, reject old session and sign in freshly. No private state uploaded.

## Alternatives / limits
Mock systemctl or process spawn cannot establish service proof; real manager is
selected. Rebooting the runner destroys its execution agent and is not this gate.
No real machine reboot, Agent Controller-mode service, online backup, migration
rollback, HA or application-policy change claimed. Existing installed Agent proof
continues to establish actual immutable Grant deadlines; this empty-policy init
fixture must not invent Grant coverage. Local lacks PID1 systemd; native required.

## Context Map (reviewed)
| File | Responsibility/change |
|---|---|
| scripts/controller-state-copy{,.test}.mjs | strict complete private tree copy/config relocation, real FS tests |
| scripts/controller-service-fixture.mjs | opt-in unique actual systemd fixture and process identity |
| web/e2e/install.mjs | existing browser flow, service mode and restore readbacks |
| .github/workflows/ci.yml | existing three jobs, additional mandatory service install step |
| deploy/controller/README{,.zh-CN}.md | tested operations and remaining limits |
| docs/AUTONOMOUS_DELIVERY{,.zh-CN}.md | short lease/checkpoint, final PR receipt |
Dependencies: shipped service template, init CLI, real browser API/SQLite. No public
interface, configuration schema or database migration. Autonomous authorization
waives routine review pauses, not TDD/independent review/native validation.

## Acceptance
Real FS RED→GREEN; full Go normal/race/vet/build, locked Web and prior native gates;
actual service/browser/restore receipt; one fresh full-branch review and one
Important fix pass; actual archives/source, expected-head merge and full main proof.
