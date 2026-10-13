# PicketX development

Read `docs/PRODUCT_DESIGN.md` and `docs/ARCHITECTURE.md` before changing domain behavior. English documentation is the default; keep corresponding Chinese documents in sync.

Read `docs/CONFIGURATION_MODEL.md` before implementing configuration schemas or bindings. Keep business workflows, standard desired-state objects and execution state separate; follow the documented omission/empty-value semantics and do not silently finalize its open decisions.

- This repository contains the Go control plane/core and `apps/web`. The Linux enforcement Agent belongs to PicketXD.
- Controller HTTP uses Echo v5. Keep framework types in HTTP adapters; business and shared configuration/core packages remain framework-independent.
- Controller persistence uses GORM. Keep ORM types in persistence implementations and use versioned migrations for deployed databases; SQLite uses official gorm.io/driver/sqlite v1.6.0 + mattn/go-sqlite3 v1.14.52 as integration baselines; PostgreSQL uses official gorm.io/driver/postgres v1.6.3 with its pgx/v5 v5.10.0 dependency as the integration baseline; migration tooling remains deferred. These drivers are selected, not implemented.
- Use go-playground/validator v10 for HTTP DTO fields; keep complex configuration invariants in shared Go validation functions and preserve omission/null/empty/zero semantics. Add the dependency when an implemented endpoint requires it.
- API documentation uses Swaggo v2 CLI pinned to v2.0.0-rc6 with `--v3.1`; introduce generation and CI contract checks with the first documented business endpoint. Keep ordinary Echo handlers; Swagger UI v5.33.2 is selected for self-hosted rendering. Code generation is not an initial requirement.
- Use explicit constructor injection and focused module assembly; no DI framework is selected. Keep dependency directions inward, avoid global service lookup, and centrally manage startup/readiness/failure cleanup/shutdown. Request identities and transactions must not live in shared service fields.
- Runtime configuration uses Koanf v2.3.8 at the loading/assembly boundary when implemented; pass typed values to modules. Explicit flags override present env values, then files, then defaults. Reject unknown keys and keep runtime configuration separate from authoritative policy manifests.
- Runtime logging uses standard-library slog. Construct/inject module loggers at assembly, keep structured fields consistent and redact secrets. Runtime logs do not replace persisted authorization/audit records.
- CLI uses Cobra v1.10.2 when implemented, with Koanf and explicit flag precedence. Keep command handlers thin and framework types out of business/Core; use authenticated Controller APIs or the scoped local Agent API. Send results to stdout and diagnostics to stderr.
- Default release binaries require no separately installed shared libraries; Linux releases must be fully static, including SQLite/C runtime. CGO is allowed in controlled builds; inspect and smoke-test actual artifacts before claiming standalone deployment.
- Initial enforcement scope is IP/CIDR network admission plus HTTP/HTTPS authorization through existing trusted ingress, with a scoped local decision API over TCP or Unix socket. Defer NFQUEUE inspection, traffic-mirror response, transparent proxies, TLS termination and native database-protocol proxies; do not add them as default Agent dependencies.
- Keep network IP/CIDR subjects separate from requesters and verified protocol identities. Approval is not confirmation of enforcement.
- Preserve independent Grant lifetimes, permission union, deny precedence, IPv6 support, and single authoritative configuration source.
- Use npm and the committed lockfile for Web development. Keep business code out of `src/components/ui`; compose shared components separately.
- Keep UI loading, failure and freshness states explicit. Never substitute sample data for real authorization state.
- Add interfaces/packages when needed; do not preimplement the entire architecture during scaffolding.
- Check Go with `go test ./...`, `go vet ./...`, and `go build ./...`. Check Web with `npm ci`, `npm run typecheck`, `npm run lint`, and `npm run build` in `apps/web`.
- Do not add credentials, generated build output, or GPL Agent implementation to this Apache-2.0 repository.
