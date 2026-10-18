# Agent stream foundation implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:executing-plans to implement task-by-task. Execute inline under the existing authorization to continue, push and merge; obtain one fresh whole-branch review.

**Goal:** A tested Controller-side scoped snapshot stream with revocable Agent credentials.
**Architecture:** Public generated protobuf DTOs map immutable shared compiled inventory. An injected in-memory Controller service owns credentials, latest desired state and untrusted Agent reports; gRPC owns transport cancellation. This slice opens no production listener and does not modify the privileged Agent.
**Tech Stack:** Go 1.27, grpc-go v1.84.0, protobuf v1.36.11; protoc 31.1 / protoc-gen-go v1.36.11 / protoc-gen-go-grpc v1.6.1.
**Spec:** `docs/superpowers/specs/2026-10-18-agent-stream-design.md` (Chinese counterpart alongside).

## Global Constraints
- Fixed authority; Agent identity and scope derive solely from one revocable Bearer Token.
- Full node inventory retains independent absolute deadlines/provenance and supports explicit empty withdrawal.
- Positive bounded memory/message/stream/time limits; no unbounded publication queues.
- Alpha protocol, no persistent revision/revocation guarantees, no production listener, no claim of kernel execution.
- Plaintext or normally verified server TLS; no mandatory mTLS or GPL imports.
- English/Chinese documentation stays synchronized.

## Context Map

### Files to Modify
| File | Purpose | Changes |
| --- | --- | --- |
| `proto/picketx/agent/v1alpha1/sync.proto` | Public wire schema | Hello, typed full inventory, reports |
| `pkg/protocol/agent/v1alpha1/*.pb.go` | Generated Go binding | Pinned generator output |
| `pkg/agentsnapshot/snapshot.go` | Compiled per-Agent projection | Typed lossless mapping and deterministic digest |
| `internal/agentsync/{service,credentials,stream}.go` | Controller transport | Injected service, atomic credential/publication changes, bidi lifecycle |
| `scripts/generate-agent-protocol.sh` | Reproduction | Reject wrong tool versions, deterministic regeneration |
| `go.mod`, `go.sum` | Dependency boundary | Pin official gRPC/protobuf |
| `.github/workflows/ci.yml` | CI | Verbose uncached tests and generation check |
| `docs/AGENT_STREAM.md`, Chinese counterpart | Embedding/limitations | Runnable tests and integration requirements |
| `README.md`, Chinese counterpart, `docs/DELIVERY_STATUS*` | Current state | Report only this delivered foundation |

### Dependencies
| File | Relationship |
| --- | --- |
| `pkg/configuration/candidate.go` | Immutable Candidate.Input/Plan supply authoritative facts |
| `pkg/networkplan/plan.go`, `types.go` | Inventory retains future/expired/revoked contributions |
| `pkg/core/network_types.go` | Grant and deny identity/lifetime domains |
| `cmd/picketx-controller/main.go` | Existing liveness-only entry point; unchanged |

### Tests and reference patterns
| File | Coverage/pattern |
| --- | --- |
| `pkg/agentsnapshot/snapshot_test.go` | Isolation, exact inventory, empty deletion, deterministic/copy-safe digest |
| `internal/agentsync/service_test.go` | Real gRPC protobuf streams via bufconn, TLS, lifecycle and abuse bounds |
| `pkg/configuration/compile_test.go` | Candidate validation/replace patterns |
| `pkg/networkplan/plan_test.go` | Independent future/expired contributions and protection |

Risk review: additive alpha API; no database migrations; no operator configuration change. Context map self-reviewed before implementation: boundaries preserve existing domain/compiler APIs.

## Review Focus
1. A credential changes while a stream is blocked sending: the old connection must terminate without retaining a slot. Test in Task 2.
2. A stale receive races stream supersession: the old stream must never replace the successor's report. Test in Task 2.
3. An Agent disappears from the next complete candidate: authorized peers receive empty inventory. Test in Task 2.
4. Credential scope is added after publication: its initial stream must still get current complete intent. Test in Task 2.
5. Future or expired contributions and nanosecond deadlines: publication must preserve them instead of extending/dropping lifetimes. Test in Task 1.

### Task 1: Typed scoped snapshots
**Files:** proto, generated bindings, generation script, `pkg/agentsnapshot/`, module pins.
**Interfaces:** consumes `*configuration.Candidate`, `Plan.Inventory(string)`; produces `Build(c *configuration.Candidate, agent string, revision uint64) (*agentv1.Snapshot, error)` and `Verify(*agentv1.Snapshot) error`. Unknown/removed Agent gets empty inventory under explicit server scope; invalid candidate, revision or authority fails.
- [ ] Write tests for two-Agent isolation, exact target/context/capability/provenance mapping, independent bounded/permanent/future/expired/revoked contributions, ns timestamps, deterministic digest, tampering, caller mutation isolation, invalid inputs and explicit empty scope.
- [ ] Run `go test ./pkg/agentsnapshot`; Expected: fails because Build/Verify are absent.
- [ ] Define protobuf messages, generate pinned code and implement mapping/digest. `Verify` validates envelope/digest; domain validity remains compiled-candidate/Agent validation responsibility.
- [ ] Run `go test ./pkg/agentsnapshot` and `go test ./...`; Expected: all pass.
- [ ] Commit `feat(protocol): add scoped complete Agent snapshots`.

### Task 2: Authenticated cancellable stream service
**Files:** `internal/agentsync/`, CI, bilingual integration/status docs.
**Interfaces:** consumes Task 1 Build/Verify; produces `New(authority string, limits Limits) (*Service,error)`, `ReplaceCredentials([]Credential) error`, `Publish(*configuration.Candidate,uint64) error`, `Report(agent string) (Observation,bool)`, `Close()`, and generated `AgentSyncServer.Connect`. `Credential` has ID, AgentRef, SHA256; `Observation` has copied report and server receive time.
- [ ] Write real gRPC tests (no fake stream) for auth rejection, independent identities, Hello/version/message errors, complete publication/reconnect/empty removal, atomic invalid revision/oversize/credentials, revoked/rotated/rebound credentials, duplicate stream, old report races, copy isolation, per-process/per-stream bounds, Hello/send timeouts, Close and verified TLS (also wrong CA/name rejection). Pin all five review-focus conditions.
- [ ] Run `go test ./internal/agentsync`; Expected: missing Service functionality fails.
- [ ] Implement synchronized admission/publication/report acceptance, one latest update signal per stream, fixed receive and send workers, terminal cancellation channels and explicit limits. Generic safe gRPC errors only; reports remain Agent assertions.
- [ ] Run focused tests then `go test -count=1 -race ./...`, `go vet ./...`, `go build ./...`, `scripts/check-static-cli.sh`; Expected: pass. Run Web `npm ci`, typecheck/lint/build; Expected: pass.
- [ ] Reproduce generated output and check clean diff; update bilingual docs, CI generation check and actual test evidence. Commit `feat(controller): add revocable Agent snapshot streams`.

## Self-review
Every spec clause maps to Task 1 or 2 and the docs acceptance step. Build's typed Snapshot is consumed unchanged by Service. Review-focus tests explicitly assigned. No promise of durable publication, global quotas, runtime listener or Agent enforcement was introduced.
