# Standalone Controller: private first installation

English | [简体中文](README.zh-CN.md)

This opt-in Linux-amd64 bundle contains fully static `bin/picketx-controller`,
`bin/picketx` and built React `web/`. Go/npm/Python and separately installed shared
libraries are **build-time only**, not runtime requirements. Nginx/another existing
trusted ingress remains responsible for public TLS and application forwarding.
This is one single-process SQLite Controller, not HA/PostgreSQL or an auto-installer.
The systemd file is a template: native process/browser acceptance is not a boot test.

## Build and inspect

In a clean committed checkout with Go (version in go.mod), Node (in .node-version),
npm, Ubuntu/Debian musl-tools, file/readelf/coreutils:

```sh
bash scripts/build-controller-bundle.sh /absolute/new/controller-bundle
cd /absolute/new/controller-bundle
sha256sum --check SHA256SUMS
```

The output must not already exist. SOURCE.json records the exact commit/tree.
Controller builds explicitly with musl-gcc; CLI uses CGO_ENABLED=0. The bundle keeps
THIRD_PARTY_NOTICES.md and verbatim installed Go/npm/musl/compiler notices under
licenses/, indexed with dependency versions and covered by SHA256SUMS. Notice
collection is not comprehensive legal clearance; no upstream license is replaced.
SHA256 detects corruption, not publisher identity: obtain the archive/checksum
through a trusted channel and compare the expected source to CI evidence. Do not
run arbitrary PR artifacts with host privileges. No release is automatically
published. Move the bundle to its final path **before initialization**.

## Manual unprivileged service installation

For a fresh machine only; adapt distro account commands. Inspect every destination
first. Never overwrite an existing deployment or initialize over existing state.
Create an unprivileged `picketx` user/group, place the verified bundle at
`/opt/picketx` (root-owned, service-readable, not service-writable). Create only the
private parent `/var/lib/picketx`, owned by that service user, mode0700. Then:

```sh
sudo -u picketx /opt/picketx/bin/picketx controller-init \
  --directory /var/lib/picketx/controller \
  --assets /opt/picketx/web --origin https://picketx.example.com
```

Parent and asset paths must exist without symlinks. Initialization refuses every
existing target, generates a unique authority, opens no listener and installs no
Grant. On I/O failure the partial private directory remains for inspection; do not
retry into it or delete live state. The config is JSON syntax, valid YAML. It uses
absolute paths; relocate by an explicit stopped-service config edit, not by rerun.
Origins must use browser-canonical spelling: lowercase ASCII/punycode host, canonical
IP/compressed IPv6, no padded port or explicit default80/443. Noncanonical origins
are rejected before writes, since browser normalization would otherwise prevent login.

Inspect `controller.yaml`. Source and human-account authority are **database**, not
file polling; policy/facts are deliberately absent. SQLite is private
`controller/state/controller.sqlite`. Default listeners are loopback HTTP8080,
Agent9090 and decision9081. `--listen`, `--agent-listen`, `--authz-listen` permit
distinct loopback IPv4 or IPv6 IP:nonzero-port values. They do not make remote
listeners public. Remote Agents require a separately reviewed explicit runtime
configuration with normal server TLS or an existing secure tunnel; no mTLS mandate.
Do not expose Agent plaintext or bearer-protected decision API on a public network.

Copy `deploy/picketx-controller.service` to your system's unit directory only after
the binary/config/user paths match. It runs without capabilities, with a strict
read-only filesystem except the initialized state directory. Start/enable manually;
verify `http://127.0.0.1:8080/healthz` and `/readyz` separately. Type=simple is process
startup, not readiness. Stop uses SIGTERM. No nftables/kernel privilege is required
by this Controller. A real host boot/systemd/Agent acceptance is a separate gate.

## First sign-in and credentials

Read `bootstrap-key` privately and paste it into the real console's **Access key**
sign-in at your configured exact origin `/console/`. The command prints paths only,
never keys. Do not put secrets in URLs, shell history, screenshots, issue/CI logs or
sample configurations. In **Accounts**, create an individual human with Admin role,
then sign in using that username/password. Database accounts, roles, disabling,
password resets and audit are managed there. Restart invalidates in-memory browser
sessions, not durable accounts/publications/audits.

Three separate random256-bit raw files (0600 inside0700 directory):

| Raw private file | SHA256-only registry | Authority |
| --- | --- | --- |
| bootstrap-key | principals.json | principal:bootstrap, role:admin recovery/API access |
| agent-token | agents.json | initial-agent, Agent ref node-a only |
| ingress-token | callers.json | initial-ingress, HTTP route UID route-web only |

`--agent-ref` and `--route-id` select those initial scopes, using1–128 letters,
digits, dot, underscore or hyphen. These files do not create the Agent facts,
Resources, Bindings, routes or permissions: publish actual host scopes through the
console; ensure the actual Agent and ingress refer to the same chosen IDs. Approval
and publication are not confirmation of kernel application. Source Grants are not
application identity, passwords, SSH keys or repository permissions.

Protect the bootstrap key as an offline recovery credential; its registry entry
remains authoritative independently of database human accounts. To revoke/rotate,
atomically replace its hash registry with a new private0600 file, retain another
tested admin recovery path, and verify invalidation. Agent and ingress credentials
are independent: never reuse the bootstrap key for either. A caller is bound to
explicit route UIDs, never all routes. Use the existing `examples/http` ingress
contract and overwrite untrusted original-request headers at the trusted proxy.
Portal reverse-proxy headers do not magically establish the network subject;
operator-approved source/facts must match actual enforcement. Grant unions, deny
priority, independent absolute expiry and graceful/strict flow behavior remain as
documented by the Agent. Graceful SSH denies new flows, not admitted established ones.

## Backup, restart and upgrade

Stop the Controller cleanly before copying the complete private state directory
(or use a tested SQLite online backup procedure); protect backups like credentials.
Keep runtime config, registry files, keys and database together with their authority.
Do not copy only the SQLite main file while WAL is active. Restore ownership0700/
0600 and the original authority; do not run init on restored state. Keep the old
verified bundle and a pre-upgrade stopped-state backup. Migration rollback is not
promised: do not open an upgraded database with an older binary without a tested
restore. No schema/automatic data migration is introduced by this bundle increment.

Native CI installs a relocated bundle outside source with an empty runtime PATH,
uses real Chromium to create/sign-in a human, publishes and restarts/readbacks
SQLite/audit, rejects cross-audience keys and old sessions, and inspects artifacts.
This does not yet establish a packaged Agent→ingress→Gitea deployment or host reboot.
