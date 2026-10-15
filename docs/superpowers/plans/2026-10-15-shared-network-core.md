# M1.1 Shared Network Core Implementation Plan

> English (default) | [简体中文](2026-10-15-shared-network-core.zh-CN.md)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan inline, task by task. Steps use checkbox (`- [ ]`) syntax. Preserve the user's selected execution method; one whole-branch independent review at the end.

**Goal:** Implement a reusable, deterministic Go Core for logical source-to-entry permission union, independent lifetime/revocation, explicit deny and explanations.

**Architecture:** A standard-library package `pkg/core` validates and copies normalized inputs into an immutable engine. Callers supply the clock and trusted logical entry identity. Controller/Agent integrations later consume the same semantics; this increment does not infer physical destination/Binding scope or enforcement status.

**Tech Stack:** Existing Go 1.27.0 module `github.com/PicketX/PicketX`, Apache-2.0; no new module dependencies.

**Spec:** [Approved M1.1 design](../specs/2026-10-15-shared-network-core-design.md).

**Status:** Approved and executed; logical Core local acceptance and independent review complete, awaiting implementation PR integration. Planning design baseline is `ec23f7467d66ee9fc5655e2ed98c6e3ba89e019c`; execution started from merged plan commit `e64ccd36f0ddd7362e551dcfed593941df34e862`.

## Global Constraints

- Core uses only the Go standard library; no HTTP/ORM/Casbin/Linux imports, goroutines, sockets, database or `time.Now`.
- Network-only logical queries; default endpoint type `network`, action `connect`; unsupported requirements are errors, never discarded.
- Source normalization: plain IPv4 /32, IPv6 /128, explicit CIDR masked; reject malformed, hostname, zoned or IPv4-mapped input before masking.
- Typed prefixes must be valid/canonical; all matching source contributions participate, never longest-prefix-only.
- Whole, named or nonempty label-selected entries; no implicit Resource/Agent/global ALL selection, label inheritance, or invalid-empty-to-omitted conversion.
- Explicit bounded/permanent lifetime; zero mode invalid; activation inclusive, expiry exclusive; caller-supplied nonzero clock; absolute deadlines never renewed by compile/restart.
- Validate the whole candidate, including inactive/revoked data; preserve immutable ownership and deterministic sorted unique explanations.
- Keep each contribution's own source/deadline; `NextChange` means reevaluate, not authorization extension or confirmed apply.
- No wire schema, UID persistence/lifecycle, broad Resource selectors, Binding/physical planner, transport, approval/storage/UI or renewal integration. Do not copy GPL Agent code into this Apache package.

## Review Focus

- Presence-sensitive empty values: empty selection/actions/Exists values must fail rather than broaden access; Tasks 1–2.
- Mapped/zoned sources and cross-family overlap: reject noncanonical forms before masking and preserve family boundaries; Tasks 1–2.
- Inactive malformed contributions: expired/revoked/selector-unmatched data must still invalidate a candidate; Task 2.
- Caller mutation and returned-result aliasing: compiled policy and later decisions must remain unchanged; Tasks 2–3.
- Future activation, exact expiry and replay: preserve per-entry deadlines and correct earliest relevant change; Tasks 1–3.

## File map and dependency flow

| Files | Responsibility |
| --- | --- |
| `pkg/core/source.go`, `source_test.go` | Canonical network source parsing/validation |
| `pkg/core/selector.go`, `selector_test.go` | Validated label requirements and matching |
| `pkg/core/validity.go`, `validity_test.go` | Explicit lifetime modes and time boundaries |
| `pkg/core/selection.go`, `selection_test.go` | Network Endpoint normalization and whole/names/labels selection |
| `pkg/core/network_types.go`, `network.go`, `network_test.go` | Snapshot/query/result types, complete validation and immutable decisions |
| `pkg/core/conformance_test.go`, `example_test.go`, `doc.go` | Cross-case/literal Gitea conformance, public example and integration boundaries |
| `.github/workflows/ci.yml`, `README.md`, `README.zh-CN.md`, `docs/README.md`, `docs/README.zh-CN.md` | Include `pkg` in format checks and record actual implementation coverage |

Task 2 consumes Task 1 primitives; Task 3 consumes the engine and primitives. The existing Echo liveness handler and module dependencies remain unchanged.

---

### Task 1: Canonical source, selection and lifetime primitives

**Files:** Create the four primitive Go files and corresponding tests in the file map. Include `// SPDX-License-Identifier: Apache-2.0` in new Go source/tests.

**Interfaces:**
- `ParseSource(text string) (netip.Prefix, error)`; `ValidateSource(prefix netip.Prefix) error`. Parse an explicitly entered CIDR before masking; reject mapped/zoned forms first. No whitespace/hostname fallback. Direct typed prefixes must already be canonical.
- `SelectorOperator string`: exactly `In`, `NotIn`, `Exists`, `DoesNotExist`; `Requirement{Key string, Operator SelectorOperator, Values []string}`; `LabelSelector{MatchLabels map[string]string, MatchExpressions []Requirement}`.
- `NormalizeSelector(LabelSelector) (LabelSelector, error)` copies maps/slices, rejects empty selector/key/unknown operator, requires nonempty values for In/NotIn, requires **nil** values for Exists/DoesNotExist. Duplicate set values may be deduplicated. Equality distinguishes an existing empty value from a missing key.
- `(LabelSelector).Matches(labels map[string]string) bool` evaluates validated AND requirements, fails closed on invalid selector; no mutation or inheritance.
- `LifetimeMode uint8`: `LifetimeBounded=1`, `LifetimePermanent=2`; `Validity{Mode LifetimeMode, NotBefore, ExpiresAt time.Time}`; `NormalizeValidity(Validity) (Validity, error)` returns UTC instants. Zero mode fails; bounded requires end; permanent forbids end; supplied start must precede bounded end.
- `(Validity).ActiveAt(at time.Time) bool`; `(Validity).NextBoundary(at time.Time) time.Time`. Invalid validity/zero clock fail closed (false/zero); boundary is the earliest strictly future start/end; zero means none.
- `Endpoint{Name, Type string, Labels map[string]string}`; `NormalizeEndpoint(Endpoint) (Endpoint, error)` requires name, defaults empty type to `network`, rejects other types, copies labels without inheritance.
- `SelectionMode uint8`: omitted=0, all=1, names=2, labels=3; `EndpointSelection{Mode SelectionMode, Names []string, Selector *LabelSelector}`.
- `NormalizeSelection(EndpointSelection) (EndpointSelection, error)`: only the fully zero selection normalizes to all; omitted/all with supplied names/selector is invalid. Names requires a nonempty list of nonempty names and nil selector; labels requires a valid non-nil selector and nil names. Explicit non-nil empty slices stay invalid; mixed modes fail.
- `SelectEndpointNames(endpoints []Endpoint, selection EndpointSelection) ([]string, error)` validates entries and selection, rejects duplicate/missing named entries, returns sorted unique matches; valid zero-match label/all selection returns nil, never all entries. An empty catalog is not an implicit wildcard.

- [x] **Write primitive tests first.** `TestParseSource`: `192.0.2.10`→`192.0.2.10/32`, `2001:db8::10`→`/128`, `192.0.2.10/24`→`192.0.2.0/24`; reject hostnames, `%eth0`, mapped address and mapped CIDR such as `::ffff:192.0.2.10/64`. `TestValidateSource`: invalid/unmasked/mapped typed prefixes fail.
- [x] **Write selector/selection assertions.** `TestSelectorOperators`: absent-key NotIn true; add Exists then false; In alternatives and AND; DoesNotExist; existing `access:""` versus missing key. `TestSelectorRejectsInvalidValues`: nil/empty selectors, empty keys, unknown operator, empty In/NotIn and non-nil empty Exists/DoesNotExist values fail. `TestEndpointSelection`: zero/explicit all equivalent, named stays fixed after adding an entry, selector follows labels, zero matches returns nil. `TestSelectionRejectsInvalidInputs`: empty/mixed modes, duplicate endpoint definitions, missing names and unsupported endpoint type fail.
- [x] **Write lifetime assertions.** `TestValidityBoundaries`: `start=2026-10-15T08:00:00Z`, `end=10:00Z`: before start false, at start true, end−1ns true, exact end false; permanent stays active after 12:00Z. `TestValidityRejectsInvalidModes`: zero/unknown mode, missing bounded end, end on permanent, end≤start and zero evaluation time cannot allow. `TestNextBoundary`: before start→start, at start→end, at end→zero; normalize equivalent timezones to UTC without shifting the instant.
- [x] **Run RED.** Add only type/signature scaffolding needed to compile, with absent behavior; run `go test ./pkg/core -run 'Test(ParseSource|ValidateSource|Selector|EndpointSelection|SelectionRejects|Validity|NextBoundary)' -count=1`. Expected: failed behavior assertions for canonicalization, presence checks and lifetime boundaries, not import/dependency errors.
- [x] **Implement the primitives.** Use net/netip and standard maps/slices/time; preserve presence before normalization and make copies where returned canonical objects otherwise alias inputs.
- [x] **Run GREEN.** `go test ./...`, `go vet ./...`, `go build ./...`, `git diff --check`; expected exit 0, all primitive cases pass.
- [x] **Commit.** `feat(core): add source selection and validity primitives`.

### Task 2: Immutable logical Grant/restriction decisions

**Files:** Create `pkg/core/network_types.go`, `network.go`, `network_test.go`. Consume Task 1 interfaces unchanged.

**Interfaces:**
- `Resource{Name string, Labels map[string]string, Endpoints []Endpoint}`; empty catalogs/resources with no entries are allowed, but duplicate/empty resource names and duplicate/empty entry names fail.
- `Scope{ResourceName string, Endpoints EndpointSelection, Actions []string}`. Nil Actions defaults to `connect`; non-nil empty or empty/unsupported values fail; repeated connect deduplicates. References must exist, even for inactive contributions. Resource labels never enter Endpoint label matching.
- `Grant{ID string, Source netip.Prefix, Scopes []Scope, Validity Validity, Revoked bool, SubjectType string, Conditions []string, Renewal string}`. `Deny` has the same fields as a distinct type. These are evolving normalized Go inputs, not authoring/wire fields.
- `SubjectType` empty/`source` is the supported network-source type; other values fail. Conditions must have length 0; any supplied condition fails as unsupported. Renewal empty/`disabled` is accepted, other modes fail. No condition/renewal interpreter is added.
- `Snapshot{Resources []Resource, Grants []Grant, Denies []Deny}`; IDs are nonempty and unique within each contribution type; one ID may exist independently in the Grant and Deny domains.
- `Compile(Snapshot) (*Engine, error)` validates everything before returning a non-nil immutable engine, makes owned copies/canonical resolved entries, uses no clock, and returns nil engine on invalid input. Removing a Grant does not remove the catalog. Invalid replacement does not mutate an already compiled engine.
- `Query{Source netip.Addr, ResourceName, EndpointName, Action string, At time.Time}`; empty Action defaults to connect; invalid/mapped/zoned source, unknown entry/action or zero time returns error.
- `DecisionReason string`: `allowed`, `no-active-grant`, `explicit-deny`; `Contribution{GrantID string, Source netip.Prefix, Validity Validity}`; `Decision{Allowed bool, Reason DecisionReason, GrantIDs, DenyIDs []string, Contributions []Contribution, NextChange time.Time}`.
- `(*Engine).Decide(Query) (Decision, error)` gathers all current non-revoked matching contributions, keeps matching allow explanations even when deny wins, returns sorted unique IDs and Contributions sorted by GrantID. No matches returns nil ID/contribution slices; no future boundary returns zero NextChange. Returned slices cannot expose engine-owned data. Nil/uninitialized engines return an error rather than panic/allow.
- NextChange is the minimum strictly future boundary among non-revoked matching source/scope/action Grants and Denies, including future activations; expired/no-match/unrelated/revoked contributions do not invent boundaries. Never use it as a new expiry or applied-status field.

- [x] **Write candidate validation tests first.** `TestCompileRejectsInvalidSnapshot`: duplicate names/IDs (within each domain; one Grant/Deny ID pair is valid), missing Resource/Endpoint, empty scopes/actions and malformed source/validity return nil engine plus error. `TestCompileValidatesInactiveContributions`: revoked, expired and unmatched-selector contributions with bad references/requirements still fail. `TestCompileRejectsUnsupportedRequirements`: subject user/service, type http, action read, condition present or renewal unique-only fail in Grants and Denies. `TestCompileDefaults`: omitted/explicit network, connect and disabled produce equivalent decisions; an empty catalog has no queryable entry; a valid nonempty label selector with no matches contributes no allow.
- [x] **Write decision/ownership tests.** `TestDecisionPermissionUnion`: source `192.0.2.10` matches Grants `a-host` `/32` and `b-prefix` `/24`; both IDs and original source/lifetime contributions present; another source only b-prefix; equivalent IPv6 case; no cross-family match. `TestDecisionDenyPrecedence`: active deny overrides both, future/expired/revoked deny does not, NotIn selector cannot cancel another Grant. `TestDecisionRevocation`: revoke a-host retains b-prefix; revoke both gives no-active-grant; permanent revoked Grant cannot allow. `TestDecisionNextChange`: future allow at08:30Z, deny at09:00Z, allow expiry10:00Z and deny expiry09:30Z; query08:00Z→08:30Z; at08:30Z→09:00Z; at09:00Z deny wins with NextChange09:30Z; at09:30Z allow resumes with NextChange10:00Z. `TestDecisionRejectsInvalidQuery` checks all query errors including nil/zero engine.
- [x] **Write mutation/determinism tests.** `TestCompileOwnsInputs` mutates original labels, endpoint/selection names, selector values, scopes, Grant ID/source/lifetime/revocation and Denies after compilation; old decisions remain unchanged. `TestDecisionOwnsResults` mutates returned IDs/contributions; a second result stays unchanged. `TestDecisionDeterministic` reorders equivalent catalogs/contributions/scopes/selector values and expects identical sorted explanations. `TestDecisionConcurrentRead` queries one engine from multiple goroutines and mutates only independently returned results; fixed results remain identical and race detection reports no conflicts. `TestInvalidReplacementPreservesEngine` rejects a bad new candidate and reevaluates the old engine at a later time so expiry still occurs.
- [x] **Run RED.** Create only compiling type/signature shells, run `go test ./pkg/core -run 'Test(Compile|Decision|InvalidReplacement)' -count=1`. Expected: permission/reason/provenance/validation assertions fail due to missing behavior; no import errors or nil-pointer test failures.
- [x] **Implement Compile/Decide.** Resolve exact Resource references and endpoint membership on the owned catalog; validate inactive data before filtering. Index by named entry if useful, iterate every matching source contribution, and calculate each boundary independently. No longest-prefix-only rule, global cache, goroutine or side effects.
- [x] **Run GREEN.** `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `git diff --check`; expected exit 0 and stable ownership/concurrency checks.
- [x] **Commit.** `feat(core): evaluate immutable network grant contributions`.

### Task 3: Gitea conformance, public example and actual coverage

**Files:** Create `pkg/core/conformance_test.go`, `example_test.go`, `doc.go`; update CI format check and English/Chinese READMEs/docs index. Consume Task 1–2 interfaces unchanged.

**Interfaces:**
- `ExampleCompile` demonstrates ParseSource→Compile→Decide for a logical Gitea HTTPS entry, with literal stable output; no HTTP/database/Agent dependencies.
- `doc.go` documents source/requester separation, trusted logical identity, explicit clock, immutable snapshot replacement, nil-vs-empty constructor inputs, unsupported requirement rejection, and evolving Go API/no wire schema guarantees.
- CI formatting command changes from `gofmt -l cmd internal` to `gofmt -l cmd internal pkg`; existing vet/race/build remain the acceptance commands.

- [x] **Write cross-case tests first.** `TestGiteaIndependentDeadlines`: catalog dev-gitea/https and dev-gitea/ssh; source192.0.2.10; HTTPS Grant ends10:00Z, SSH ends12:00Z. At09:00Z both allowed, at10:00Z only SSH, at12:00Z neither; repeat IPv6. Add another HTTPS prefix Grant ending11:00Z and require HTTPS remains until11:00Z without changing SSH's12:00Z deadline; at10:00Z the expired HTTPS Grant ID is absent.
- [x] **Write scoped update/conformance assertions.** `TestGiteaOverlapAndRevocation`: remove/revoke one host contribution while prefix remains; matching explicit deny wins for HTTPS only; SSH remains allowed. `TestCatalogSelectionEvolution`: append a git-admin entry/relabel entries and compile a replacement; whole expands, named stays fixed, selector follows endpoint labels; original engine remains unchanged. `TestEquivalentProviderInputs`: independently constructed normalized Controller/static-style inputs with omitted/explicit defaults and timezone-equivalent deadlines produce identical decisions. `TestRecompilePreservesDeadlines` compiles the same absolute snapshot twice, queries both engines before and at expiry, and requires identical deadlines/decisions with no fresh duration. No provider/decoder or physical kernel bridge is claimed.
- [x] **Run the new tests and example.** `go test ./pkg/core -run 'Test(Gitea|CatalogSelection|EquivalentProvider|RecompilePreserves)|ExampleCompile' -count=1`. Expected: existing engine passes correct invariants. For a new uncovered behavior, first observe a failed assertion before its minimal fix; do not invent failing work if Task 2 already implements the contract. Check that a longest-prefix-only, resource-wide latest-TTL or end-inclusive mutation would fail the literal cases.
- [x] **Add public guidance and format coverage.** Describe current implemented logical Core and remaining M1 planner/workflow tasks in both READMEs; keep M0 empirical scope and schemas under design. Do not represent logical allow as successful packet/application enforcement.
- [x] **Run complete acceptance.** `test -z "$(gofmt -l cmd internal pkg)"`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `git diff --check`; expected exit0. `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./pkg/core` must list only `github.com/PicketX/PicketX/pkg/core` as nonstandard; go.mod/go.sum stay unchanged. Existing Go CI must cover pkg/core; verify actual results, not just configured steps.
- [x] **Commit.** `test(core): verify Gitea logical authorization contract`.
- [x] **Final review.** Dispatch one fresh whole-branch reviewer through requesting-code-review; fix Important/Critical findings in one test-verified pass, record deferred minors, and publish the implementation PR with actual unit/race/build/dependency evidence. No merge without the user's integration decision.

## Literal assertion patterns

These excerpts belong to the named Task 1–3 tests. Test-only fixture construction remains in test files; it is not another public API.

```go
// TestParseSource
p, err := ParseSource("192.0.2.10/24")
if err != nil || p.String() != "192.0.2.0/24" { t.Fatalf("prefix=%v err=%v", p, err) }
// TestValidityBoundaries: v is normalized and end is the fixed deadline.
if !v.ActiveAt(end.Add(-time.Nanosecond)) || v.ActiveAt(end) { t.Fatal("exclusive expiry") }
// TestDecisionPermissionUnion: engine contains a-host /32 and b-prefix /24; q queries .10.
d, err := engine.Decide(q)
if err != nil || !d.Allowed || !reflect.DeepEqual(d.GrantIDs, []string{"a-host", "b-prefix"}) { t.Fatalf("decision=%+v err=%v", d, err) }
// TestGiteaIndependentDeadlines: q queries https, endHTTPS is 10:00Z, without the extra HTTPS prefix Grant.
q.At = endHTTPS
https, err := engine.Decide(q)
if err != nil || https.Allowed || https.Reason != "no-active-grant" { t.Fatalf("https=%+v err=%v", https, err) }
q.EndpointName = "ssh"
ssh, err := engine.Decide(q)
if err != nil || !ssh.Allowed { t.Fatalf("ssh=%+v err=%v", ssh, err) }
```

## Completion and next stage

Accept M1.1 only after all named tests and the public example execute, the whole Go suite/race/vet/build pass, package dependencies remain standard-library-only and final review findings are addressed. Record every deliberate interface/spec ruling in the execution ledger.

This plan delivers logical Core semantics only. The next focused design is physical Endpoint/Binding planning plus Core-to-M0 conformance; it must resolve destination/enforcement scope, ownership conflicts and lifecycle edits before Agent integration. Persistence/approval, reliable publication, transport/status and Gitea portal/CLI follow in later increments.

## Execution record

- Three implementation commits deliver primitives, immutable decisions and Gitea conformance. Nine primitive and fourteen engine tests first failed behavior assertions, then passed implementation. Five cross-case tests and the public example verified existing engine behavior; actual first-Grant-only and end-inclusive mutations failed the literal Gitea tests and were restored.
- All 29 Core tests/example and 3 existing Controller tests pass, with no skipped tests; the command package has no tests. Go unit/race/vet/build, formatting and diff checks pass; Web npm ci/typecheck/lint/build pass. Core's only nonstandard dependency is itself, and go.mod/go.sum are unchanged.
- Independent whole-branch review: 0 Critical and 0 Important findings; reviewer independently ran uncached race/vet/build/format/dependency checks. Remote [CI #38608827993](https://github.com/PicketX/PicketX/actions/runs/38608827993) succeeds on commit `03efd46722dcf245f354ff56e96a3a03d8ec2b29`: Go and Web jobs pass, with Go logs confirming pkg/core race execution. Subsequent evidence-only documentation changes do not alter runtime or test sources.
- Deferred Minor: Validity.NextBoundary can treat omitted NotBefore as a future zero boundary for nonzero evaluation times before year 1, hiding a bounded expiry. Normal operational clocks are unaffected. A future IsZero guard and historical-time regression can address this.
- Execution ruling: Task 3 final-review/publication items run at the whole-branch finish gate, following executing-plans; cost is process order only, with review still before publication.
- Scope rulings preserve approved boundaries: physical destination/Binding/kernel enforcement deferred, so logical allow cannot establish packet admission; persistence/approval/reliable publication/apply status deferred, so no complete browser-to-Agent workflow; external schemas/decoding deferred, so no stable wire compatibility promise; large-scale capacity unclaimed, so deployment capacity requires measurement.
