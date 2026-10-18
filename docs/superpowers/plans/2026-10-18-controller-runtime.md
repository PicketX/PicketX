# Running Controller implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:executing-plans inline task-by-task. User authorized automatic development and merge; use one fresh whole-branch review.

**Goal:** Deployable Controller process with strict runtime config, durable scoped publication and finite lifecycle.
**Architecture:** Typed Koanf loading at assembly; GORM/SQLite authoritative accepted record; existing agentsync service preflight/distribution; injected listeners and cancellable poll owner.
**Tech Stack:** existing Go1.27/Echo5/gRPC; Koanf2.3.8/confmap1.0.0; GORM1.31.2/SQLite1.6.0/mattn1.14.52.
**Spec:** `docs/superpowers/specs/2026-10-18-controller-runtime-design.md`.

## Global Constraints
- Explicit flags > present environment > file > defaults; reject unknown/null/duplicate fields.
- Exactly one fixed authority, positive durable CAS revision, complete compile/Replace before commit and commit before distribution.
- Runtime serializes credential/policy mutation; bad policy must not prevent valid credential revocation.
- Optional normally configured server TLS, no mTLS requirement; no raw credential/SQL logging.
- Finite shutdown/rollback, private SQLite state and inspected runnable fully static Linux artifact.
- No unauthenticated management API, no claim of remote kernel/L7/management completion yet.

## Context Map
| File | Purpose/change |
| --- | --- |
| `internal/runtimeconfig/config.go`, tests | Strict typed Koanf defaults/file/env/flag boundary |
| `pkg/configuration/facts.go`, `internal/strictjson/`, tests; `internal/cli/input.go` | Reuse exact strict facts parser; CLI wrapper preserved |
| `internal/publication/store.go`, tests | Explicit SQLite migration, accepted record integrity/CAS/Replace/recovery |
| `internal/agentsync/service.go`, tests | Nonmutating publication preflight using same validation as Publish |
| `internal/controllerapp/{runtime,inputs}.go`, tests | Serial poll owner, credential parser, central HTTP/gRPC/listener lifecycle |
| `internal/controller/http.go`, tests | Optional generic readiness probe |
| `cmd/picketx-controller/main.go` | Thin flags/signals/config/Run assembly |
| `scripts/check-static-controller.sh`, CI | Actual fully static SQLite Controller launch/stop smoke |
| `examples/controller/`, bilingual runtime/checkpoint docs and README | Runnable setup and precise delivery status |

Dependencies: configuration Candidate/Input/Objects/Replace and networkplan constraints; agentsync scoped stream; existing Controller liveness and CLI tests. No Agent GPL code changes. Risk: additive process config, new explicit SQLite migration; no existing deployed Controller DB schema to migrate.

## Review Focus
1. Bad policy coincides with Token revocation; apply valid credential replacement first (Task2 test).
2. SQLite durable commit succeeds but distribution cannot finish; stop rather than advertise success, recover same revision (Task2 injected failure test).
3. Concurrent DB writer changes revision; CAS must reject stale commit (Task1 test).
4. Existing database contains authority mismatch/corrupt envelope/newer migration; startup must fail without overwrite (Task1 tests).
5. A later listener fails or context cancels with open RPCs; all listeners/store/workers must close finitely (Task2 tests).

### Task 1: Strict configuration and durable publication
**Interfaces:** `runtimeconfig.Load(path string, env map[string]string, overrides map[string]string) (Config,error)`; `publication.Open(path,authority string) (*Store,error)`; `Store.Current() (*configuration.Candidate,uint64)`; `Store.Accept(policy,facts []byte, preflight func(*configuration.Candidate,uint64) error) (bool,error)`; `Store.Close() error`; `agentsync.CheckPublication(c,revision) error`.
- [ ] Write failing tests for exact config precedence/invalid input/defaults, strict reusable facts decoding, SQLite migration/reopen/no-op/invalid replacement/authority/integrity/CAS/overflow/private files, preflight failure leaving DB unchanged.
- [ ] Run affected package tests; Expected missing functionality fails. Extract decoder, implement strict loading and selected SQLite adapter/migration; use same agentsync validation for CheckPublication/Publish.
- [ ] Run `go test ./...`; Expected all pass. Commit `feat(controller): persist validated desired publications`.

### Task 2: Process lifecycle and operational evidence
**Interfaces:** consumes Task1; produces `controllerapp.Run(ctx context.Context,c runtimeconfig.Config,logger *slog.Logger,d Dependencies) error` with injected `Listen func(string,string)(net.Listener,error)`; readiness callback only exposes bool. Add strict bounded credential file decoding into agentsync.Credential.
- [ ] Write failing real TCP tests for snapshot delivery/restart monotonicity/no renewal, bad source retaining LKG/readiness false, credential revocation during bad policy, newly credentialed nonempty scope, listener failure rollback, cancellation/open RPC shutdown, TLS file errors and postcommit distribution failure.
- [ ] Run affected tests; Expected missing Run behavior fails. Implement serial initial restore/load/tick and central lifecycle, preserve CLI default behavior and add flags/environment support.
- [ ] Run full uncached Go/race/vet/build, Web lockfile checks, pinned generation and static artifact smoke. Expected all pass; add same smoke to CI and bilingual setup/acceptance docs.
- [ ] Commit `feat(controller): run durable Agent publication service`; get fresh whole-branch review, fix blockers with RED→GREEN, publish checked tree, merge expected head after green CI and verify main CI. Update durable delivery checkpoint and release lease.

## Self-review
All spec clauses map to Task1/2; mutation preflight and owner serialization are explicit interfaces. Five additional failure classes have assigned tests. Scope remains the running publisher dependency of the final management/L7 goal, not a replacement for those outcomes.
