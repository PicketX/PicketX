# PicketX

English (default) | [简体中文](README.zh-CN.md)

**Simple, client-free, on-demand access authorization.**

PicketX is designed to grant time-bounded access to protected resources through self-service requests, policies and approval. IP/CIDR network permissions can be shared across matching devices and applications; existing application credentials remain required. Further identity-aware authorization integrates through trusted adapters.

## Project status

This repository is an **initial development scaffold**. It includes shared logical authorization and physical planning libraries, an Echo v5-based Go Controller liveness endpoint and a bilingual React starting page. Authentication, approval, Grant issuance, persistence, Agent synchronization and enforcement are not implemented. Do not use this scaffold to protect resources.

The control plane and Web live here. The official Linux Agent, `picketxd`, belongs to the separate **PicketXD** repository. This initialization does not create or modify that repository.

The selected backend stack is Go + Echo v5 + GORM, with go-playground/validator v10 for request fields. API documentation generation uses Swaggo v2.0.0-rc6 targeting OpenAPI 3.1, with Swagger UI v5.33.2 for documentation and interactive requests. GORM, validator, Swaggo and Swagger UI are architecture decisions; persistence, migrations, request validation and API documentation generation/display are not implemented yet.

Dependency assembly uses explicit constructors and module-level composition, with centralized lifecycle management; no DI framework is selected. See architecture section 5.7 for the design boundaries.

Runtime configuration loading is selected as Koanf v2.3.8, with typed module configuration and explicit file/environment/flag precedence. It is not implemented yet; the scaffold still reads `-listen` through the standard `flag` package.

Runtime logging uses standard-library `log/slog`. The scaffold already emits JSON to standard error; module logger injection, selectable formats, configurable levels and persisted authorization audit remain to be implemented.

The planned `picketx` CLI uses Cobra v1.10.2 with Koanf configuration. CLI commands are not implemented; the existing Controller entry point continues to use standard-library flags.

SQLite is selected as the official GORM driver (v1.6.0 baseline) with mattn/go-sqlite3 (v1.14.52 baseline). Default Linux releases must be fully static without additional shared-library installation. PostgreSQL uses official gorm.io/driver/postgres v1.6.3 with pgx/v5 v5.10.0. Database integration and the static-CGO release pipeline are not implemented; migration tooling remains deferred.

Initial enforcement scope is IP/CIDR network admission, HTTP/HTTPS authorization through existing ingress, and a local decision interface over TCP or Unix socket. NFQUEUE/mirror inspection, transparent proxy/TLS termination and native database proxies are deferred; these endpoints and integrations are not implemented yet.

Authorization is selected as Casbin RBAC with the official GORM Adapter. The business layer supplies the current user's trusted groups/roles; shared rules use in-process enforcement and configurable periodic policy synchronization, with startup scale/cost recommendations. ABAC is deferred until needed. This is not implemented; see [ADR 0001](docs/decisions/0001-rbac-authorization.md).

Runtime support selections are SCS Sessions, Echo built-in CSRF and separately configured ordinary/Agent rate limits, official gRPC/protobuf with independent Agent Tokens and optional server TLS, Prometheus metrics and standard-library periodic jobs. These remain unimplemented; see [ADR 0002](docs/decisions/0002-runtime-support.md).

M0 Linux data-plane validation is implemented and merged in [PicketXD](https://github.com/PicketX/PicketXD/pull/1). The [final tested source](https://github.com/PicketX/PicketXD/commit/13fdf2d9aacd79be74578ebe9c23af19a6b33d6a) passed [17 required real-kernel scenarios](https://github.com/PicketX/PicketXD/actions/runs/38547987642) on Linux 6.17.0-1022-azure / Go 1.27.0. This is reusable validation infrastructure, not a production Agent. The approved [M1.1 design](docs/superpowers/specs/2026-10-15-shared-network-core-design.md) and [implementation plan](docs/superpowers/plans/2026-10-15-shared-network-core.md) now have a standard-library-only logical engine in [`pkg/core`](pkg/core). It validates sources, selectors, entry scopes and explicit lifetime modes; unions matching Grants, prioritizes denies and explains independent contributions and future changes. Literal Gitea HTTPS/SSH tests cover overlap, revocation, immutable replacement and deadline-preserving replay. Core-to-M0 integration and the Controller workflow remain unimplemented; a logical allow does not confirm Agent enforcement.

The approved [M1.2 design](docs/superpowers/specs/2026-10-15-physical-network-planner-design.md) and [executed plan](docs/superpowers/plans/2026-10-15-physical-network-planner.md) now have a standard-library-only physical planner in [`pkg/networkplan`](pkg/networkplan), consuming Core's owned resolved view. It validates explicit TCP/UDP single-port definitions, trusted context boundaries/capabilities, Binding cross-products and physical conflicts; exposes owned immutable inventory and explicit-time allow/deny projection; and guards full replacements with former-target withdrawal provenance. Nil entry addresses use the explicit context boundary; non-nil empty addresses fail. Offline targets remain desired; unbound entries are unmanaged. NextChange includes future allow/deny boundaries without renewing absolute deadlines. The M1.1 omitted-start/historical-clock boundary regression is corrected.

Each plan has one authority; Controller/static plans are not merged. Context facts cannot broaden requested class/stage/destinations. Retargeting with retained typed Grant/Deny IDs fails; fresh-ID approval lineage and retention of original limits remain publisher obligations. Inventory and withdrawals are desired history, not proof of installation or a deletion list. The separate M1.3 PicketXD bridge still needs concrete guards/original tuples, runtime scheduling, applied historical inventory, apply confirmation before scoped strict cleanup and explicit empty-owned cleanup (M0 rejects empty Protected). Pure Go conformance does not establish host/container or packet compatibility. See [package guidance](pkg/networkplan/doc.go) and its external `ExampleCompile`.

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

Open the URL printed by Vite. The frontend and backend build independently. Vite proxies `/healthz` to the local Controller. The Controller does not serve static Web assets yet.

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
| `pkg/networkplan` | Scoped per-Agent physical intent and replacement guards |
| `apps/web` | React, TypeScript, Vite, shadcn/ui, Tailwind CSS, i18next |
| `docs` | Current bilingual product, architecture and configuration model baselines |

`pkg/core` contains the evolving logical network Go API and a runnable `ExampleCompile`; public wire/protocol packages remain deferred. See [Product Design](docs/PRODUCT_DESIGN.md), [Architecture](docs/ARCHITECTURE.md), [Configuration Model](docs/CONFIGURATION_MODEL.md), and [Web development](apps/web/README.md).

## License

Project code is licensed under [Apache-2.0](LICENSE). Copied third-party components retain their original notices; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). PicketXD has a separate GPL-3.0-or-later/OEM licensing design.
