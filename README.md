# PicketX

English (default) | [简体中文](README.zh-CN.md)

**Simple, client-free, on-demand access authorization.**

PicketX is designed to grant time-bounded access to protected resources through self-service requests, policies and approval. IP/CIDR network permissions can be shared across matching devices and applications; existing application credentials remain required. Further identity-aware authorization integrates through trusted adapters.

## Project status

This repository is under active development, not a complete access-management product. Core/configuration, durable SQLite publication, authenticated gRPC and the native PicketXD Controller Provider are implemented. The [managed API](docs/MANAGEMENT_API.md) and [browser console](docs/BROWSER_CONSOLE.md) supply independent principals/Casbin/CAS/audit, SCS/CSRF and live configuration. The [alpha HTTP adapter](docs/HTTP_AUTHORIZATION.md) provides source-backed HTTPRoute, scoped local decisions and real Nginx integration; ingress/caller setup remains explicit and separate.

Live management includes [Grant issue/revoke](docs/GRANT_MANAGEMENT.md), [Resource/Endpoint creation, network edits/removal and exact-host Binding creation, named-entry edits/removal](docs/RESOURCE_MANAGEMENT.md), HTTPRoute create/read/remove, [local human password/key login](docs/HUMAN_LOGIN.md), and [durable requests with independent narrowing approval/rejection](docs/ACCESS_REQUESTS.md). Approval atomically publishes a bounded Grant; it is not confirmation of execution. Exact source/CI and integration status are in [the checkpoint](docs/AUTONOMOUS_DELIVERY.md).

[OIDC sign-in](docs/OIDC_LOGIN.md) adds an opt-in verified issuer/subject binding to existing human accounts, with current local roles and SCS sessions; no claim-role or network permission is inferred. Managed account lifecycle, native human approval/Agent/Nginx/Gitea and standalone service/recovery increments are recorded in the checkpoint. Real deployed-IdP acceptance and its installed Agent/application/recovery conjunction remain; signed protocol fixtures do not establish those. No complete production product is claimed.

[Node observations](docs/NODE_OBSERVATIONS.md) separate distributed targets, stream presence and report freshness. Agent assertions are not independent kernel verification; continuous applied-state proof remains out of scope.

The control plane and Web live here. The official Linux Agent, `picketxd`, belongs to the separate **PicketXD** repository. This initialization does not create or modify that repository.

The backend uses Go + Echo v5 + GORM, with validator v10 and pinned Swaggo v2.0.0-rc6/OpenAPI3.1 integrated for management. GORM/SQLite persists publications, source binding and change audit through explicit migrations. Swagger UI v5.33.2 rendering and broader workflow schemas remain pending.

Dependency assembly uses explicit constructors and module-level composition, with centralized lifecycle management; no DI framework is selected. See architecture section 5.7 for the design boundaries.

Runtime configuration loading is selected as Koanf v2.3.8, with typed module configuration and explicit file/environment/flag precedence. It is implemented for the Controller; explicitly changed standard-library flags feed the same typed configuration boundary.

Runtime logging uses standard-library `log/slog`, JSON stderr and injected module loggers. Managed configuration audit is separately persisted; selectable formats/levels and request-time authorization records remain pending.

The `picketx` CLI uses Cobra v1.10.2 for configuration/deployment commands and authenticated [Controller status/doctor](docs/CLI_DIAGNOSTICS.md). Diagnostics separate desired versions from current Agent assertions and never claim independent kernel verification. The Controller loads Koanf runtime configuration.

SQLite is selected as the official GORM driver (v1.6.0 baseline) with mattn/go-sqlite3 (v1.14.52 baseline). Default Linux releases must be fully static without additional shared-library installation. PostgreSQL uses official gorm.io/driver/postgres v1.6.3 with pgx/v5 v5.10.0. The Controller publication store and fully static SQLite artifact smoke are implemented; PostgreSQL, business migrations and release packaging remain pending.

Initial enforcement scope is IP/CIDR network admission, HTTP/HTTPS authorization through existing ingress, and a local decision interface over TCP or Unix socket. The [alpha HTTP adapter](docs/HTTP_AUTHORIZATION.md) adds source-backed HTTPRoute, scoped local TCP/Unix decisions, durable audit and Nginx integration; exact CI acceptance is in the checkpoint. It is not verified-user authorization or complete product acceptance. NFQUEUE/mirror inspection, transparent proxy/TLS termination and native database proxies remain deferred.

Management API RBAC uses Casbin3.11.0 and official GORM adapter3.41.0, with trusted independent-principal roles, combined deny precedence and observable periodic reload. Browser sessions resolve these same trusted roles on every request; local human users are implemented, while OIDC and per-resource delegation remain pending. See [ADR 0001](docs/decisions/0001-rbac-authorization.md) and the [implemented boundary](docs/MANAGEMENT_API.md).

Runtime support uses SCS2.9.0 browser sessions, native Echo CSRF, independent ordinary/Agent limits and official gRPC with independent Agent Tokens and optional server TLS. Metrics and broader runtime integration remain pending. See [ADR 0002](docs/decisions/0002-runtime-support.md).

## Historical implementation stages

The stages below retain their original evidence and stage-specific limitations;
the project status and checkpoint above supersede their old pending-work lists.

M0 Linux data-plane validation is implemented and merged in [PicketXD](https://github.com/PicketX/PicketXD/pull/1). The [final tested source](https://github.com/PicketX/PicketXD/commit/13fdf2d9aacd79be74578ebe9c23af19a6b33d6a) passed [17 required real-kernel scenarios](https://github.com/PicketX/PicketXD/actions/runs/38547987642) on Linux 6.17.0-1022-azure / Go 1.27.0. This is reusable validation infrastructure, not a production Agent. The approved [M1.1 design](docs/superpowers/specs/2026-10-15-shared-network-core-design.md) and [implementation plan](docs/superpowers/plans/2026-10-15-shared-network-core.md) now have a standard-library-only logical engine in [`pkg/core`](pkg/core). It validates sources, selectors, entry scopes and explicit lifetime modes; unions matching Grants, prioritizes denies and explains independent contributions and future changes. Literal Gitea HTTPS/SSH tests cover overlap, revocation, immutable replacement and deadline-preserving replay. The Controller workflow was not part of M1.1; a logical allow does not confirm Agent enforcement.

M1.2 is implemented and merged in [PR #7](https://github.com/PicketX/PicketX/pull/7). The approved [M1.2 design](docs/superpowers/specs/2026-10-15-physical-network-planner-design.md) and [executed plan](docs/superpowers/plans/2026-10-15-physical-network-planner.md) now have a standard-library-only physical planner in [`pkg/networkplan`](pkg/networkplan), consuming Core's owned resolved view. It validates explicit TCP/UDP single-port definitions, trusted context boundaries/capabilities, Binding cross-products and physical conflicts; exposes owned immutable inventory and explicit-time allow/deny projection; and guards full replacements with former-target withdrawal provenance. Nil entry addresses use the explicit context boundary; non-nil empty addresses fail. Offline targets remain desired; unbound entries are unmanaged. NextChange includes future allow/deny boundaries without renewing absolute deadlines. The M1.1 omitted-start/historical-clock boundary regression is corrected.

Each plan has one authority; Controller/static plans are not merged. Context facts cannot broaden requested class/stage/destinations. Retargeting with retained typed Grant/Deny IDs fails; fresh-ID approval lineage and retention of original limits remain publisher obligations. Inventory and withdrawals are desired history, not proof of installation or a deletion list. The separate M1.3 PicketXD bridge now validates concrete guards/original tuples, fixture boundary scheduling, applied historical inventory, apply confirmation before scoped strict cleanup and explicit empty-owned cleanup. Pure Go conformance does not establish host/container or packet compatibility. See [package guidance](pkg/networkplan/doc.go) and its external `ExampleCompile`.

M1.3 has a real-kernel-verified implementation in [PicketXD PR #3](https://github.com/PicketX/PicketXD/pull/3), with all 17 M0 + 19 required bridge cases passing on [the final pushed head](https://github.com/PicketX/PicketXD/actions/runs/38750455517). It confirms shared-plan application and scoped withdrawal in isolated transit fixtures; production host/container deployment remains future work.

M2A adds [`pkg/configuration`](pkg/configuration) and `picketx config check`: strict versioned YAML, pinned Resource references, independent absolute lifetimes, complete planner validation/replacement and explicit empty withdrawal. See the [runnable Gitea example](examples/static/README.md). This is a usable configuration-check entry point; runtime providers/LKG, Agent deployment/synchronization, L7 and administration remain unimplemented.

M2B is merged in [PicketXD PR #4](https://github.com/PicketX/PicketXD/pull/4): `picketxd prepare` durably recovers static desired state, reloads complete configurations and schedules absolute boundaries. Its [final CI](https://github.com/PicketX/PicketXD/actions/runs/38760479190) passes all 43 required kernel scenarios, including seven host INPUT cases. Preparation reports `enforcementConfirmed=false`; production execution remains pending. The [M1.3 implementation plan](docs/superpowers/plans/2026-10-15-core-m0-bridge.md) is retained with links to the executed Agent increments. Continue through the [basic-function roadmap](docs/BASIC_FUNCTION_ROADMAP.md).

M2D1 implements the [Controller Agent stream foundation](docs/AGENT_STREAM.md): public typed gRPC/protobuf, complete node snapshots, revocable Tokens, correlated Agent assertions and bounded lifecycle, tested over TCP loopback and verified TLS. M2D2 below adds production process assembly and durable revisions; the PicketXD Controller provider and enrollment management remain pending, so end-to-end Controller Mode is not claimed.

M2D2 adds a [running Controller publisher](docs/CONTROLLER_RUNTIME.md): strict Koanf file/env/explicit-flag loading, GORM/SQLite durable complete revisions, independent credential revocation reload, optional server TLS and bounded HTTP/gRPC lifecycle. The subsequent [PicketXD PR16](https://github.com/PicketX/PicketXD/pull/16) verifies actual Core/SQLite→Agent→IPv4/IPv6 TCP/UDP allow/revoke/withdraw with provider-bound LKG. This management increment adds explicit database authority and a real authenticated API, not a UI or L7 adapter. Static CI checks actual file and managed SQLite/Casbin/API/audit/restart/SIGTERM binaries. See [delivery checkpoint](docs/AUTONOMOUS_DELIVERY.md) for exact acceptance/remaining limits; historical M0–M2D1 paragraphs above describe their original slices, not current completion.

## Development

Requirements: Go 1.27+, Node.js 24 LTS and npm.

Start the Controller from the repository root:

```bash
go run ./cmd/picketx-controller
```

It listens on `127.0.0.1:8080`; use `-listen` to change the address. `GET /healthz` returns process liveness only, not readiness to authorize access.

Start the Web development server in another terminal:

```bash
cd apps/web
npm ci
npm run dev
```

Go and Web build independently. For a working console, first configure database authority, management credentials and the opt-in browser origin as described in [browser deployment](docs/BROWSER_CONSOLE.md); the health-only default cannot authenticate it. Vite proxies `/healthz` and `/console/api`; a configured Controller can serve built assets at `/console/`.

## Verification

```bash
go test ./...
go vet ./...
go build ./...

cd apps/web
npm run typecheck
npm run lint
npm run build
```

## Layout and design

| Path | Purpose |
| --- | --- |
| `cmd/picketx-controller` | Local development server entry point |
| `internal/controller` | HTTP handler and tests |
| `pkg/core` | Logical source-to-entry authorization and resolved membership |
| `cmd/picketx`, `internal/cli` | Read-only Cobra configuration check |
| `pkg/configuration` | Strict alpha policy decoding and immutable compiler candidates |
| `pkg/networkplan` | Scoped per-Agent physical intent and replacement guards |
| `apps/web` | React, TypeScript, Vite, shadcn/ui, Tailwind CSS, i18next |
| `docs` | Current bilingual product, architecture and configuration model baselines |

`pkg/core` contains the evolving logical network Go API and a runnable `ExampleCompile`; public wire/protocol packages remain deferred. See [Product Design](docs/PRODUCT_DESIGN.md), [Architecture](docs/ARCHITECTURE.md), [Configuration Model](docs/CONFIGURATION_MODEL.md), and [Web development](apps/web/README.md).

## License

Project code is licensed under [Apache-2.0](LICENSE). Copied third-party components retain their original notices; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). PicketXD has a separate GPL-3.0-or-later/OEM licensing design.
