# M1.3 Core/规划层到 M0 桥接实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Preserve native inline execution and one fresh whole-branch review at the end.

> [English (default)](2026-10-15-core-m0-bridge.md) | 简体中文

**Goal:** 将共享规划结果接入 PicketXD 隔离夹具，以真实双栈报文证明期限、替换及精确撤销。

**Architecture:** 纯转换器只处理显式时刻与来源记录；串行协调器管理确认、历史和测试边界驱动。后端提供实际发出回执、原子入站/DNAT、精确自有清理。

**Tech Stack:** Go 1.27.0; existing pinned Go Netlink modules; standard-library clocks/timers; `github.com/PicketX/PicketX/pkg/core` and `pkg/networkplan`.

**Spec:** [M1.3 approved design](../specs/2026-10-15-core-m0-bridge-design.zh-CN.md), [PR #8](https://github.com/PicketX/PicketX/pull/8), merged as `9d53b13e2d8a805b2a7b2f3a8ad957fcf84ea12d`.

**Status:** 保留的历史规划基线，已通过持续实施授权批准。M1.3 已由聚焦的[编译器增量](https://github.com/PicketX/PicketXD/pull/2)和[确认生命周期增量](https://github.com/PicketX/PicketXD/pull/3)交付并合并。最终 API、新增 19 项桥接场景和官方依赖修复以 Agent 仓库的已执行计划及验收记录为准；下方未勾选步骤是原始计划，不代表当前待办。[M2B](https://github.com/PicketX/PicketXD/pull/4) 也已合并，全部 43 项必需内核场景通过。生产执行、七层及管理端仍待后续推进。

## 全局约束

- 共享依赖固定为不可变 `e39f96e2787154a8af8e7d996b2f65aa10b6c3eb`，解析并记录 pseudo-version/校验和；不提交浮动分支或本地 `replace`。
- Go 1.27.0；GPL-3.0-or-later PicketXD 可以依赖 Apache Core/规划层，禁止反向依赖。仅导入 `pkg/core`、`pkg/networkplan`，区分模块图与实际链接的包依赖。
- 仅支持已证明的 `transit`、`original-destination`、固定 graceful/strict 模式，每协调器绑定一个命名空间/epoch/权威来源，不扩大夹具范围。
- 普通目录为 TCP/443、TCP/22、UDP/8002；NAT 目录为原始 TCP/8000、TCP/8001 -> Server/9000，使用独立显式上下文。IPv4/IPv6 同等覆盖。
- 仅操作自有 `inet picketx` 和非零 `nftbackend.TestMark`；禁止全局 conntrack flush、哨兵保护及放松通用夹具校验。
- 转换/重放保留绝对期限，仅显式永久贡献映射零到期。边界观察窗口 **1 秒**，不是权限延期。
- 完整候选 -> 新鲜时间物化 -> 应用/ACK/读回 -> 记录 -> 可选精确清理。应用不明阻断清理与后续应用；清理失败保留已确认策略和历史证据。
- 保留全部 17 项 M0 必需场景，新增下表 16 项桥接内核场景。显式启用后的权限/初始化失败必须失败，不得跳过；单元通过不算内核证据。
- 本轮为私有契约验证设施，不实现生产 provider/传输/LKG/schema/UI，不声称完成 TLS/SSH 认证；保留 M1.2 selector 诊断 Minor。


## 评审重点

| 输入/故障 | 预期行为 | 所属单元测试 |
| --- | --- | --- |
| Agent/上下文同名但命名空间重建 | 副作用前拒绝外来 epoch | `TestBridgeCoordinatorIdentityAndOwnership` |
| ACK 与读回之间成员到期 | 保留实际发出历史，仅容许合法成员到期 | `TestBridgeCoordinatorHistoryAcrossExpiry` |
| flush 前追加准备失败 | 重试前丢弃待提交批次 | `TestConfirmedApplyBatchIsolation` |
| Agent 移除与任意查询错误 | 仅接受通过校验的显式空依据 | `TestBridgeCoordinatorEmptyAndRemovedAgent` |
| 慢应用跨过下一边界 | 立即新鲜刷新，过时 timer 不得重放 | `TestBridgeDriverStaleCancellationAndDelay` |

## 上下文、文件与依赖地图

以下实现路径均相对 **PicketX/PicketXD**；中央计划/路线图及链接保留在 **PicketX/PicketX**。上下文地图随本计划一并评审。

| Task | Files to create/modify | Responsibility |
| --- | --- | --- |
| 1 | `go.mod`, `go.sum`; create `internal/validation/planbridge/{types,prepare,materialize}.go` and corresponding `_test.go` | Immutable shared dependency, scope and pure conversion |
| 2 | modify `internal/validation/kerneltest/{topology_linux,nat_linux}.go`; create `bridge_linux.go`, `bridge_integration_test.go` | Actual namespace identity, tuple proof, listener/connection cleanup, append-only NAT |
| 3 | modify `internal/validation/nftbackend/backend_linux.go`; create `confirmed_linux.go`, `readback_linux.go`, `confirmed_linux_test.go` | Dispatch-time expiry, receipts, atomic fixture batch, confirmation, owned-table removal |
| 4 | modify `internal/validation/ctbackend/{backend_linux,select_linux}.go`; create `empty_linux_test.go` | Explicit empty-after precise revocation |
| 5 | create `internal/validation/planbridge/{coordinator,history,driver}.go` and corresponding `_test.go` | Serialized apply state machine, cumulative history, explicit retry and cancelable driver |
| 6 | create `internal/validation/planbridge/{conformance_integration_test,helpers_integration_test}.go` | Literal Gitea/UDP/NAT and lifecycle packet cases |
| 7 | modify `internal/validation/report/{report.go,report_test.go}`, `cmd/m0-report/main.go`, `scripts/test-kernel.sh`, `.github/workflows/validation.yml`, paired Agent READMEs; create paired `docs/validation/m13-core-bridge.md` | Required evidence and truthful delivery report |

依赖：`planbridge` 导入 Core/规划层、fixture 及回执/结果类型；`kerneltest` 不导入 `planbridge`。外部集成包 `planbridge_test` 将拓扑、nftbackend、ctbackend 组装成 Effects。后端不导入 Core/Controller。现有 M0 `scenario.ApplyStrict`、`Backend.Apply`、`Backend.Revoke` 继续可用。参考 `fixture.Validate`、`ctbackend.exactFilter`、`kerneltest.WithTopology`、现有 integration 标签及 `report.CheckRequired`。

风险评估：不修改公开传输 API、数据库迁移或生产运行配置。私有后端 API 扩展，现有 M0 测试须持续通过。验收需要真实特权 Linux。直接执行顺序为 **1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7**；每任务独立提交与验证，全部结束后一次独立全分支评审。

### 任务 1: 纯范围、完整候选校验与物化

**文件：** Task 1 paths above; `types_test.go`, `prepare_test.go`, `materialize_test.go`.

**消费接口：** existing `networkplan.Compile(Input)`, `Replace(*Plan, Input)`, `Inventory(string)`, `Projection(string,time.Time)` and Core validity.

**产出接口：**

```go
type NamespaceID struct { Device, Inode uint64 }
type Scope struct {
    AuthorityID, AgentRef, Epoch string
    Namespace NamespaceID
    Context networkplan.ExecutionContext
    Catalog []networkplan.Match
}
type Origin struct {
    AuthorityID string
    Target networkplan.TargetKey
    BindingIDs []string
    Domain string // "grant" or "deny"
    Contribution core.ResolvedContribution
    Match networkplan.Match
}
type Member struct { Rule fixture.Rule; Origins []Origin }
type Materialized struct {
    Snapshot fixture.Snapshot
    Allow, Deny []Member
    Inventory []networkplan.TargetInventory
    At, NextChange time.Time
}
type Candidate struct { /* private immutable plan, withdrawals, scope and explicit empty proof */ }
func Prepare(previous *networkplan.Plan, input networkplan.Input, scope Scope) (*Candidate, error)
func Materialize(candidate *Candidate, at time.Time) (Materialized, error)
```

- [ ] **1. 编写失败测试。** 在当前包构造字面量夹具；下列代码为各测试的断言片段，局部变量由对应案例设置。固定 t0=`2026-10-15T10:00:00Z`，HTTPS end=t0+2s，SSH end=t0+4s，未来 start=t0+1s；纯单元测试使用假时钟。

```go
// TestBridgePrepareRejectsUnsupportedScope: err != nil; candidate == nil; backendCalls == 0 for host-local, published-ingress, post-DNAT, mismatched Agent/context/profile/namespace, unsupported family/protocol/mode, wider destination, TCP/444 and NAT TCP/443.
if err == nil || candidate != nil || backendCalls != 0 { t.Fatal(candidate, err, backendCalls) }
// TestBridgePrepareUsesFullReplacement: authority switch and retarget with retained Grant/Deny IDs fail even when unbound; previous plan remains unchanged.
if err == nil || !reflect.DeepEqual(previousInventory, afterInventory) { t.Fatal(err, afterInventory) }
// TestBridgeMaterializeProvenanceAndDeadlines: TCP/443 and TCP/22 map to protocol 6, UDP/8002 to 17; all Binding IDs and separate grant/deny origins survive; exact end is inactive; only permanent has zero expiry.
if https.Protocol != 6 || https.PortFirst != 443 || https.PortLast != 443 || ssh.PortFirst != 22 || udp.Protocol != 17 || udp.PortFirst != 8002 { t.Fatal(got) }
if !bounded.ExpiresAt.Equal(end) || !permanent.ExpiresAt.IsZero() || len(bindingIDs) != 2 || grantOrigin.Domain != "grant" || denyOrigin.Domain != "deny" { t.Fatal(got) }
// TestBridgeMaterializeOwnsData: mutating input/scope/returned origins, slices and inventory leaves a fresh materialization unchanged; concurrent reads pass race.
if !reflect.DeepEqual(before, fresh) { t.Fatal(fresh) }
// TestBridgeMaterializeInactiveAndEmpty: future/revoked/expired members absent; zero active Grants retains Protected; known unselected Agent is explicitly empty; arbitrary query error fails.
if len(got.Snapshot.Allow) != 0 || len(got.Snapshot.Deny) != 0 || len(got.Snapshot.Protected) == 0 { t.Fatal(got) }
```

- [ ] **2. 确认失败。** 执行 `go test ./internal/validation/planbridge -run "TestBridge(Prepare|Materialize)" -count=1`；缺失 API/行为应导致失败。集成场景若基线已通过，记录基线并增加字面量反例，不人为破坏正常代码。

- [ ] **3. 实现。** Prepare 首次使用 Compile，随后使用 Replace；独立持有 scope/catalog，依据完整合法候选及 withdrawals 保存 Agent 移除的显式依据。Materialize 使用调用者时刻的 projection 与完整 inventory，保留非活跃来源记录，但仅发出当前活跃成员。Grant/Deny 来源域独立；不能合并独立期限，一贡献一物理成员即可。所有选中元组及上下文必须精确符合 Scope；元数据本身不证明拓扑。空物化必须携带候选的明确依据，任意零快照无权触发清理。

- [ ] **4. 确认通过。** 执行 `go test ./internal/validation/planbridge -run "TestBridge(Prepare|Materialize)" -count=1`；每个具名测试须通过，不得意外跳过，先前任务及 M0 回归继续通过。

- [ ] **5. 提交。** 仅暂存当前任务文件，提交信息为 `feat(validation): materialize shared network plans`；最终验证记录保留失败及通过证据。

### 任务 2: 已证明拓扑与原子 NAT 追加接口

**文件：** Task 2 paths above; preserve existing 8000/8001/8002/9000 listeners.

**消费接口：** existing WithTopology, explicit namespace handles, fixture links/routes and packet helpers.

**产出接口：**

```go
// In kerneltest; no planbridge import, so integration tests can join both packages.
type BridgeIdentity struct { Device, Inode uint64; Epoch string }
func (x *Topology) BridgeIdentity() (BridgeIdentity, error)
func (x *Topology) VerifyBridge(ctx context.Context, id BridgeIdentity, nat bool) error
func AppendDNAT(c *nftables.Conn, table *nftables.Table) error
```

- [ ] **1. 编写失败测试。** 在当前包构造字面量夹具；下列代码为各测试的断言片段，局部变量由对应案例设置。固定 t0=`2026-10-15T10:00:00Z`，HTTPS end=t0+2s，SSH end=t0+4s，未来 start=t0+1s；纯单元测试使用假时钟。

```go
// TestBridgeTopologyIdentity: same live namespace yields stable device/inode and epoch; new topology has a new epoch; closed/replaced handle rejects.
if id.Device == 0 || id.Inode == 0 || id.Epoch == "" || id != again || id.Epoch == other.Epoch { t.Fatal(id, again, other) }
// TestBridgeTopologyTupleCatalog: unfiltered IPv4 and IPv6 reach TCP/443, TCP/22 and UDP/8002; NAT 8000/8001 reaches Server/9000; protected destinations are not Enforcement-local.
if err != nil || ipv4Probes != 3 || ipv6Probes != 3 || destinationIsEnforcementLocal { t.Fatal(err) }
```

- [ ] **2. 确认失败。** 执行 `PICKETX_KERNEL_TESTS=1 go test -tags=integration ./internal/validation/kerneltest -run "TestBridgeTopology" -count=1`；缺失 API/行为应导致失败。集成场景若基线已通过，记录基线并增加字面量反例，不人为破坏正常代码。

- [ ] **3. 实现。** BridgeIdentity 记录 fstat 的 device/inode 及每拓扑随机 epoch。VerifyBridge 在每次应用前检查句柄身份、转发、路由/地址事实和所有权。初始化时证明精确未过滤目录可达；应用前检查身份/能力，不能将策略拒绝误判为网络故障。增加 TCP/443、TCP/22 echo 监听，关闭已接受连接及监听器，保证重复测试释放命名空间资源。AppendDNAT 仅排队现有固定 VIP/8000-8001 -> Server/9000 翻译，不能 flush；原 DNAT helper 改为追加后 Flush，不支持由输入生成任意 NAT 策略。

- [ ] **4. 确认通过。** 执行 `PICKETX_KERNEL_TESTS=1 go test -tags=integration ./internal/validation/kerneltest -run "TestBridgeTopology" -count=1`；每个具名测试须通过，不得意外跳过，先前任务及 M0 回归继续通过。

- [ ] **5. 提交。** 仅暂存当前任务文件，提交信息为 `test(validation): prove bridge namespace and tuple scope`；最终验证记录保留失败及通过证据。

### 任务 3: 实际发出确认与组合自有事务

**文件：** Task 3 paths above; extend existing nftbackend integration tests for atomic rejected batch.

**消费接口：** fixture.Validate, existing nftables membership construction, Task 2 AppendDNAT through trusted integration assembly.

**产出接口：**

```go
// In nftbackend; preserve New, Apply and Close for existing M0 callers.
type ApplyState string
const (
    ApplyRejected ApplyState = "rejected"
    ApplyUnknown ApplyState = "unknown"
    ApplyConfirmed ApplyState = "confirmed"
)
type Omission struct { Domain string; Rule fixture.Rule; Reason string }
type Receipt struct {
    State ApplyState
    DispatchedAt, ConfirmedAt time.Time
    Emitted fixture.Snapshot
    Omitted []Omission
    OwnedAbsent bool
}
type ConfirmOptions struct {
    Now func() time.Time
    AppendFixture func(*nftables.Conn, *nftables.Table) error
}
func (b *Backend) ApplyConfirmed(ctx context.Context, s fixture.Snapshot, options ConfirmOptions) (Receipt, error)
func (b *Backend) RemoveOwnedConfirmed(ctx context.Context, now func() time.Time) (Receipt, error)
```

- [ ] **1. 编写失败测试。** 在当前包构造字面量夹具；下列代码为各测试的断言片段，局部变量由对应案例设置。固定 t0=`2026-10-15T10:00:00Z`，HTTPS end=t0+2s，SSH end=t0+4s，未来 start=t0+1s；纯单元测试使用假时钟。

```go
// TestConfirmedApplyFreshClockAndOmissions: clock read after backend lock; expired and <1ms remainder omitted with reason; bounded emitted rule retains original end; no zero-time conversion.
if len(receipt.Emitted.Allow) != 1 || len(receipt.Omitted) != 2 || !receipt.Emitted.Allow[0].ExpiresAt.Equal(end) { t.Fatal(receipt) }
// TestConfirmedApplyReadback: wrong family/prefix/port, missing required chain/set, extra allow/deny/member or wrong NAT fails; expected native-expired member may disappear; no required structural object may disappear.
if err == nil || receipt.State == ApplyConfirmed { t.Fatal(receipt, err) }
// TestConfirmedApplyBatchIsolation: append preparation error discards pending operations; retry does not replay them; known rejected transaction returns rejected; ambiguous send/readback returns unknown.
if residualQueuedOperations != 0 || retryContainsFailedAppend { t.Fatal(residualQueuedOperations) }
// TestRemoveOwnedConfirmed: delete only inet picketx, confirm absence even initially empty, preserve foreign tables; missing ACK/readback never confirms.
if receipt.State != ApplyConfirmed || !receipt.OwnedAbsent || !foreignTableExists { t.Fatal(receipt) }
```

- [ ] **2. 确认失败。** 执行 `go test ./internal/validation/nftbackend -run "Test(Confirmed|RemoveOwned)" -count=1`；缺失 API/行为应导致失败。集成场景若基线已通过，记录基线并增加字面量反例，不人为破坏正常代码。

- [ ] **3. 实现。** ApplyConfirmed 校验非空输入并独立持有结果。获得锁且完成可能阻塞的准备后，在排队/flush 前以新鲜 Now 从原绝对期限生成成员；准备跨过到期时丢弃并重建，发出后跨边界由协调器立即刷新。入站与受信固定 DNAT 追加共用一个事务，回执记录实际发出及到期/亚毫秒省略。成功要求 ACK 和对全部自有规则、对象、成员及 NAT 表达式的独立语义读回；仅容许预期成员原生到期，不容许结构缺失、额外权限或更宽/更长超时，考虑内核舍入。确定整批拒绝才报告 rejected；传输/ACK/读回不明确均 unknown。flush 前失败须丢弃所有排队消息。RemoveOwnedConfirmed 仅操作精确自有表并确认不存在，包括初始空状态。追加回调仅由可信夹具组装，不能来自策略 manifest。

- [ ] **4. 确认通过。** 执行 `go test ./internal/validation/nftbackend -run "Test(Confirmed|RemoveOwned)" -count=1`；每个具名测试须通过，不得意外跳过，先前任务及 M0 回归继续通过。

- [ ] **5. 提交。** 仅暂存当前任务文件，提交信息为 `feat(validation): confirm emitted atomic nftables batches`；最终验证记录保留失败及通过证据。

### 任务 4: 显式空 after 严格撤销

**文件：** Task 4 paths above; preserve standard fixture validation.

**消费接口：** existing Result, exactFilter and SelectWithdrawn(original/reply, zone, family, mark).

**产出接口：**

```go
// In ctbackend; Revoke remains unchanged for existing M0 callers.
func (b *Backend) RevokeEmpty(ctx context.Context, before fixture.Snapshot, mark uint32, now time.Time) (Result, error)
```

- [ ] **1. 编写失败测试。** 在当前包构造字面量夹具；下列代码为各测试的断言片段，局部变量由对应案例设置。固定 t0=`2026-10-15T10:00:00Z`，HTTPS end=t0+2s，SSH end=t0+4s，未来 start=t0+1s；纯单元测试使用假时钟。

```go
// TestRevokeEmptyValidatesOwnership: empty history or zero mark fails before dump/delete; fixture.Validate(empty) still fails; complete both family dumps before any deletion.
if err == nil || dumpCalls != 0 || deleteCalls != 0 { t.Fatal(err, dumpCalls, deleteCalls) }
// TestRevokeEmptyPreciseAndPartial: match original/reply, zone, family, mark; preserve foreign/unmarked/nonhistorical flows; interrupted initial dump deletes zero; partial deletion returns selected/deleted/remaining plus error.
if err == nil || result.Selected != 2 || result.Deleted != 1 || result.Remaining != 1 { t.Fatal(result, err) }
```

- [ ] **2. 确认失败。** 执行 `go test ./internal/validation/ctbackend -run "TestRevokeEmpty" -count=1`；缺失 API/行为应导致失败。集成场景若基线已通过，记录基线并增加字面量反例，不人为破坏正常代码。

- [ ] **3. 实现。** 抽取内部删除过程，不放宽现有 Revoke。RevokeEmpty 校验非空历史 before、规范化来源/入口及非零 mark，私有构造明确空 after。删除前完成两个地址族 dump；所有失败结果保留计数，未确认时 Remaining=-1。协调器仅在确认自有表不存在且空意图通过校验后调用。禁止按来源宽泛删除或 flush conntrack。

- [ ] **4. 确认通过。** 执行 `go test ./internal/validation/ctbackend -run "TestRevokeEmpty" -count=1`；每个具名测试须通过，不得意外跳过，先前任务及 M0 回归继续通过。

- [ ] **5. 提交。** 仅暂存当前任务文件，提交信息为 `feat(validation): revoke explicit empty owned state`；最终验证记录保留失败及通过证据。

### 任务 5: 串行协调器、累计历史与边界驱动

**文件：** Task 5 paths above; split files by state transition, history and driver responsibilities.

**消费接口：** Tasks 1/3/4; Effects implementation remains in integration tests and binds the same namespace handles, clock and fixed append helper.

**产出接口：**

```go
// In planbridge; effects are bound by the trusted topology adapter, not by input manifests.
type Timer interface { C() <-chan time.Time; Stop() bool }
type Clock interface { Now() time.Time; NewTimer(time.Duration) Timer }
type Effects interface {
    Verify(context.Context, Scope) error
    Apply(context.Context, fixture.Snapshot) (nftbackend.Receipt, error)
    Remove(context.Context) (nftbackend.Receipt, error)
    Revoke(context.Context, fixture.Snapshot, fixture.Snapshot, uint32, time.Time) (ctbackend.Result, error)
    RevokeEmpty(context.Context, fixture.Snapshot, uint32, time.Time) (ctbackend.Result, error)
}
type AppliedMember struct { Rule fixture.Rule; Origins []Origin }
type State struct {
    Generation uint64
    Status string // "initial", "confirmed", "cleanup-pending", "unknown"
    Receipt nftbackend.Receipt
    History []AppliedMember
    Cleanup ctbackend.Result
    NextChange time.Time
}
type Coordinator struct { /* serialized accepted plan, scope, cumulative history, state */ }
func NewCoordinator(scope Scope, effects Effects, clock Clock) (*Coordinator, error)
func (c *Coordinator) Replace(ctx context.Context, input networkplan.Input) (State, error)
func (c *Coordinator) Refresh(ctx context.Context) (State, error)
func (c *Coordinator) RetryCleanup(ctx context.Context) (State, error)
func (c *Coordinator) State() State
func (c *Coordinator) Run(ctx context.Context) error
```

- [ ] **1. 编写失败测试。** 在当前包构造字面量夹具；下列代码为各测试的断言片段，局部变量由对应案例设置。固定 t0=`2026-10-15T10:00:00Z`，HTTPS end=t0+2s，SSH end=t0+4s，未来 start=t0+1s；纯单元测试使用假时钟。

```go
// TestBridgeCoordinatorOrdering: Verify -> Apply/Remove -> confirmation -> record -> Revoke; rejected candidate/apply invokes no revoker and preserves accepted generation.
if !reflect.DeepEqual(events, []string{"verify", "apply", "confirm", "record", "revoke"}) { t.Fatal(events) }
// TestBridgeCoordinatorHistoryAcrossExpiry: emitted old allow survives expiry and later confirmed refresh; never-emitted future/revoked/omitted membership never enters history; native expiry during confirmation does not erase ACKed emission.
if len(state.History) != 1 || !state.History[0].Rule.ExpiresAt.Equal(end) { t.Fatal(state) }
// TestBridgeCoordinatorCleanupRetry: cleanup failure advances confirmed policy without rollback, keeps history and counters; explicit retry rematerializes and confirms fresh after then achieves Remaining == 0.
if failed.Status != "cleanup-pending" || failed.Generation != 2 || retried.Status != "confirmed" || retried.Cleanup.Remaining != 0 { t.Fatal(failed, retried) }
// TestBridgeCoordinatorUnknownIsTerminal: missing/ambiguous receipt or failed confirmation marks unknown; no cleanup or future application; preserve last known history; namespace disposal is harness responsibility.
if state.Status != "unknown" || cleanupCalls != 0 || effectsAfterFault != 0 { t.Fatal(state) }
// TestBridgeCoordinatorEmptyAndRemovedAgent: last Binding removal and validated complete Agent removal use exact owned-table removal; arbitrary unknown-Agent error cannot authorize it; empty-state retry works without new withdrawals; graceful calls no revoker.
if removeCalls != 1 || nonemptyApplyCalls != 0 || !state.Receipt.OwnedAbsent || state.Cleanup.Remaining != 0 { t.Fatal(state) }
// TestBridgeCoordinatorIdentityAndOwnership: wrong epoch or namespace fails before effects; receipt cannot add extra emitted rules or omit omission accounting; mutated State does not mutate coordinator.
if err == nil || effectCalls != 0 || !reflect.DeepEqual(before, fresh) { t.Fatal(err, fresh) }
// TestBridgeDriverFutureAndExpiry: fake clock installs future allow and deny and removes expired deny at their exact boundaries using current generation.
if !allowAppliedAt.Equal(start) || !denyAppliedAt.Equal(denyStart) || !denyRemovedAt.Equal(denyEnd) { t.Fatal(events) }
// TestBridgeDriverStaleCancellationAndDelay: stale wakeup after Replace never replays old plan; elapsed boundary during a blocked operation triggers immediate refresh; cancelled Run returns after joining in-flight work, with no later apply.
if oldGenerationReplayed || appliesAfterCancel != 0 || !refreshedImmediately { t.Fatal(events) }
```

- [ ] **2. 确认失败。** 执行 `go test -race ./internal/validation/planbridge -run "TestBridge(Coordinator|Driver)" -count=1`；缺失 API/行为应导致失败。集成场景若基线已通过，记录基线并增加字面量反例，不人为破坏正常代码。

- [ ] **3. 实现。** NewCoordinator 拒绝空 Effects/Clock 及非法 scope。Replace 在一个串行区内校验并应用；Refresh/RetryCleanup 使用已接受的不可变计划。仅匹配的确认回执/读回成功后、严格清理前推进 generation。将实际 Emitted 与全部来源关联，累计保存 allow/保护历史，不丢弃期限或来源；发出加省略必须完整对应提交成员，仅允许有期限依据的省略。strict before 取累计已确认历史，after 取新鲜物化。若准备/应用/确认跨过 NextChange，在任何严格删除前先用新鲜时刻重新物化、应用并确认，after 必须对应实际已确认的当前策略。空清理必须来自完整替换/撤回依据或保留的已确认空状态；可信初始空状态也须确认表不存在。回执或确认不明将夹具置 unknown，阻断后续应用/清理；确定拒绝保留旧状态。清理失败返回 cleanup-pending、错误及计数，但保留新计划；重试先按新鲜时间重新应用/确认。时钟/timer 注入验证旧事件与取消，实钟使用标准 time.Timer。Run 每次唤醒/操作后读取当前版本和 NextChange，跨边界立即刷新，取消时停止并等待在途工作结束。State 并发读取返回独立副本；每次副作用前再检查 ctx。

- [ ] **4. 确认通过。** 执行 `go test -race ./internal/validation/planbridge -run "TestBridge(Coordinator|Driver)" -count=1`；每个具名测试须通过，不得意外跳过，先前任务及 M0 回归继续通过。

- [ ] **5. 提交。** 仅暂存当前任务文件，提交信息为 `feat(validation): reconcile confirmed plans and historical cleanup`；最终验证记录保留失败及通过证据。

### 任务 6: 真实双栈契约场景

**文件：** Task 6 paths above; external integration package `planbridge_test`.

**消费接口：** Tasks 1-5, kerneltest OpenTCP/OpenUDP/SendTCP/Exchange, actual nftables/ctnetlink adapters.

**产出接口：** 本任务只增加集成契约/证据，不新增领域 API。

- [ ] **1. 编写失败测试。** 在当前包构造字面量夹具；下列代码为各测试的断言片段，局部变量由对应案例设置。固定 t0=`2026-10-15T10:00:00Z`，HTTPS end=t0+2s，SSH end=t0+4s，未来 start=t0+1s；纯单元测试使用假时钟。

使用下表确定的必需内核场景，断言真实报文/状态观察，不能只检查转换结果。

- [ ] **2. 确认失败。** 执行 `PICKETX_KERNEL_TESTS=1 go test -tags=integration ./internal/validation/planbridge -run "^TestBridge" -count=1 -timeout=8m`；缺失 API/行为应导致失败。集成场景若基线已通过，记录基线并增加字面量反例，不人为破坏正常代码。

- [ ] **3. 实现。** 按下表实现全部 16 项内核场景，以独立字面量期望对比 Core.Decide、生成成员及真实报文。每项覆盖两个地址族，注明 UDP/NAT 协议；拒绝测试前必须证明未过滤路径可达。普通/NAT、strict/graceful 使用独立拓扑。记录绝对边界、首次观察、应用/确认时间，必须在 1 秒内观察，不得调整期限掩盖问题。部分删除/dump 故障在真实连接及真实适配器边界注入，并明确标注；纯 mock 不满足内核部分。非法批次必须实际被内核拒绝，不能只注入返回错误。解除绑定后不承诺 socket 终止，此时检查表不存在和未管理对照仍可用。

- [ ] **4. 确认通过。** 执行 `PICKETX_KERNEL_TESTS=1 go test -tags=integration ./internal/validation/planbridge -run "^TestBridge" -count=1 -timeout=8m`；每个具名测试须通过，不得意外跳过，先前任务及 M0 回归继续通过。

- [ ] **5. 提交。** 仅暂存当前任务文件，提交信息为 `test(validation): prove core bridge with real dual-stack packets`；最终验证记录保留失败及通过证据。

#### 必需桥接内核场景

| Test | Design acceptance class | Mandatory observations |
| --- | --- | --- |
| `TestBridgeGiteaIndependentDeadlines` | Gitea-shaped independent limits | TCP/443 and TCP/22, exact sources plus /26 and /80 contributions, literal Core results and actual new connections agree |
| `TestBridgeDenyAndContributionRevocation` | Deny and revocation | deny wins; removing one Grant preserves wider independent Grant and other entry/source controls |
| `TestBridgeUDPCurrentPermission` | UDP | UDP/8002 authorized echo, revoke/strict deletion then old UDP cannot bypass current policy |
| `TestBridgeNativeExpiryGraceful` | Native expiry | no reapply; new TCP denied by end+1s while previously established graceful echo continues |
| `TestBridgeInactiveAndOmittedMembership` | Invalid or inactive input | future/revoked/expired absent, submillisecond omitted receipt, protected tuple denies |
| `TestBridgeFutureBoundaries` | Future boundaries | real driver activates future allow/deny, expires deny, observes within 1s; stale event/cancel control never reinstalls old permission |
| `TestBridgeReplayAndLateApply` | Replay/late application | near-expiry backend reconstruction, repeated application and actual lock/preparation delay retain original end; expired never becomes permanent |
| `TestBridgeHistoricalStrictRetry` | Historical strict cleanup | old established flow survives expiry plus an intervening confirmed apply, injected cleanup failure preserves history, retry actually selects/deletes it |
| `TestBridgeStrictPreservesIndependentFlows` | Strict preservation | overlap/other entry/source, unmarked and foreign mark flow survive; protected withdrawn new TCP and ACK/PSH remain denied with drop counter proof |
| `TestBridgeRejectedAtomicBatch` | Failed apply/confirmation | real invalid combined batch preserves old admission and NAT reachability; no revoker; separate readback-failure seam yields unknown |
| `TestBridgeInterruptedAndPartialCleanup` | Interrupted/partial cleanup | real established flows plus actual ctnetlink adapter with injected dump/delete failure: incomplete dump deletes zero; partial failure retains counts and explicit retry finishes |
| `TestBridgeBindingOwnership` | Binding ownership | delete one of two Bindings retains protection and remaining origin IDs; delete final Grant still protects |
| `TestBridgeEmptyBindingStrict` | Empty owned desired | last Binding removed: exact table absent and historical marked Remaining == 0, foreign table/mark and unmanaged port remain |
| `TestBridgeRemovedAgentStrict` | Empty owned desired | complete validated Agent removal produces withdrawals, table absence and zero owned remaining; explicit cleanup retry works after accepted empty state |
| `TestBridgeEmptyGraceful` | Empty owned desired | last Binding removed: confirmed table absence, no conntrack deletion, unmanaged controls available |
| `TestBridgeNATOriginalReply` | NAT original/reply tuples | VIP TCP/8000 withdrawn while 8001 remains; observe Server/9000 reply and original/reply/zone/mark precision |

设计共 16 类验收：上下文与来源记录仅做单元验证，其余 14 类映射为上述 16 个内核场景（空所有权拆为三项）。所有 `both` 类同时在任务 1/3/4/5 有单元断言。每项记录 IPv4/IPv6 子结果并由门禁强制核验；顶层通过而某地址族跳过/缺失，仍不能通过验收。

### 任务 7: 证据门禁与交付

**文件：** Task 7 paths above; add bridge result parsing and structured metadata without deleting M0 acceptance.

**消费接口：** existing Report/ScenarioResult, ReadEvents and CheckRequired plus Task 6 case list.

**产出接口：** 本任务只增加集成契约/证据，不新增领域 API。

- [ ] **1. 编写失败测试。** 在当前包构造字面量夹具；下列代码为各测试的断言片段，局部变量由对应案例设置。固定 t0=`2026-10-15T10:00:00Z`，HTTPS end=t0+2s，SSH end=t0+4s，未来 start=t0+1s；纯单元测试使用假时钟。

```go
// TestBridgeEvidenceRequiredAndDuplicate: missing/skipped/failed/duplicate bridge top-level scenario or wrong shared revision fails; duplicate terminal events for one case cannot be collapsed to pass.
if missingErr == nil || skippedErr == nil || duplicateErr == nil || wrongRevisionErr == nil { t.Fatal("gate accepted invalid evidence") }
// TestBridgeEvidenceFailureArtifact: malformed/missing event stream still produces report.json with errors and available evidence; old bridge output removed on rerun; unit pass cannot satisfy kernel gate.
if exitCode == 0 || !reportExists || len(reportErrors) == 0 || staleBridgeAccepted { t.Fatal(exitCode, reportErrors) }
```

- [ ] **2. 确认失败。** 执行 `go test ./internal/validation/report ./cmd/m0-report -count=1`；缺失 API/行为应导致失败。集成场景若基线已通过，记录基线并增加字面量反例，不人为破坏正常代码。

- [ ] **3. 实现。** 为 ScenarioResult/Report 向后兼容增加桥接结构化证据：共享版本/模块、范围身份、地址族、元组/来源、阶段/时间与清理结果。集成测试输出带 JSON 前缀的事件，人类日志仅用于诊断。ReadEvents 拒绝重复顶层终态，正常输出仍可累计；地址族结果独立记录并门禁。原 17 项名单追加恰好 16 项桥接场景。runner 构建桥接二进制，清除旧结果，各二进制在一次性 unshare --net 中执行，始终尝试生成报告；构建/初始化失败也记录当前元数据。缺失/损坏输入时 reporter 仍写入含可用观察及错误的 report.json，再非零退出。CI 用 always() 上传失败工件。检查 go list -deps 不链接 Echo/GORM/Controller/Web，记录实际模块与不可变校验和。双语文档仅报告实际内核/工具链结果及待完成生产边界。

- [ ] **4. 确认通过。** 执行 `go test ./internal/validation/report ./cmd/m0-report -count=1`；每个具名测试须通过，不得意外跳过，先前任务及 M0 回归继续通过。

- [ ] **5. 提交。** 仅暂存当前任务文件，提交信息为 `test(validation): gate bridge evidence and document results`；最终验证记录保留失败及通过证据。

## 完整验收与交接

- [ ] 在 PicketXD 执行全部单元/回归、race、vet、build 与格式检查：

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
test -z "$(gofmt -l cmd internal)"
go list -deps ./internal/validation/planbridge
go list -m all
PICKETX_KERNEL_TESTS=1 scripts/test-kernel.sh
```

- [ ] 必须通过全部 **33** 项内核场景（原 17 + 桥接 16），记录桥接双地址族子结果、当前共享不可变版本/校验和及失败工件。缺少权限时验收仍未完成，不能用 mock 或旧 M0 结果替代。
- [ ] 完成一次独立全分支评审，覆盖设计、全部任务、失败状态转换、到期/读回语义及精确删除过滤；集成前修复阻断问题并重新执行相关检查。
- [ ] 在 PicketXD 发布实现 PR，中央仓库同步双语状态，链接受测提交/运行/工件；沿用本会话直接执行，不采用逐任务并行实现。
- [ ] 实施前评审本书面计划。[基本功能路线图](../../BASIC_FUNCTION_ROADMAP.zh-CN.md)记录后续产品切片；完成 M1.3 不等于四/七层及管理端已完成。
