# Basic-function delivery roadmap

> English (default) | [简体中文](BASIC_FUNCTION_ROADMAP.zh-CN.md)

Date: 2026-10-15. The user has approved continued delivery in several increments until usable Layer 4, Layer 7 and management functions exist. This is sequencing and acceptance guidance for the agreed product/architecture, not a new API specification or a claim of completed functionality. Preserve native inline execution with a fresh whole-branch review for each implemented slice; keep the existing design/plan review gates.

## Current position

| Slice | Verified state |
| --- | --- |
| M0 Linux backend validation | Merged in PicketXD; 17 real-kernel scenarios at the recorded tested revision |
| M1.1 shared logical Core | Merged; union, deny precedence, independent lifetimes and immutable snapshots |
| M1.2 physical planner | Merged; scoped per-Agent projections, conflict/retarget checks and withdrawals |
| M1.3 shared-to-kernel bridge | PicketXD PRs #2 and #3 merged; 17 M0 + 19 bridge required kernel scenarios pass; [execution links](superpowers/plans/2026-10-15-core-m0-bridge.md) |
| M2A static configuration | Strict YAML compiler/replacement and read-only `picketx config check`; PicketX PR #10 |
| M2B desired-state preparation | PicketXD PR #4 merged; durable reload/restart recovery, independent boundary scheduling; all 43 required kernel scenarios pass, including seven host INPUT cases; production enforcement remains pending |
| Controller / Web | Health endpoint and frontend scaffold; business workflow, usable management screens and production Agent pending |

## Delivery sequence

The rows below subdivide the existing M1 usable-workflow goal. Their labels are work packages, not commitments to stable schemas. Each gets a focused design/plan before implementation; later evidence may adjust task boundaries without reopening settled architecture.

| Order | Working deliverable | Acceptance evidence |
| --- | --- | --- |
| 1. M1.3 bridge | Shared plan -> confirmed owned nftables state -> precise cleanup in disposable fixtures | 17 existing + 19 bridge kernel cases, dual-stack family evidence, absolute deadlines and actual applied history |
| 2. Agent runtime foundation | A runnable `picketxd`, scoped runtime configuration, one authoritative provider, static complete-candidate loading, explicit capabilities/context, shutdown and durable ownership/LKG recovery | Real Linux static deployment; invalid reload leaves confirmed state; restart/provider identity cannot renew or inherit foreign permissions; static release artifact inspection/smoke test |
| 3. Controller access workflow | Versioned configuration/business APIs, SQLite persistence/migrations, OIDC session login, RBAC, requests, basic approval, Grant issue/revoke/expiry, authorization/audit records | API/database tests and a Gitea HTTPS/SSH independent-approval scenario; approved state remains distinct from applied state |
| 4. Controller-to-Agent synchronization | Token-authenticated gRPC with configured plaintext/server TLS, per-Agent desired/applied generations, reconnect/full resync and absolute-time scheduling | Delayed/reordered/repeated messages cannot restore revoked/expired access; disconnected/restarting Agent has explicit freshness/failure state; no dual authority |
| 5. Layer 7 usable integration | Shared request authorization semantics, scoped local TCP/Unix decision API and at least one actual trusted HTTP ingress adapter | Real HTTP/HTTPS requests allowed/denied by resource/host/method/path and verified identity when required, including reused connections; forged client headers fail; required network deny still wins |
| 6. Usable management console | Resource/endpoint and Binding editing, Agent health/applied status, access requests/approval, active Grants/revoke, audit/diagnostics; compose the existing React/shadcn stack | Browser walkthrough against live APIs, loading/error/empty/stale states, IPv6 display, clear decision reasons and an operator can finish the Gitea workflow without hand-editing policy files |
| 7. Basic release hardening | Integrated deployment, upgrade/backup/restore instructions, basic CLI/status/doctor, static and Controller operating modes, reproducible artifacts | Full browser -> approval -> Agent confirmation -> L4/L7 request -> expiry/revoke walkthrough, real Linux tests and inspected standalone release binaries |

Add relevant management screens alongside implemented APIs rather than postponing all usability work until the end. Agree user flows and repository-owned visual/component rules before frontend implementation; apply UI UX Pro Max and frontend-design without changing the selected stack. A static mockup or sample data does not satisfy the management milestone.

## What counts as basic completion

- L4: real TCP/UDP IP/CIDR admission, IPv4/IPv6, overlapping independent Grants, explicit deny precedence, hard deadlines, selected graceful/strict handling and precise ownership-scoped cleanup; HTTPS and SSH permissions remain independent.
- L7: actual HTTP/HTTPS authorization through trusted ingress, request-bound verified identity where required, meaningful resource/method/path decisions, bounded failures/caches and a protected scoped local decision interface. For resources requiring both layers, both must allow.
- Management: administrators can configure supported resources/deployments, inspect desired vs confirmed applied state and freshness, approve/reject requests, revoke permissions and trace decisions. Requesters can log in and understand access and expiry. Screens use live backend results.
- Operations: documented single-node Controller/SQLite plus Agent setup, persisted absolute deadlines, explicit provider identity, recovery/failure tests, basic authorization records, runnable static mode and verified release artifacts.

Keep the agreed first-release boundary: existing ingress owns TLS and forwarding. NFQUEUE inspection, mirror response, transparent proxying, native database-protocol proxies, WASM, HA and large-scale capacity commitments remain later work. No fixture result alone establishes production host-local/container support: each implemented topology must be explicitly scoped and tested.

## Reporting cadence

Each increment reports concrete code/PRs, tests actually run, observed kernel evidence where required, remaining gaps and the next deliverable. Continue in this project over subsequent turns; this instruction does not create an unattended background job or scheduled automation. Detailed written designs/plans retain their review gates. Review findings and failed validation are resolved before claiming that slice is complete.
