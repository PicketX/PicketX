# PicketX development

Read `docs/PRODUCT_DESIGN.md` and `docs/ARCHITECTURE.md` before changing domain behavior. English documentation is the default; keep corresponding Chinese documents in sync.

- This repository contains the Go control plane/core and `apps/web`. The Linux enforcement Agent belongs to PicketXD.
- Keep network IP/CIDR subjects separate from requesters and verified protocol identities. Approval is not confirmation of enforcement.
- Preserve independent Grant lifetimes, permission union, deny precedence, IPv6 support, and single authoritative configuration source.
- Use npm and the committed lockfile for Web development. Keep business code out of `src/components/ui`; compose shared components separately.
- Keep UI loading, failure and freshness states explicit. Never substitute sample data for real authorization state.
- Add interfaces/packages when needed; do not preimplement the entire architecture during scaffolding.
- Check Go with `go test ./...`, `go vet ./...`, and `go build ./...`. Check Web with `npm ci`, `npm run typecheck`, `npm run lint`, and `npm run build` in `apps/web`.
- Do not add credentials, generated build output, or GPL Agent implementation to this Apache-2.0 repository.
