# Private Controller initialization and portable Linux bundle

## Intent and boundary
Make the accepted real console/API available without a source checkout, Go, npm,
Python or shared-library installation at runtime. One Linux-amd64 bundle contains
fully static Controller and CLI plus built Web assets and an opt-in systemd unit.
This does not install onto the user's machine, expose listeners publicly, change
policy semantics or claim a fresh deployed Agent/kernel/Gitea acceptance.

## Design
`picketx controller-init --directory ABS --assets ABS --origin ORIGIN` creates a
new private directory, never reuses/overwrites any existing path. Its parent must
already exist and resolve without symlinks. Assets must resolve to an existing
directory with a regular index.html. Validate all inputs before any writes.
Defaults: HTTP127.0.0.1:8080, Agent127.0.0.1:9090, local decision127.0.0.1:9081;
operator may explicitly override each with a loopback IP and nonzero port.
Agent ref defaults node-a and ingress route UID route-web; both configurable and
validated against the existing contracts. Authority is newly randomized per init.
Origin uses existing exact HTTPS/loopback-HTTP validation plus install-only rejection
of browser-noncanonical host/IP/default/padded port spellings before writing.

Generate three independent cryptographically random256-bit keys, raw files
bootstrap-key/agent-token/ingress-token0600; respective SHA256-only registries
principals.json/agents.json/callers.json0600. Bootstrap is role:admin recovery
access, Agent is bound to one Agent ref, caller to one route UID. No default
Grants, human accounts, Resources, Bindings or HTTP routes are installed. Database
source and database human authority are explicit/sticky; policy/facts files absent.
Config controller.yaml0600 uses absolute paths including private state SQLite.
Directory0700. Output reports paths only, never keys. Init never starts services,
modifies host ownership/firewall or prints a shell command containing secrets.
Failed creation leaves the exclusively-created directory for operator inspection
and refuses rerun; never recursively deletes or rolls back somebody else's files.
Trusted parent and exclusive service user remain deployment requirements.

Build script produces a new output directory only; rejects existing output, copies
regular Web files (no symlinks), includes licenses and bilingual runbooks. Build
requires Go/CGO/Ubuntu-Debian musl-tools, Node/npm and readelf; runtime does not. It runs
locked npm ci/typecheck/lint/build, verifies no ELF INTERP/NEEDED for both binaries,
records source commit/tree and SHA256 manifest. Retains project third-party notices
and verbatim Go/npm/musl/compiler license files with an index and checksums; this
does not certify comprehensive legal clearance. No automatic GitHub Release upload.
Bundle remains relocatable until init; config uses absolute deployment paths.
Sample systemd unit is manually installed with explicit unprivileged user, unit
and /opt paths; no root auto-installer, no claim of boot compatibility from lint.

## Acceptance
TDD command tests: exclusive directory/no overwrite/symlink parent rejection,
invalid origins/assets/IDs/listeners no side effects; real generated config loads,
secrets independent and never output, modes and hash/scopes correct. Full Go/race/
vet/build and locked Web gates. Native CI builds bundle, copies outside checkout,
runs CLI init and actual Controller with runtime PATH empty, health/readiness,
unauthenticated and cross-key rejection; real Chromium bootstrap login creates
database human, signs in, modifies publication, restarts process and proves durable
account/publication/audit. Screenshots/public report exclude all credentials.
One independent whole-branch review, exact tree/PR/head/raw/download/main checks.
Next separate increment: packaged Controller→deployed Agent→ingress/application
fresh-install/reboot acceptance; retain graceful SSH limitation.

## Context map
| File | Responsibility / relation |
| --- | --- |
| internal/cli/config.go, controller_init.go | Cobra command assembly and new private init adapter |
| internal/cli/controller_init_test.go | command filesystem/validation/secret behavior |
| internal/runtimeconfig/config.go | consumer of generated runtime document, unchanged |
| internal/management/origin.go | existing exact origin validation, unchanged |
| scripts/build-controller-bundle.sh | static artifacts + locked Web packaging |
| web/e2e/install.mjs | real detached bundle/bootstrap/restart browser acceptance |
| deploy/controller/* | bilingual runbooks and opt-in service template |
| .github/workflows/ci.yml | mandatory bundle/browser gate and public evidence artifact |

Risk: no API/database migration; no domain behavior; new opt-in CLI filesystem writes.
中文：交付独立安装包和私有初始化，不自动修改宿主机。三类密钥独立、权限范围固定，
数据库权威从空策略启动，凭据只落0600文件。原生浏览器验证真实开户、发布与重启恢复；
完整安装到Agent/入口/应用的验收另做，不据此宣称整体完成。
