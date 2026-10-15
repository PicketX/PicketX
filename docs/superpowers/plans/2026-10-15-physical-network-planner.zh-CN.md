# M1.2 物理网络规划实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking. 用户已选择本会话直接执行，全部完成后做一次独立全分支评审；保留该选择。

> [English (default)](2026-10-15-physical-network-planner.md) | 简体中文

**目标：** 将逻辑授权与明确 Binding 编译为不可变、有明确范围的每 Agent 物理意图，并支持安全的完整替换。

**架构：** Core 负责权限解析；`pkg/networkplan` 负责物理规范化、部署、冲突和变更清单。两者均为显式传入时间的纯 Go 库。独立 PicketXD M0 桥接留在 M1.3。

**技术栈：** Go 1.27.0、标准库（`net/netip`、`time`、排序/复制辅助）、现有 `pkg/core`；go.mod/go.sum 不变。

**设计依据：** [已批准 M1.2 设计](../specs/2026-10-15-physical-network-planner-design.zh-CN.md)，已通过 [PR #5](https://github.com/PicketX/PicketX/pull/5) 合并。

**状态：** 计划经 PR #6 批准，从合并提交 `dc7158ad8583d3cfafe95c94bdef8d708bf31118` 开始执行。五项任务均已通过本地验证，发布前进行全分支评审。规划基线是 `315e0dd5617eed6cfc5055322d23b56378179cd2`；任务 1 已修复 M1.1 历史时间回归。

## 全局约束

1. 共享代码采用 Apache-2.0，不复制或导入 GPL Agent 实现。
2. 规划层仅依赖标准库及 Core；不添加产品依赖、HTTP/数据库/Casbin/Linux 导入、goroutine、系统时间、传输、Manifest 解码、持久化生命周期服务或 UI。
3. 明确 TCP 或 UDP 及 1–65535 的单个目的端口，协议无默认值。IPv4/IPv6 同等支持；拒绝 mapped/zoned 地址、DNS、端口范围、NAT64、隐式端口映射和 DNAT 后匹配。
4. 完整候选校验、不可变独立数据、确定性结果、独立绝对期限；不延期，不合并 Controller/Static 状态，不部分替换。
5. 输出保留明确可信上下文、原始目的观察契约和 graceful/strict 要求。旧期望清单不是安装证据；M1.2 不宣称流量、主机/容器或内核清理已验证。

## 评审重点

| 输入/失败模式 | 预期行为 | 所属测试 |
| --- | --- | --- |
| 已知空 Resource 与不存在的 Resource | 空选择诊断与报错区分，不变通配 | 任务 1 `TestNetworkViewResolvedMembership`；任务 3 `TestBindingNoMatchesAndOffline` |
| 公元 1 年之前的非零时间、省略开始时间 | 正确返回实际到期边界 | 任务 1 `TestNextBoundaryOmittedStartHistoricalClock` |
| 当前上下文边界隐藏原始定义变更 | 保留同类型贡献 ID 时拒绝改指向 | 任务 4 `TestRetargetRetainedContributions` |
| Agent 离线/移除或最后 Binding 消失 | 离线目标保留，移除时完整保留旧目标 | 任务 3 `TestBindingNoMatchesAndOffline`；任务 4 `TestWithdrawalInventory` |
| 到期 allow 仍可能对应 graceful 已建立连接 | 保留旧期望成员，不把它当实际安装历史 | 任务 4 `TestHistoricalFormerInventory`；任务 5 文档 |

## 文件及依赖映射

| 任务 | 文件 | 职责 |
| --- | --- | --- |
| 1 | `pkg/core/network.go`, `network_types.go`, `network_view_test.go`, `validity.go`, `validity_test.go` | 独立解析目录与历史时间回归 |
| 2 | `pkg/networkplan/types.go`, `network.go`, `context.go`, `network_test.go`, `context_test.go` | 物理原语与权威上下文边界 |
| 3 | `pkg/networkplan/binding.go`, `conflict.go`, `plan.go`, `binding_test.go`, `conflict_test.go`, `plan_test.go`; extend `types.go` | 部署、不可变清单与显式时间投影 |
| 4 | `pkg/networkplan/transition.go`, `transition_test.go`; extend `types.go` | 完整替换约束与旧目标清单 |
| 5 | `pkg/networkplan/conformance_test.go`, `example_test.go`, `doc.go`; root `README.md`, `README.zh-CN.md`, `docs/README.md`, `docs/README.zh-CN.md` | 公开用法、组合场景与准确里程碑状态 |

表中 1–4 行未重复目录的文件属于该行包目录。任务 1 修改现有 Core 文件并新增 `network_view_test.go`；任务 2 新增 `pkg/networkplan`；后续任务扩展该包。不涉及生成 Schema、新模块或 Agent 文件。

执行顺序：1 → 2 → 3 → 4 → 5。每个任务进行一次红/绿验证并独立提交，不逐任务派评审 Agent。全部完成后进行一次独立全分支评审，处理发现并重新验证后集成。执行开始时检查 AGENTS.md/worktree 状态，保留无关修改。

### 任务 1：Core 独立解析视图与时间前置修正

**消费接口：** 现有 `Compile(Snapshot) (*Engine, error)`、`(*Engine).Decide(Query) (Decision, error)`、`Validity.NextBoundary(time.Time) time.Time`。

**产出接口：** 位于 `network_types.go` / `network.go`，以下名称均属于 `core` 包：

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

Engine 保留规范化且独立持有的目录，包含 Resource 标签和空 Resource。`NetworkView` 不重新解析策略、不按时间过滤：保留未来、到期和已撤销成员，Grant/Deny 分域。Resource/入口按名称排序，贡献按 ID 排序；返回标签和切片均深复制。已编译空快照有效；nil/零值 Engine 报错。现有 `Compile`/`Decide` 语义不变。

只在 `validity.go` 修正已记录的 `NextBoundary` 省略值问题：即使传入非零时间早于公元 1 年，省略的零值 `NotBefore` 也不是未来生效点。不新增时间范围限制，不改变期限规则。

- [x] **1. 在本任务测试文件编写下列具名失败测试。** 断言片段使用测试内 fixture 变量；在所属测试包内构造，不复用其他包的私有测试 helper。

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

- [x] **2. 运行下方聚焦命令，观察预期行为失败。** 新 API 先补声明和最小返回错误的桩，使红色证据来自断言，不是缺少导入。

```bash
go test ./pkg/core -run 'Test(NetworkView|NextBoundaryOmittedStartHistoricalClock)' -count=1
```

- [x] **3. 在文件映射所列文件实现声明的接口，** 遵守本任务规则与已批准设计，不附带无关重构。

- [x] **4. 重跑聚焦命令和指定回归命令；** 要求 PASS 且退出码为 0。

```bash
go test ./pkg/core -run 'Test(NetworkView|NextBoundaryOmittedStartHistoricalClock)' -count=1
go test -race ./pkg/core -count=1
```

- [x] **5. 检查 `git diff --check` 和暂存差异后，只提交本任务文件，** 使用下方提交信息。

```bash
git commit -m "feat(core): expose owned resolved network view"
```

### 任务 2：物理原语与可信上下文解析

**消费接口：** `core.ParseSource(string) (netip.Prefix, error)`、`core.ValidateSource(netip.Prefix) error`，复用地址语法并包装目的地址角色的错误；来源和目的仍是独立角色。

**产出接口：** 位于 `types.go`、`network.go`、`context.go`，属于 `networkplan` 包：

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

字符串解析可掩码化 CIDR；强类型规范化要求输入已规范且有效。`CanonicalDestinations(nil)` 保留省略，非 nil 空列表报错。规范并集按地址族移除包含项、递归合并兄弟前缀，不枚举主机。`IntersectDestinations` 要求两侧非空有效集合；不相交时返回 nil 且无错误，随后由选择逻辑拒绝。

全部规范化结果独立持有集合。上下文要求非空 Agent/Profile/ID/网络空间、明确类别、原始观察阶段、连接处理及目的边界。`ResolveContext` 要求 ProfileRef/类别/阶段/处理一致且能力支持。可信事实须已是请求边界子集：事实更宽即拒绝，不通过裁剪后复用上下文身份。运行时解析器可用适当版本身份提供更窄事实。这一保守首版选择实现已批准的解析/收窄/拒绝规则，不改写可信事实。前缀本身不证明流量类别约束。

`NormalizeCapabilities(nil)` 返回 nil 且无错误，表示清单元数据未知；选中上下文使用 nil 能力时报错。未使用描述中的枚举也须校验，集合排序/去重/复制。实际物理协议/地址族在任务 3 交集后校验。非 nil 描述的空集合表示不支持任何对应值；`OriginalDestination=false` 不满足观察阶段。

- [x] **1. 在本任务测试文件编写下列具名失败测试。** 断言片段使用测试内 fixture 变量；在所属测试包内构造，不复用其他包的私有测试 helper。

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

- [x] **2. 运行下方聚焦命令，观察预期行为失败。** 新 API 先补声明和最小返回错误的桩，使红色证据来自断言，不是缺少导入。

```bash
go test ./pkg/networkplan -run 'Test(ParseDestinations|CanonicalDestinations|DestinationIntersection|NormalizeNetwork|ResolveContext|NormalizeExecutionContext|CapabilityValidation)' -count=1
```

- [x] **3. 在文件映射所列文件实现声明的接口，** 遵守本任务规则与已批准设计，不附带无关重构。

- [x] **4. 重跑聚焦命令和指定回归命令；** 要求 PASS 且退出码为 0。

```bash
go test ./pkg/networkplan -run 'Test(ParseDestinations|CanonicalDestinations|DestinationIntersection|NormalizeNetwork|ResolveContext|NormalizeExecutionContext|CapabilityValidation)' -count=1
go test ./pkg/core ./pkg/networkplan -count=1
```

- [x] **5. 检查 `git diff --check` 和暂存差异后，只提交本任务文件，** 使用下方提交信息。

```bash
git commit -m "feat(networkplan): normalize physical definitions and contexts"
```

### 任务 3：部署、冲突与不可变物化

**消费接口：** 任务 1 `NetworkView`、任务 2 规范化/解析器；Core 的 `NormalizeSelector`、`LabelSelector.Matches`、`SelectEndpointNames`、`Validity.ActiveAt`、`Validity.NextBoundary`。规划层不遍历原始 Grant scopes。

**产出接口：** 位于 `types.go` / `plan.go`；选择位于 `binding.go`，重叠校验位于 `conflict.go`：

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

`AuthorityID`、名称、Binding ID 在各自域内非空唯一。只调用一次 Core.Compile，保留其完整独立视图。每个逻辑入口（含未绑定入口）恰好一个规范网络定义。未使用行也须完整校验。`Requirements` 只要非空就报错并列出值，测试 `port-range`、`nat64`、`dns`、`post-dnat`、`unknown`；该字段不允许静默丢弃后续 wire 字段，未来解码器须单独拒绝未知字段。

每个 Binding 的 Resource/Agent 项列表均非空。保留 Resource selector 字段仅用于明确拒绝；支持精确 Resource 引用和 Core 入口选择。Agent 项只能有已知引用或非空 Core selector 之一。并集去重后取入口×Agent 笛卡尔积；其他维度为空也须校验显式引用。诊断 `no-agents-matched`、`no-entries-matched` 按字段元组排序。离线状态不改变部署；标签不继承。

可信上下文按 `(AgentRef, ProfileRef)` 索引；该 key 或 `(AgentRef, ID)` 重复即拒绝，即使值相等。上下文须引用已知 Agent。选中目标缺少上下文/能力即失败；未使用 nil 能力可作为未知清单元数据。解析请求上下文后取目的交集（入口 nil 地址使用边界），选中交集为空即报错并包含 Agent/Binding/入口。按实际协议/地址族加类别/阶段/处理校验能力，输出相同最小 `Required` 描述供 Adapter 再校验。

即使无活跃 Grant，冲突仍按可信 Agent/网络空间、原始目的、协议/端口和前缀重叠检查。不同逻辑入口的物理元组重叠即失败，包括同 Resource 的两个入口。完全相同入口/上下文/匹配/处理合并已排序 Binding ID。同入口的不同上下文身份重叠保守拒绝，标签/ID 不证明隔离。不相交元组或不同 Agent 可共存。任何错误均返回 nil Plan。

保留完整规范定义/视图/上下文目录供任务 4 使用，包括未绑定/未生效行。Plan 字段私有，全部读取方法返回独立数据。TargetKey 按 AgentRef/ContextID/ResourceName/EndpointName 排序；匹配按协议/端口/地址族/地址/前缀位数排序；成员再按来源地址族/地址/位数/ID 排序。诊断按声明字段排序，规范并集使等价顺序相同。`AgentRefs` 只列选中期望 Agent。已知未选 Agent 返回有效空 Inventory/Projection；未知 Agent、零时间、nil/零值 Plan 报错。输出 At 转 UTC 并去除单调时间数据。

Inventory 保留所有地址族相关贡献，包括已撤销/未生效/未来成员。Projection 保留保护目标，分别输出当前活跃且未撤销的 Allow/Deny 成员，按目标/匹配/类型域 ID 去重。Adapter 求值时活跃 deny 优先，领域清单仍保留重叠 allow 和 deny。NextChange 取相关未撤销成员中最早未来 Core 边界，含 deny 生效/到期；不转为 TTL。活跃 Grant 为零时保护仍存在。这些领域成员不规定每 Grant 一条内核规则。不添加定时器、应用 goroutine 或连接清理。

- [x] **1. 在本任务测试文件编写下列具名失败测试。** 断言片段使用测试内 fixture 变量；在所属测试包内构造，不复用其他包的私有测试 helper。

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

- [x] **2. 运行下方聚焦命令，观察预期行为失败。** 新 API 先补声明和最小返回错误的桩，使红色证据来自断言，不是缺少导入。

```bash
go test ./pkg/networkplan -run 'Test(Binding|ContextSelection|Projection|Protected|PhysicalConflict)' -count=1
```

- [x] **3. 在文件映射所列文件实现声明的接口，** 遵守本任务规则与已批准设计，不附带无关重构。

- [x] **4. 重跑聚焦命令和指定回归命令；** 要求 PASS 且退出码为 0。

```bash
go test ./pkg/networkplan -run 'Test(Binding|ContextSelection|Projection|Protected|PhysicalConflict)' -count=1
go test -race ./pkg/core ./pkg/networkplan -count=1
```

- [x] **5. 检查 `git diff --check` 和暂存差异后，只提交本任务文件，** 使用下方提交信息。

```bash
git commit -m "feat(networkplan): compile immutable per-agent intent"
```

### 任务 4：完整替换与撤回清单

**消费接口：** 任务 3 的 `Compile`、私有完整规范定义/视图/上下文目录、`Inventory` 和不可变 Plan。

**产出接口：** 位于 `types.go` / `transition.go`：

```go
type Withdrawal struct { Former TargetInventory; Reason string }
type Replacement struct { Plan *Plan; Withdrawals []Withdrawal }
func Replace(previous *Plan, input Input) (Replacement, error)
```

要求前一计划非 nil 且有效，AuthorityID 相同；`Compile` 是独立可信初始化路径，不用于接管未知已安装规则。完整编译候选后才接受。任意失败返回零值 Replacement，不修改旧计划、不延长其期限。

对保留 EntryRef 比较完整规范网络定义（协议、端口、地址是否省略及并集），不能只比较当前可见交集。省略地址与当前等价显式列表仍是不同意图。定义变化时，若旧 Grant ID 或 Deny ID 在同一类型域仍解析到该入口则拒绝，含已撤销/到期/未绑定成员；错误列出入口和保留的类型域 ID。等价规范并集不算变化。新 ID 重发 fixture 复制原绝对期限，规划层从不生成延期。输入没有旧→新审批映射，因此无法证明新 ID 对应某个旧审批；重发资格及原期限保留证明由权威发布方负责，不能虚构 ID 推断规则。

比较所有共享 `(AgentRef,ID)` 的新旧可信上下文行，含未使用行。该身份下 NetworkSpaceID/类别/阶段/规范边界/处理/ProfileRef 不可变。新 ID 仍经过常规校验、冲突和撤回处理。新增入口/标签变化保留 Core 的 whole/label 动态语义，名称选择固定。名称引用缺失即失败；移除/重命名须配套移除或显式重发具名贡献。不实现 tombstone、持久 UID、重启接管或 Controller/Static 合并。

按 TargetKey 比较期望保护。旧 key 消失产生 `Reason="removed"`；同 key 匹配变化产生 `Reason="match-changed"`。每项携带完整独立 `Former`（匹配/上下文/Binding ID/全部相关贡献），按 TargetKey 排序，即使 Agent 已移除。少一个 Binding 但仍有其他贡献，或仅权限成员变化，都不属于保护撤回；新 Inventory 将成员变化提供给后续 Adapter。这是重新核对完整旧目标的意图，不允许删除其全部旧连接：实际严格清理须比较已安装/已观察历史所有权与已确认新策略，并保留仍允许的流。

- [x] **1. 在本任务测试文件编写下列具名失败测试。** 断言片段使用测试内 fixture 变量；在所属测试包内构造，不复用其他包的私有测试 helper。

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

- [x] **2. 运行下方聚焦命令，观察预期行为失败。** 新 API 先补声明和最小返回错误的桩，使红色证据来自断言，不是缺少导入。

```bash
go test ./pkg/networkplan -run 'Test(Replace|Retarget|ContextIdentity|Withdrawal|HistoricalFormer|EntryLifecycle)' -count=1
```

- [x] **3. 在文件映射所列文件实现声明的接口，** 遵守本任务规则与已批准设计，不附带无关重构。

- [x] **4. 重跑聚焦命令和指定回归命令；** 要求 PASS 且退出码为 0。

```bash
go test ./pkg/networkplan -run 'Test(Replace|Retarget|ContextIdentity|Withdrawal|HistoricalFormer|EntryLifecycle)' -count=1
go test -race ./pkg/core ./pkg/networkplan -count=1
```

- [x] **5. 检查 `git diff --check` 和暂存差异后，只提交本任务文件，** 使用下方提交信息。

```bash
git commit -m "feat(networkplan): validate replacements and retain withdrawals"
```

### 任务 5：公开契约验证、文档与最终验证

**消费接口：** 任务 1–4 公开 API；不产出运行时新接口。`ExampleCompile` 使用外部包 `networkplan_test` 和明确固定时间。fixture helper 在 `conformance_test.go` 内构造，期望值为字面量，不能再次调用规划器计算。

在 `doc.go` 和双语 README 说明：权威输入/可信事实边界、nil 与空地址、不支持要求、新 ID 重发边界、不可变独立结果、通过 NextChange 调度、期望与安装历史、明确空所有权清理、保护与未管理流量、保守上下文冲突。代码及验证通过后才标记 M1.2 已实现；M1.3 仍须独立桥接设计和真实特权流量证据。桥接须保留 guard/原始元组、调度未来生效、先应用确认再严格范围清理、使用实际安装历史 before 状态，明确处理 M0 空 Protected 限制。纯测试不证明这些运行时结果。

- [x] **1. 在本任务测试文件编写下列具名失败测试。** 断言片段使用测试内 fixture 变量；在所属测试包内构造，不复用其他包的私有测试 helper。

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

- [x] **2. 运行下方聚焦命令，观察预期行为失败。** 新 API 先补声明和最小返回错误的桩，使红色证据来自断言，不是缺少导入。

```bash
go test ./pkg/networkplan -run 'Test(GiteaConformance|ProviderNormalizedEquivalence|PlanOwnershipAndConcurrentReads)|ExampleCompile' -count=1
```

- [x] **3. 在文件映射所列文件实现声明的接口，** 遵守本任务规则与已批准设计，不附带无关重构。

- [x] **4. 重跑聚焦命令和指定回归命令；** 要求 PASS 且退出码为 0。

```bash
go test ./pkg/networkplan -run 'Test(GiteaConformance|ProviderNormalizedEquivalence|PlanOwnershipAndConcurrentReads)|ExampleCompile' -count=1
go test -race ./... -count=1
```

- [x] **5. 检查 `git diff --check` 和暂存差异后，只提交本任务文件，** 使用下方提交信息。

```bash
git commit -m "test(networkplan): add gitea conformance and public usage"
```

## 最终验收与交接

在仓库根目录用 Go 1.27.0 执行。工具链未在 PATH 时前置其 bin 目录，不通过改 go.mod 下载其他版本。

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

要求 gofmt 无输出、测试/构建/vet 全部退出 0，非标准依赖仅 `github.com/PicketX/PicketX/pkg/core` 和 `github.com/PicketX/PicketX/pkg/networkplan`。go.mod/go.sum 无差异。最终 PR head 的现有 Go CI（格式/vet/race/build）须实际通过，现有 Web CI 也须保持绿色；记录实际 run/head ID，不能只引用工作流配置。不添加跳过、不声称已做内核测试、不引入无关依赖。

设计 17 行验收均映射到任务 1–5：校验/地址/上下文（1–3）、部署/冲突/时间/来源记录（3）、生命周期/撤回（4）、数据所有权/提供方等价/Gitea（5）。自查覆盖设计全部八节、跨任务类型及五行评审重点；M1.2 内无遗漏留给未定义任务，明确排除的 M1.3 运行时义务仍如上保留。

## 执行记录

任务 1–4 均在最小 API 桩上的断言级失败后实现；历史时间回归在实际 M1.1 代码上先失败，再修正。任务 5 为已实现行为增加跨层契约验证与公开示例，首次运行即通过，不声称人为制造过失败。这是唯一执行裁定；代价是单元实现后的验收测试可能遗漏问题，通过字面量期望和最终独立评审补足。

Go 1.27.0：test、race、vet、build 均通过。32 个计划具名测试全部执行，Core（35）、规划层（27）、Controller（3）合计 65 个顶层测试/示例通过，没有跳过测试；命令入口包没有测试文件。依赖检查的非标准库输出仅 Core/规划层，go.mod/go.sum 不变。不声称已有数据包/内核验证；独立评审及最终提交 CI 证据在可用后记录到实现 PR。
