# M1.2 Physical Network Planner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking. The user has selected native inline execution and one fresh whole-branch review at the end; preserve that choice.

> English (default) | [简体中文](2026-10-15-physical-network-planner.zh-CN.md)

**Goal:** Compile logical authorization and explicit Bindings into immutable, scoped per-Agent physical intent with safe complete replacements.

**Architecture:** Core owns permission resolution; `pkg/networkplan` owns physical normalization, placement, conflicts and transition inventory. Both are pure Go libraries with explicit time. The separate PicketXD M0 bridge remains M1.3.

**Tech Stack:** Go 1.27.0, standard library (`net/netip`, `time`, sorting/copy helpers), existing `pkg/core`; go.mod/go.sum unchanged.

**Spec:** [Approved M1.2 design](../specs/2026-10-15-physical-network-planner-design.md), approved and merged in [PR #5](https://github.com/PicketX/PicketX/pull/5).

**Status:** Approved in PR #6 and executed from merged plan commit `dc7158ad8583d3cfafe95c94bdef8d708bf31118`. All five tasks are locally verified; whole-branch review follows before publication. Planning baseline is `315e0dd5617eed6cfc5055322d23b56378179cd2`. Task 1 fixes the M1.1 historical-clock regression.

## Global constraints

1. Apache-2.0 shared code; no GPL Agent implementation copied or imported.
2. Standard-library-only planner plus Core; no product dependencies, HTTP/database/Casbin/Linux imports, goroutines, wall clock, transport, manifest decoder, persistent lifecycle service or UI.
3. Explicit TCP or UDP and one destination port in 1–65535; no protocol default. IPv4/IPv6 equal; no mapped/zoned addresses, DNS, port ranges, NAT64, implicit alternate-port mapping or post-DNAT matching.
4. Whole-candidate validation, immutable owned state, deterministic results, independent absolute lifetimes; no deadline renewal, Controller/Static merge or partial replacement.
5. Explicit trusted context, original-destination contract and graceful/strict requirement survive into output. Previous desired inventory is not installation evidence; no packet, host/container or kernel-cleanup claim in M1.2.

## Review focus

| Input/failure mode | Expected behavior | Owning test |
| --- | --- | --- |
| Known empty Resource vs missing Resource | Empty selection diagnostic vs error, never wildcard | Task 1 `TestNetworkViewResolvedMembership`; Task 3 `TestBindingNoMatchesAndOffline` |
| Historical nonzero clock before AD 1 with omitted start | Real expiry remains the next boundary | Task 1 `TestNextBoundaryOmittedStartHistoricalClock` |
| Raw definition change hidden by current context boundary | Reject retarget with retained typed contribution IDs | Task 4 `TestRetargetRetainedContributions` |
| Removed/offline Agent or vanished final Binding | Keep desired offline targets; retain full former target after removal | Task 3 `TestBindingNoMatchesAndOffline`; Task 4 `TestWithdrawalInventory` |
| Expired allow may still own established graceful flows | Preserve former desired membership; never claim it proves actual installed history | Task 4 `TestHistoricalFormerInventory`; Task 5 documentation |

## File and dependency map

| Task | Files | Responsibility |
| --- | --- | --- |
| 1 | `pkg/core/network.go`, `network_types.go`, `network_view_test.go`, `validity.go`, `validity_test.go` | Owned resolved catalog and historical-clock regression |
| 2 | `pkg/networkplan/types.go`, `network.go`, `context.go`, `network_test.go`, `context_test.go` | Physical primitives and authoritative context limits |
| 3 | `pkg/networkplan/binding.go`, `conflict.go`, `plan.go`, `binding_test.go`, `conflict_test.go`, `plan_test.go`; extend `types.go` | Placement, immutable inventory and explicit-time projection |
| 4 | `pkg/networkplan/transition.go`, `transition_test.go`; extend `types.go` | Complete replacement guards and former-target inventory |
| 5 | `pkg/networkplan/conformance_test.go`, `example_test.go`, `doc.go`; root `README.md`, `README.zh-CN.md`, `docs/README.md`, `docs/README.zh-CN.md` | Public usage, integration cases and accurate milestone status |

All paths in rows 1–4 without a directory inherit that row's package directory. Task 1 modifies existing Core files and creates `network_view_test.go`; Task 2 creates `pkg/networkplan`; later tasks extend that package. No generated schema, new module or Agent file is part of this plan.

Execution order: 1 → 2 → 3 → 4 → 5. Each task has one red/green cycle and a focused commit; no per-task reviewer dispatch. After all tasks, request one fresh whole-branch review, address findings and reverify before integration. At execution start inspect AGENTS.md/worktree state and preserve unrelated changes.

### Task 1: Owned Core resolution view and clock prerequisite

**Interfaces consumed:** existing `Compile(Snapshot) (*Engine, error)`, `(*Engine).Decide(Query) (Decision, error)`, `Validity.NextBoundary(time.Time) time.Time`.

**Interfaces produced** in `network_types.go` / `network.go` (all names in package `core`):

```go
type ResolvedContribution struct {
    ID string
    Source netip.Prefix
    Validity Validity
    Revoked bool
}
type ResolvedEntry struct {
    Endpoint Endpoint
    Grants, Denies []ResolvedContribution
}
type ResolvedResource struct {
    Name string
    Labels map[string]string
    Entries []ResolvedEntry
}
type NetworkView struct { Resources []ResolvedResource }
func (e *Engine) NetworkView() (NetworkView, error)
```

Retain an owned normalized catalog in Engine, including Resource labels and empty Resources. `NetworkView` resolves no new policy and uses no time filter: include future, expired and revoked members, separating Grant/Deny domains. Sort Resources/entries by name and memberships by ID. Deep-copy all returned labels and slices. Empty compiled snapshots are valid; nil/zero Engine errors. Existing `Compile`/`Decide` semantics remain unchanged.

Fix only the recorded `NextBoundary` omission bug in `validity.go`: an omitted zero `NotBefore` is not a future activation even when the supplied nonzero time predates AD 1. Do not introduce a new clock range or alter lifetime rules.

- [x] **1. Write the named failing tests** in the task's test files. Assertion excerpts below use local fixture variables; build those fixtures in the owning test package. Do not reuse another package's private test helpers.

```go
// TestNetworkViewResolvedMembership: "empty" remains a resource with zero entries;
// "gitea" has sorted "https", "ssh"; duplicate scopes produce one membership.
if got.Resources[0].Name != "empty" || len(got.Resources[0].Entries) != 0 { t.Fatal(got) }
// Same string ID "shared" is permitted once in each independent domain.
if grants[0].ID != "shared" || denies[0].ID != "shared" { t.Fatal(grants, denies) }
// TestNetworkViewIncludesInactive: future, expired and revoked permanent retained.
if len(entry.Grants) != 3 || !revoked.Revoked { t.Fatal(entry) }
// TestNetworkViewOwnsData: mutate input and returned labels/slices/contributions;
// a fresh view and Decide must still equal their pre-mutation values.
if !reflect.DeepEqual(before, after) { t.Fatal(after) }
// TestNetworkViewDeterministicAndConcurrent: reorder all input lists and scopes;
// equal views, concurrent NetworkView/Decide, independent result mutation.
if !reflect.DeepEqual(a, b) { t.Fatal(a, b) }
// TestNetworkViewEmptyAndInvalidEngine: compiled empty snapshot succeeds;
// nil and zero Engine return errors, not a panic or a valid empty view.
// TestNextBoundaryOmittedStartHistoricalClock:
at := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
end := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
v := Validity{Mode: LifetimeBounded, ExpiresAt: end}
if got := v.NextBoundary(at); !got.Equal(end) { t.Fatal(got) }
// Also assert explicit future NotBefore is returned and permanent/no-start is zero.
```

- [x] **2. Run the focused command** below and observe the expected behavior failure. For new APIs, add only declarations/minimal error-returning stubs first so red evidence exercises assertions rather than missing imports.

```bash
go test ./pkg/core -run 'Test(NetworkView|NextBoundaryOmittedStartHistoricalClock)' -count=1
```

- [x] **3. Implement the declared interfaces** in the mapped files, following the task's rules and the approved spec; no unrelated refactor.

- [x] **4. Re-run the focused command** and the indicated regression command; require PASS and exit 0.

```bash
go test ./pkg/core -run 'Test(NetworkView|NextBoundaryOmittedStartHistoricalClock)' -count=1
go test -race ./pkg/core -count=1
```

- [x] **5. Commit only this task's files** with the message below after reviewing `git diff --check` and the staged diff.

```bash
git commit -m "feat(core): expose owned resolved network view"
```

### Task 2: Physical primitives and trusted context resolution

**Interfaces consumed:** `core.ParseSource(string) (netip.Prefix, error)` and `core.ValidateSource(netip.Prefix) error` for shared address syntax, with destination-specific error context. These remain separate source/destination roles.

**Interfaces produced** in `types.go`, `network.go`, `context.go` (package `networkplan`):

```go
type EntryRef struct { ResourceName, EndpointName string }
type Protocol string
const ( TCP Protocol = "tcp"; UDP Protocol = "udp" )
type TrafficClass string
const (
    HostLocal TrafficClass = "host-local"
    PublishedIngress TrafficClass = "published-ingress"
    Transit TrafficClass = "transit"
)
type ObservationStage string
const OriginalDestination ObservationStage = "original-destination"
type HandlingMode string
const ( Graceful HandlingMode = "graceful"; Strict HandlingMode = "strict" )
type Family uint8
const ( IPv4 Family = 4; IPv6 Family = 6 )
type NetworkDefinition struct {
    Entry EntryRef
    Protocol Protocol
    Port uint16
    Addresses []netip.Prefix
}
type ContextRequest struct {
    ProfileRef string
    Class TrafficClass
    Stage ObservationStage
    AllowedDestinations []netip.Prefix
    Handling HandlingMode
}
type ExecutionContext struct {
    AgentRef, ProfileRef, ID, NetworkSpaceID string
    Class TrafficClass
    Stage ObservationStage
    Destinations []netip.Prefix
    Handling HandlingMode
}
type CapabilityProfile struct {
    Classes []TrafficClass
    Protocols []Protocol
    Families []Family
    Handling []HandlingMode
    OriginalDestination bool
}
func ParseDestinations(values []string) ([]netip.Prefix, error)
func CanonicalDestinations(values []netip.Prefix) ([]netip.Prefix, error)
func IntersectDestinations(a, b []netip.Prefix) ([]netip.Prefix, error)
func NormalizeNetwork(n NetworkDefinition) (NetworkDefinition, error)
func NormalizeContextRequest(r ContextRequest) (ContextRequest, error)
func NormalizeExecutionContext(c ExecutionContext) (ExecutionContext, error)
func NormalizeCapabilities(c *CapabilityProfile) (*CapabilityProfile, error)
func ResolveContext(r ContextRequest, c ExecutionContext, caps *CapabilityProfile) (ExecutionContext, error)
```

Parsing may mask CIDRs; typed normalization requires already canonical valid prefixes. `CanonicalDestinations(nil)` preserves omission; non-nil empty errors. Canonical union removes contained prefixes and recursively merges siblings per family, without enumerating hosts. `IntersectDestinations` requires two nonempty valid sets; a disjoint result is nil without error, which selection later rejects.

All normalized results own their collections. Contexts require nonempty Agent/Profile/ID/network-space, explicit class, original stage, handling and destination boundary. `ResolveContext` requires matching ProfileRef/class/stage/handling and declared support. Trusted facts must already be a subset of the request's allowed boundary: reject broader facts, do not clip them while reusing a context identity. A runtime resolver can supply narrower facts under an appropriate versioned ID. This conservative first-version choice implements the approved resolve/narrow/reject rule without rewriting trusted facts. Prefixes alone never prove the traffic guard.

`NormalizeCapabilities(nil)` returns nil without error (unknown inventory metadata); resolving a selected context with nil capabilities errors. Validate enum sets even on unused profiles, sort/deduplicate/copy them. Required physical protocol/family checks happen in Task 3 after intersection. A non-nil profile with empty sets supports nothing; `OriginalDestination=false` cannot satisfy the stage.

- [x] **1. Write the named failing tests** in the task's test files. Assertion excerpts below use local fixture variables; build those fixtures in the owning test package. Do not reuse another package's private test helpers.

```go
// TestParseDestinationsPresenceAndFamilies: nil -> nil; []string{} -> error;
// "192.0.2.9" -> "192.0.2.9/32"; "2001:db8::9" -> "2001:db8::9/128";
// "192.0.2.129/24" -> "192.0.2.0/24".
if got[0] != netip.MustParsePrefix("192.0.2.0/24") { t.Fatal(got) }
// Error cases: "::ffff:192.0.2.1", "fe80::1%eth0", "gitea.example", "bad".
// TestCanonicalDestinations: /25 siblings merge to /24; contained /32 removed;
// IPv6 siblings /65 merge to /64; /0 handled without address enumeration.
if !reflect.DeepEqual(got, []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}) { t.Fatal(got) }
// Typed noncanonical prefixes and non-nil empty lists error; nil stays omitted.
// TestDestinationIntersection: /24 intersect /25 == /25; disjoint or cross-family
// result is nil without error; nil operand is invalid, never an unrestricted set.
// TestNormalizeNetworkRejectsInvalid: protocol "", "sctp", port 0, empty refs error;
// TCP/UDP port 1 and 65535 succeed; returned addresses are owned.
// TestResolveContextAuthorityBoundary: request /24 accepts trusted facts /25;
// facts /23, changed class/stage/handling/profile and empty boundary error.
if got.Destinations[0] != netip.MustParsePrefix("192.0.2.128/25") { t.Fatal(got) }
// TestNormalizeExecutionContext: missing Agent/Profile/ID/space, unknown enum error;
// explicit /0 succeeds only as an explicit boundary, with class requirement retained.
// TestCapabilityValidation: nil profile normalizes to nil but selected resolution errors;
// unknown enum errors; missing class/stage/handling support errors; all sets copied/sorted.
```

- [x] **2. Run the focused command** below and observe the expected behavior failure. For new APIs, add only declarations/minimal error-returning stubs first so red evidence exercises assertions rather than missing imports.

```bash
go test ./pkg/networkplan -run 'Test(ParseDestinations|CanonicalDestinations|DestinationIntersection|NormalizeNetwork|ResolveContext|NormalizeExecutionContext|CapabilityValidation)' -count=1
```

- [x] **3. Implement the declared interfaces** in the mapped files, following the task's rules and the approved spec; no unrelated refactor.

- [x] **4. Re-run the focused command** and the indicated regression command; require PASS and exit 0.

```bash
go test ./pkg/networkplan -run 'Test(ParseDestinations|CanonicalDestinations|DestinationIntersection|NormalizeNetwork|ResolveContext|NormalizeExecutionContext|CapabilityValidation)' -count=1
go test ./pkg/core ./pkg/networkplan -count=1
```

- [x] **5. Commit only this task's files** with the message below after reviewing `git diff --check` and the staged diff.

```bash
git commit -m "feat(networkplan): normalize physical definitions and contexts"
```

### Task 3: Placement, conflicts and immutable materialization

**Interfaces consumed:** Task 1 `NetworkView`, Task 2 normalizers/resolver; Core `NormalizeSelector`, `LabelSelector.Matches`, `SelectEndpointNames`, `Validity.ActiveAt`, `Validity.NextBoundary`. Do not walk raw Grant scopes in the planner.

**Interfaces produced** in `types.go` / `plan.go`; selection in `binding.go`, overlap validation in `conflict.go`:

```go
type ResourceSelection struct {
    ResourceName string
    ResourceSelector *core.LabelSelector
    Endpoints core.EndpointSelection
}
type AgentSelection struct { AgentRef string; Selector *core.LabelSelector }
type Binding struct {
    ID string
    Resources []ResourceSelection
    Agents []AgentSelection
    Context ContextRequest
}
type Agent struct {
    Name string
    Labels map[string]string
    Offline bool
    Capabilities *CapabilityProfile
}
type Input struct {
    AuthorityID string
    Logical core.Snapshot
    Networks []NetworkDefinition
    Agents []Agent
    Contexts []ExecutionContext
    Bindings []Binding
    Requirements []string
}
type Diagnostic struct { Code, BindingID, ResourceName, AgentRef, EndpointName string }
type TargetKey struct { AgentRef, ContextID string; Entry EntryRef }
type Match struct { Destination netip.Prefix; Protocol Protocol; Port uint16 }
type Protection struct {
    Key TargetKey
    Context ExecutionContext
    BindingIDs []string
    Matches []Match
    Required CapabilityProfile
}
type TargetInventory struct {
    Protection Protection
    Grants, Denies []core.ResolvedContribution
}
type Membership struct {
    Target TargetKey
    Match Match
    Contribution core.ResolvedContribution
}
type Projection struct {
    AuthorityID, AgentRef string
    At time.Time
    Protected []Protection
    Allow, Deny []Membership
    NextChange time.Time
}
type Plan struct { /* private owned state */ }
func Compile(input Input) (*Plan, error)
func (p *Plan) AgentRefs() ([]string, error)
func (p *Plan) Diagnostics() ([]Diagnostic, error)
func (p *Plan) Projection(agentRef string, at time.Time) (Projection, error)
func (p *Plan) Inventory(agentRef string) ([]TargetInventory, error)
```

`AuthorityID`, names and Binding IDs must be nonempty and unique in their domains. Call Core.Compile once and retain its complete owned view. Require exactly one normalized network definition per logical entry, including unbound entries. Validate every supplied row even if unused. Any nonempty `Requirements` list fails with the supplied unsupported values (test `port-range`, `nat64`, `dns`, `post-dnat`, `unknown`); this field does not authorize silently dropping future wire fields. Future decoders must reject unknown fields separately.

Each Binding has nonempty Resource/Agent item lists. Resource selectors are represented only so they can be explicitly rejected; exact Resource references and Core endpoint selection are supported. Agent items have exactly one known ref or nonempty Core selector. Union/deduplicate, then take the entry×Agent cross-product. Validate explicit references even if another dimension is empty. Diagnostics `no-agents-matched` and `no-entries-matched` are sorted by their field tuple. Offline status does not change placement; labels are not inherited.

Index trusted contexts by `(AgentRef, ProfileRef)` and reject duplicate keys or `(AgentRef, ID)` identities, even if equivalent. Context rows must reference known Agents. Missing selected context/capabilities fails; unused nil capabilities may remain inventory metadata. Resolve requested context, intersect destinations (nil entry addresses use its boundary), reject selected empty intersection with Agent/Binding/entry in the error. Check capabilities for the actual physical protocol/families plus class/stage/handling; emit the same minimal `Required` profile for adapter revalidation.

Conflict checking uses trusted Agent/network-space, original destination, protocol/port and prefix overlap, even with zero active Grants. Different logical entries on overlapping tuples fail, including two endpoints in one Resource. Identical entry/context/matches/handling merge sorted Binding IDs. Conservatively reject overlapping distinct context identities for one entry; labels/IDs alone prove no isolation. Disjoint tuples or distinct Agents coexist. Return nil Plan on any error.

Store full normalized definitions/view/context catalog for Task 4, including unbound/inactive rows. Plan fields are private, all read methods return owned data. Sort TargetKey by AgentRef/ContextID/ResourceName/EndpointName; matches by protocol/port/address family/address/prefix bits; memberships additionally by source family/address/bits/ID. Sort diagnostics by declared fields; canonical unions make equivalent orderings equal. `AgentRefs` lists only selected desired Agents. A known unselected Agent has valid empty Inventory/Projection; unknown Agent, zero time, nil/zero Plan errors. Normalize output At to UTC without monotonic data.

Inventory retains all same-family relevant contributions, including revoked/inactive/future. Projection retains protected targets and emits active nonrevoked Allow/Deny memberships separately, deduplicated by target/match/typed ID. Active deny takes precedence when an adapter evaluates a flow; the domain lists preserve overlapping allows and denies. NextChange is the earliest future Core boundary among relevant nonrevoked members, including deny activation/expiry; no TTL conversion. Protection survives zero active Grants. These domain memberships do not specify one kernel rule per Grant. No timer, applying goroutine or connection cleanup is added.

- [x] **1. Write the named failing tests** in the task's test files. Assertion excerpts below use local fixture variables; build those fixtures in the owning test package. Do not reuse another package's private test helpers.

```go
// TestBindingCrossProduct: 2 entries x 2 Agents == 4 owned targets;
// overlapping selectors and duplicate scopes never duplicate memberships.
if len(a1)+len(a2) != 4 { t.Fatal(a1, a2) }
// TestBindingValidation: duplicate/missing IDs, resource selector, both/neither Agent
// selection, empty selection lists, orphan/missing/duplicate Networks, invalid unused
// inputs and each nonempty Requirements value error; returned Plan is nil.
if err == nil || plan != nil { t.Fatal(plan, err) }
// TestBindingNoMatchesAndOffline: zero selector matches yields "no-agents-matched";
// known empty Resource yields "no-entries-matched"; explicit missing ref errors;
// Offline=true does not remove a target; Resource labels do not become Agent labels.
// TestContextSelectionAndCapabilities: missing selected context or unsupported
// physical protocol/family errors; same ProfileRef on two Agents resolves separately;
// same (AgentRef,ID) or (AgentRef,ProfileRef) duplicate errors, even if equal.
// TestProjectionBoundaryAndFamilies: entry /25 within /24 -> exactly /25;
// omitted addresses -> context union; v4 entry -> no v6 membership;
// selected empty intersection errors with Agent/Binding/entry in error text.
// TestProjectionMembershipAndSchedule: at 09:00 allow host(/32) until 10:00 and
// prefix(/24) until 11:00, future deny starts 09:30 and expires 10:30.
if !projection.NextChange.Equal(time.Date(2026,10,15,9,30,0,0,time.UTC)) { t.Fatal(projection) }
// Separate future-allow case: at 08:00, NotBefore=08:30 -> no active membership,
// NextChange=08:30; at 08:30 membership appears. At 10:00 host absent, prefix retained;
// at 10:30 deny absent; revoked permanent absent; explicit permanent has no expiry;
// same-family allow/deny lists retain both domains with ID "shared".
// TestProtectedWithoutPermissions: Protected remains with no active Grants;
// unbound logical allow has no physical target; unknown Agent/zero At/nil or zero Plan error;
// known unselected Agent returns an empty owned projection, not ALL.
// TestPhysicalConflict: same Agent/space/protocol/port and overlapping destinations
// under different entries (including same Resource) errors even without Grants;
// same entry with incompatible overlapping contexts or graceful/strict handling errors;
// different IDs prove no isolation.
// Disjoint destinations, different protocol/port, or distinct Agents succeed.
// TestBindingProvenanceMerge: equivalent same-entry/context matches merge IDs sorted.
if !reflect.DeepEqual(protection.BindingIDs, []string{"b1", "b2"}) { t.Fatal(protection) }
```

- [x] **2. Run the focused command** below and observe the expected behavior failure. For new APIs, add only declarations/minimal error-returning stubs first so red evidence exercises assertions rather than missing imports.

```bash
go test ./pkg/networkplan -run 'Test(Binding|ContextSelection|Projection|Protected|PhysicalConflict)' -count=1
```

- [x] **3. Implement the declared interfaces** in the mapped files, following the task's rules and the approved spec; no unrelated refactor.

- [x] **4. Re-run the focused command** and the indicated regression command; require PASS and exit 0.

```bash
go test ./pkg/networkplan -run 'Test(Binding|ContextSelection|Projection|Protected|PhysicalConflict)' -count=1
go test -race ./pkg/core ./pkg/networkplan -count=1
```

- [x] **5. Commit only this task's files** with the message below after reviewing `git diff --check` and the staged diff.

```bash
git commit -m "feat(networkplan): compile immutable per-agent intent"
```

### Task 4: Complete replacements and withdrawal inventory

**Interfaces consumed:** Task 3 `Compile`, private full normalized definitions/view/context catalog, `Inventory` and immutable Plan.

**Interfaces produced** in `types.go` / `transition.go`:

```go
type Withdrawal struct { Former TargetInventory; Reason string }
type Replacement struct { Plan *Plan; Withdrawals []Withdrawal }
func Replace(previous *Plan, input Input) (Replacement, error)
```

Require a nonnil valid previous plan and identical AuthorityID; `Compile` is the separate trusted initial-bootstrap path, not adoption of unknown installed rules. Compile the entire candidate before accepting it. Return zero Replacement on any failure; never mutate the previous plan or extend its deadlines.

For every retained EntryRef compare the full canonical network definition (protocol, port, address presence and union), not the currently visible intersection. Omitted addresses and an explicit currently equivalent list remain distinct intent. If changed, reject each retained Grant ID or Deny ID in the same domain that still resolves to this entry, including revoked/expired/unbound members. Report entry and retained typed IDs. Equivalent canonical unions are unchanged. Fresh-ID reissuance fixtures copy original absolute limits; the planner never creates a renewed deadline. Since there is no old→new approval mapping in this input, it cannot prove a genuinely new ID represents a particular old approval: that eligibility and retained-limit proof belong to the authoritative publisher, not a fabricated ID heuristic.

Compare all old/new trusted context catalog rows sharing `(AgentRef,ID)`, even unused rows. NetworkSpaceID/class/stage/canonical boundary/handling/ProfileRef are immutable for that identity. A new ID follows ordinary validation, conflicts and withdrawal handling. Added entries and label edits preserve Core's dynamic whole/label semantics; named selections stay fixed. Missing named references fail; removal/rename requires consistent removal or explicit reissuance of named contributions. No tombstones, durable UID/restart adoption or Controller/Static merge is implemented.

Diff desired protection by TargetKey. Absent old key yields `Reason="removed"`; changed matches under the same key yields `Reason="match-changed"`. Each withdrawal carries the complete owned `Former` target (matches/context/Binding IDs/all relevant contributions), sorted by TargetKey, even if the Agent disappeared. Losing one Binding while another remains, or changing permission membership alone, is not a protection withdrawal. New Inventory exposes those membership changes to the later adapter. This is full old-target reconsideration intent, not permission to delete every former flow: actual strict cleanup must compare installed/observed historical ownership with the confirmed new policy and preserve remaining allowed flows.

- [x] **1. Write the named failing tests** in the task's test files. Assertion excerpts below use local fixture variables; build those fixtures in the owning test package. Do not reuse another package's private test helpers.

```go
// TestReplaceAuthorityAndAtomicity: nil/zero previous or changed AuthorityID errors;
// every rejected replacement has nil Plan and no Withdrawals; old view stays equal,
// old absolute deadlines still expire at 10:00/11:00, not replacement time + TTL.
if err == nil || got.Plan != nil || len(got.Withdrawals) != 0 { t.Fatal(got, err) }
// TestRetargetRetainedContributions: change protocol/port/full destination union with
// old Grant OR Deny ID still resolving to same entry -> error; includes unbound,
// expired and revoked members and a changed address region hidden by current boundary.
// Same string in the OTHER ID domain alone does not count as a retained ID.
// TestRetargetCanonicalAndReissue: reordered equivalent unions succeed; omitted vs
// explicit addresses is a definition change; fresh IDs with original absolute ends succeed.
if !reissued.Validity.ExpiresAt.Equal(original.Validity.ExpiresAt) { t.Fatal(reissued) }
// TestContextIdentityMutation: reuse (AgentRef,ID) with changed space/class/stage/
// boundary/handling/ProfileRef errors, including unused catalog rows; new ID is allowed
// after normal placement/conflict validation and yields former-target withdrawal.
// TestWithdrawalInventory: remove b1 -> b2 protection retained, no withdrawal;
// remove last Grant -> protection retained; remove last Binding -> one full Former.
if len(got.Withdrawals) != 1 || got.Withdrawals[0].Reason != "removed" { t.Fatal(got) }
// Removing an Agent still returns its old matches, context, BindingIDs and contributions.
// Same target key with changed matches yields Reason "match-changed"; membership-only
// changes yield no protection withdrawal and remain visible in new Inventory.
// TestHistoricalFormerInventory: expired former allow is retained in Former; this is
// desired history, explicitly not proof it was installed or an exact conntrack before-set.
// TestEntryLifecycleSelections: added entry/labels follow whole/label dynamics;
// names stay fixed; removing referenced named entry fails; remove/reissue named grants succeeds.
```

- [x] **2. Run the focused command** below and observe the expected behavior failure. For new APIs, add only declarations/minimal error-returning stubs first so red evidence exercises assertions rather than missing imports.

```bash
go test ./pkg/networkplan -run 'Test(Replace|Retarget|ContextIdentity|Withdrawal|HistoricalFormer|EntryLifecycle)' -count=1
```

- [x] **3. Implement the declared interfaces** in the mapped files, following the task's rules and the approved spec; no unrelated refactor.

- [x] **4. Re-run the focused command** and the indicated regression command; require PASS and exit 0.

```bash
go test ./pkg/networkplan -run 'Test(Replace|Retarget|ContextIdentity|Withdrawal|HistoricalFormer|EntryLifecycle)' -count=1
go test -race ./pkg/core ./pkg/networkplan -count=1
```

- [x] **5. Commit only this task's files** with the message below after reviewing `git diff --check` and the staged diff.

```bash
git commit -m "feat(networkplan): validate replacements and retain withdrawals"
```

### Task 5: Public conformance, documentation and final verification

**Interfaces consumed:** Tasks 1–4 public APIs; no new runtime interfaces produced. `ExampleCompile` uses external package `networkplan_test` and explicit fixed time. Build local fixture helpers in `conformance_test.go`; assertions use literal expected values rather than another planner invocation.

Document in `doc.go` and both README languages: input authority/trusted facts boundary, nil vs empty addresses, unsupported requirements, fresh-ID reissuance boundary, immutable owned results, scheduling via NextChange, desired vs installed history, explicit empty-owned cleanup, protected vs unmanaged traffic and conservative context conflicts. Mark M1.2 implemented only after code and checks pass; M1.3 still needs its own bridge design and real privileged packet evidence. The bridge must preserve guards/original tuples, schedule future activation, apply/confirm before scoped strict cleanup, use installed historical before-state and handle M0's empty Protected limitation explicitly. Pure tests demonstrate none of those runtime outcomes.

- [x] **1. Write the named failing tests** in the task's test files. Assertion excerpts below use local fixture variables; build those fixtures in the owning test package. Do not reuse another package's private test helpers.

```go
// TestGiteaConformance: gitea https TCP/443 ends 10:00, ssh TCP/22 ends 12:00;
// source 198.51.100.9/32 plus 198.51.100.0/24 ends 11:00, independent deny;
// host-local boundary 192.0.2.100/32 + 2001:db8::100/128, explicit original stage.
// Assert literal targets, sources, IDs and timestamps (no expected value from planner).
// For declared owned tuples at fixed times, compare active source membership with
// literal expected Core.Decide results, applying deny precedence in the test helper.
// Keep an unbound/unrelated tuple as an unmanaged control, never an implicit allow.
// TestProviderNormalizedEquivalence: equivalent Controller/static-style input order
// yields equal projections after removing only AuthorityID metadata; Replace across
// those different authorities still errors. No provider or decoder is implemented.
// TestPlanOwnershipAndConcurrentReads: mutate every nested input/output collection,
// including capabilities/context prefixes/provenance/contributions; fresh results unchanged.
if !reflect.DeepEqual(before, after) { t.Fatal(after) }
// Concurrent Projection/Inventory/AgentRefs/Diagnostics and independent mutation pass -race.
// ExampleCompile: external package networkplan_test; deterministic printed result:
// Output:
// agent-a gitea/https tcp/443 192.0.2.100/32
```

- [x] **2. Run the focused command** below and observe the expected behavior failure. For new APIs, add only declarations/minimal error-returning stubs first so red evidence exercises assertions rather than missing imports.

```bash
go test ./pkg/networkplan -run 'Test(GiteaConformance|ProviderNormalizedEquivalence|PlanOwnershipAndConcurrentReads)|ExampleCompile' -count=1
```

- [x] **3. Implement the declared interfaces** in the mapped files, following the task's rules and the approved spec; no unrelated refactor.

- [x] **4. Re-run the focused command** and the indicated regression command; require PASS and exit 0.

```bash
go test ./pkg/networkplan -run 'Test(GiteaConformance|ProviderNormalizedEquivalence|PlanOwnershipAndConcurrentReads)|ExampleCompile' -count=1
go test -race ./... -count=1
```

- [x] **5. Commit only this task's files** with the message below after reviewing `git diff --check` and the staged diff.

```bash
git commit -m "test(networkplan): add gitea conformance and public usage"
```

## Final acceptance and handoff

Run from the repository root using Go 1.27.0. If the toolchain is outside PATH, prepend its bin directory; do not alter go.mod to fetch a different version.

```bash
gofmt -l cmd internal pkg
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./pkg/core ./pkg/networkplan
git diff --exit-code 315e0dd5617eed6cfc5055322d23b56378179cd2 -- go.mod go.sum
git diff --check
```

Require no gofmt output, all test/build/vet commands exit 0, and nonstandard dependency output limited to `github.com/PicketX/PicketX/pkg/core` and `github.com/PicketX/PicketX/pkg/networkplan`. go.mod/go.sum diff stays empty. Existing Go CI must pass on the final PR head (format/vet/race/build); existing Web CI also remains green. Record actual run/head IDs rather than only workflow configuration. No new skip, kernel test claim or unrelated dependency is acceptable.

All 17 spec acceptance rows map to Tasks 1–5: validation/addresses/context (1–3), placement/conflict/time/provenance (3), lifecycle/withdrawal (4), ownership/provider-equivalence/Gitea (5). Self-review checked all eight design sections, all cross-task types and five Review Focus rows. No scope gap is assigned to an undefined later task inside M1.2; the explicitly excluded M1.3 runtime obligations remain visible above.

## Execution record

Tasks 1–4 were implemented after assertion-level red runs against minimal API stubs; the historical-clock regression failed against actual M1.1 code before correction. Task 5 adds cross-layer conformance and a public example over that implemented behavior: all pass on their first run, so no artificial failure is claimed. This is the sole execution ruling; the cost is that acceptance added after unit implementation may miss a gap, mitigated by literal expectations and the fresh final review.

Go 1.27.0: test, race, vet and build pass. All 32 planned named tests ran; 65 top-level tests/examples pass across Core (35), planner (27) and Controller (3), with no skipped tests. The command entry package has no test files. Dependency inspection lists only Core/planner outside the standard library; go.mod/go.sum stay unchanged. No packet/kernel validation is claimed. Independent review and final-head CI evidence are recorded in the implementation PR when available.
