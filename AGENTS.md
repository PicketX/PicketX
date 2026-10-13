# PicketX development

Read `docs/PRODUCT_DESIGN.md` and `docs/ARCHITECTURE.md` before changing domain behavior. English documentation is the default; keep corresponding Chinese documents in sync.

Read `docs/CONFIGURATION_MODEL.md` before implementing configuration schemas or bindings. Keep business workflows, standard desired-state objects and execution state separate; follow the documented omission/empty-value semantics and do not silently finalize its open decisions.

- This repository contains the Go control plane/core and `apps/web`. The Linux enforcement Agent belongs to PicketXD.
- Controller HTTP uses Echo v5. Keep framework types in HTTP adapters; business and shared configuration/core packages remain framework-independent.
- Controller persistence uses GORM. Keep ORM types in persistence implementations and use versioned migrations for deployed databases; drivers and migration tooling are not yet selected.
- Use go-playground/validator v10 for HTTP DTO fields; keep complex configuration invariants in shared Go validation functions and preserve omission/null/empty/zero semantics. Add the dependency when an implemented endpoint requires it.
- API documentation uses Swaggo v2 CLI pinned to v2.0.0-rc6 with `--v3.1`; introduce generation and CI contract checks with the first documented business endpoint. Keep ordinary Echo handlers; Swagger UI v5.33.2 is selected for self-hosted rendering. Code generation is not an initial requirement.
- Use explicit constructor injection and focused module assembly; no DI framework is selected. Keep dependency directions inward, avoid global service lookup, and centrally manage startup/readiness/failure cleanup/shutdown. Request identities and transactions must not live in shared service fields.
- Keep network IP/CIDR subjects separate from requesters and verified protocol identities. Approval is not confirmation of enforcement.
- Preserve independent Grant lifetimes, permission union, deny precedence, IPv6 support, and single authoritative configuration source.
- Use npm and the committed lockfile for Web development. Keep business code out of `src/components/ui`; compose shared components separately.
- Keep UI loading, failure and freshness states explicit. Never substitute sample data for real authorization state.
- Add interfaces/packages when needed; do not preimplement the entire architecture during scaffolding.
- Check Go with `go test ./...`, `go vet ./...`, and `go build ./...`. Check Web with `npm ci`, `npm run typecheck`, `npm run lint`, and `npm run build` in `apps/web`.
- Do not add credentials, generated build output, or GPL Agent implementation to this Apache-2.0 repository.
