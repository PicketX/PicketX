# PicketX

English (default) | [简体中文](README.zh-CN.md)

**Simple, client-free, on-demand access authorization.**

PicketX is designed to grant time-bounded access to protected resources through self-service requests, policies and approval. IP/CIDR network permissions can be shared across matching devices and applications; existing application credentials remain required. Further identity-aware authorization integrates through trusted adapters.

## Project status

This repository is an **initial development scaffold**. It includes an Echo v5-based Go Controller liveness endpoint and a bilingual React starting page. Authentication, approval, Grants, persistence, Agent synchronization and enforcement are not implemented. Do not use this scaffold to protect resources.

The control plane and Web live here. The official Linux Agent, `picketxd`, belongs to the separate **PicketXD** repository. This initialization does not create or modify that repository.

The selected backend stack is Go + Echo v5 + GORM, with go-playground/validator v10 for request fields. API documentation generation uses Swaggo v2.0.0-rc6 targeting OpenAPI 3.1, with Swagger UI v5.33.2 for documentation and interactive requests. GORM, validator, Swaggo and Swagger UI are architecture decisions; persistence, migrations, request validation and API documentation generation/display are not implemented yet.

Dependency assembly uses explicit constructors and module-level composition, with centralized lifecycle management; no DI framework is selected. See architecture section 5.7 for the design boundaries.

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
| `apps/web` | React, TypeScript, Vite, shadcn/ui, Tailwind CSS, i18next |
| `docs` | Current bilingual product, architecture and configuration model baselines |

Core/public protocol packages will be introduced as their first contracts are implemented. See [Product Design](docs/PRODUCT_DESIGN.md), [Architecture](docs/ARCHITECTURE.md), [Configuration Model](docs/CONFIGURATION_MODEL.md), and [Web development](apps/web/README.md).

## License

Project code is licensed under [Apache-2.0](LICENSE). Copied third-party components retain their original notices; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). PicketXD has a separate GPL-3.0-or-later/OEM licensing design.
