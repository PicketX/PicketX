# Management publication implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver authenticated, audited database-authoritative configuration management through the running Controller.

**Architecture:** Reuse the existing publication compiler/store and Agent service. Keep HTTP DTOs and trusted principal/RBAC integration separate; serialize publication and distribution at the runtime owner. Persist authority mode and transactional audit.

**Tech Stack:** Go1.27, Echo5.4.0, Koanf2.3.8, GORM1.31.2/official SQLite1.6.0+mattn1.14.52, Casbin v3/official GORM adapter, validator v10, pinned Swaggo2.0.0-rc6. Existing React stack unchanged.

**Spec:** docs/superpowers/specs/2026-10-18-management-publication-design.md

## Global Constraints

- Preserve a single authoritative source, absolute Grant deadlines, deny precedence, IPv4/IPv6 and Apache/GPL repository separation.
- Default file mode stays compatible; management mode never polls policy/facts files.
- SQLite single Controller only; no PostgreSQL/HA claim from transitive dependencies.
- API acceptance never claims Agent application. Independent review and required CI/main checks gate merge.

## Review Focus

- Source-mode mismatch after restart must not allow file polling to overwrite managed state.
- Concurrent/no-op requests with stale If-Match must fail without an audit or revision change.
- A multi-role principal's allow must never bypass a deny from another role.
- Credential and RBAC reload failures must fail closed without returning successful freshness.
- A post-commit distribution error must stop admission and recover the exact durable revision on restart.

## Context map

| Files | Responsibility |
| --- | --- |
| internal/publication/store.go, managed.go, *_test.go | source binding, versioned migration, CAS/raw snapshot/transaction audit |
| internal/managementaccess/*.go | private Token principal resolution and one Casbin adapter/enforcer lifecycle |
| internal/management/*.go | authenticated application-facing operations/interfaces, framework-independent |
| internal/controller/management.go, *_test.go | strict Echo DTOs, If-Match, errors, route documentation |
| internal/runtimeconfig/config.go, *_test.go | explicit mode, management credentials and RBAC reload/diagnostic settings |
| internal/controllerapp/{runtime,inputs,management}.go, *_test.go | construct adapters, serialize writes/poll, shutdown and TCP acceptance |
| scripts/generate-management-api.sh, docs/api/*, .github/workflows/ci.yml | pinned OpenAPI generation and contract checks |
| docs/AUTONOMOUS_DELIVERY.md and bilingual design/operator docs | lease, evidence, limits and handoff |

### Task 1: Durable managed publication

**Interfaces:** `OpenMode(path, authority, mode string) (*Store,error)`; `Snapshot() (Document,error)`; `AcceptManaged(expected uint64, actor,reason string, policy,facts []byte, preflight func(*configuration.Candidate,uint64) error) (bool,error)`; `Audit(after int64,limit int) ([]AuditEvent,error)`; `GORM() *gorm.DB` for assembly-only adapter reuse.

- [ ] Write mode binding/migration, stale-noop, concurrent writer and rollback/reopen tests in `internal/publication/managed_test.go`.
- [ ] Run `go test ./internal/publication` and observe missing managed behavior.
- [ ] Implement the interfaces, keeping original `Accept` file-only and migration/audit atomic.
- [ ] Run `go test -count=1 ./...`; expected all pass; commit.

### Task 2: Trusted management principals and Casbin

**Interfaces:** `managementaccess.New(db,settings,logger) (*Access,error)`; `Authenticate(rawToken string) (Principal,error)`; `Allowed(Principal,operation string) (bool,error)`; `Run(ctx)`; typed interface consumed by management HTTP, no role from client input.

- [ ] Write private-file/strict document/token/revocation, combined deny and periodic reload failure/recovery tests.
- [ ] Run focused tests and observe missing behavior, then implement official adapter/enforcer and credential resolution.
- [ ] Run `go test -count=1 ./...`; expected all pass; commit.

### Task 3: Running API and end-to-end publication

**Interfaces:** inject `management.Service` (publication read/write/audit/access) into the Echo adapter; runtime owns serialization and fatal durability/distribution notification.

- [ ] Add runtime mode validation and HTTP tests for 401/403, malformed DTO, limits, 428/412, keyset audit and disabled file-mode routes.
- [ ] Run tests RED; implement typed runtime configuration and thin HTTP/application adapters.
- [ ] Add real TCP test: empty managed boot, API publish, authenticated gRPC nonempty snapshot, concurrent stale writer, credential revocation, no file overwrite, restart exact revision, and post-commit failure recovery.
- [ ] Run real integration test RED then implement lifecycle changes and verify GREEN.
- [ ] Add pinned OpenAPI generation and contract checks, operator examples and synchronized Chinese design summary.
- [ ] Run complete uncached Go/race/vet/build/static and Web lockfile checks; commit.

### Task 4: Independent review and integration

- [ ] Request one independent whole-branch review from base0b2c5e9 to final head, including the spec and Review Focus above.
- [ ] Reproduce blocking findings RED→GREEN, record rulings/deferred minors, run complete suite.
- [ ] Verify local/remote tree equality, push using expected remote SHA, create PR and inspect required raw CI evidence.
- [ ] Merge only expected reviewed head after checks; verify main CI, persist exact evidence and release lease.
