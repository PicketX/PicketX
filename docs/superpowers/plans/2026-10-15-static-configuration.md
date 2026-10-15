# Static configuration implementation plan

> For agentic workers: use superpowers:executing-plans; the user requested continuous implementation and pushing without routine confirmation gates.

**Goal:** Check complete source-scoped network configuration through existing Core/planner semantics.
**Architecture:** Strict alpha DTO decoder → complete semantic translation → immutable planner Candidate; independent Cobra diagnostic command.
**Tech:** Go 1.27, go.yaml.in/yaml/v3 v3.0.5, Cobra v1.10.2.
**Spec:** [M2A](../specs/2026-10-15-static-configuration-design.md).

## Constraints and context map

Keep Apache/GPL separation; unknown and null fields fail; IP families and absolute deadlines are equal; no kernel or transport effects. `pkg/core` and `pkg/networkplan` supply all permission and replacement semantics unchanged. No migrations, UI changes or process settings loader.

| Files | Role |
| --- | --- |
| pkg/configuration/types.go, decode.go | Private wire DTOs, bounded strict YAML and owned Bundle |
| pkg/configuration/compile.go, candidate.go | Translate pinned references, compile/replace, independent input copies |
| pkg/configuration/*_test.go | Literal policy, rejection, lifetime, replacement and ownership coverage |
| internal/cli/config.go, cmd/picketx/main.go | Cobra configuration-check command, strict diagnostic facts |
| internal/cli/config_test.go | Real command success/failure and bounded reads |
| examples/static/* | Runnable Gitea policy and diagnostic runtime facts |
| README*, docs/CONFIGURATION_MODEL*, docs/README* | Current status and alpha entry point |
| go.mod, go.sum, THIRD_PARTY_NOTICES.md | Pinned dependencies and notices |

## Review focus

1. Malformed restriction silently broadens scope: reject null/empty/type errors recursively.
2. Same display name gets recreated: require pinned Resource UID and preserve replacement guards.
3. CLI failures look successful: emit no success object before full decoding/compilation/projection.
4. Caller mutates accepted input/output: return owned copies and immutable plan only.
5. Empty/truncated publication removes protection accidentally: require explicit empty object.

### Task 1: Decoder and policy representation

Interfaces: `Decode([]byte) (*Bundle,error)`; Bundle holds private normalized DTOs and deterministic hash. Write failing tests for literal policy, unknown/duplicate/null/empty fields, aliases, size/depth/object limits and order/default hash. Run `go test ./pkg/configuration`, observe missing behavior. Implement strict node validation and typed JSON decode; pin YAML dependency. Run package and full Go suite; commit.

### Task 2: Complete planner candidate

Interfaces: `Facts{AuthorityID string; Agents []networkplan.Agent; Contexts []networkplan.ExecutionContext}`; `Compile(*Bundle,Facts) (*Candidate,error)`; `Replace(*Candidate,*Bundle,Facts) (*Candidate,[]networkplan.Withdrawal,error)`; Candidate `Plan()`, `Input()`, `Hash()`, `Objects()`. Write failing tests for dual-stack explicit-time projection, UID pinning/recreation, independent validity/deny/future starts, replacement retargeting, UID kind reuse, full-candidate rejection, empty withdrawals and immutable copies. Implement via core/networkplan, not duplicate permission decisions. Run package/full suite; commit.

### Task 3: Runnable command and integration

Interfaces: `cli.New(out,errOut io.Writer) *cobra.Command`; `picketx config check` requires single --file and --facts, optional --at. Tests use actual files/planner, assert JSON validation state and no success on errors, strict/trailing/oversize facts, expired replay and required flags. Watch fail, implement with bounded reads and Cobra; add examples, bilingual status/schema guidance, dependency notices. Run test/race/vet/build, inspect/smoke-test static Linux executable and examples. Commit; dispatch one fresh whole-branch reviewer, fix important findings with regression tests, publish branch/PR and inspect CI.

## Execution ledger

- Baseline Go suite passes on 2026-10-15. Feature branch codex/m2-static-configuration in an isolated task checkout; clean main retained in git.
- Ruling: follow the user's standing continuous-delivery instruction instead of repetitive design/plan/worktree permission handoffs. New schema decisions are explicit above and reviewable in the PR.
- Ruling: this first entry point consumes a single atomic publication file; full directory generation/LKG/provider lifecycle remains M2B, avoiding a false multi-file transaction guarantee.

- Task 1: complete, 6c549dc. Strict decoder RED (missing Decode) → GREEN; complete Go suite passes.
- Task 2: complete, f6774e9. Literal dual-stack, deadlines/deny, all-candidate errors, pinned references, retargeting, empty withdrawals and ownership tests pass.
- Task 3: CLI RED (missing New) → GREEN. Additional RED→GREEN regressions reject Go JSON case aliases while retaining case-sensitive label maps. A true 4097-distinct-object test verifies the object bound rather than short-circuiting on duplicate identities.
- Ruling: policy JSON fields require exact spelling because encoding/json otherwise accepts case aliases; diagnostic Go facts reject aliases of one struct field while preserving distinct label keys. No change to Core or kernel semantics.
- Review fix: one Important Unicode JSON field alias bypass reproduced RED with Protocolſ, then fixed by exact recursive fact struct keys; case-sensitive label maps remain valid. Manual static-binary smoke test also found missing empty known-agent projections; regression RED→GREEN and CLI enumeration fixed without changing planner AgentRefs semantics.
- Task 3: complete. Full Go suite/race/vet/build pass; npm ci/typecheck/lint/build pass (only environment npm http-proxy warning). Static Linux amd64 CLI inspected with file/readelf: no interpreter/shared-library dependency. Actual examples: 09:00Z=2 allow/2 protected, 10:00Z=1/2, 12:00Z=0/2; explicit empty=known Agent/0 protected. Added the same build-and-smoke script to CI.
- Independent whole-branch review: one Important Unicode alias finding, reproduced and fixed as above; no other actionable findings. Final fix race/vet/build/full-suite checks pass. Native inline execution complete; remote PR/CI recorded in PR metadata to avoid self-referential evidence commits.
