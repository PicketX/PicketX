# M1.1 共享网络 Core 实施计划

> [English (default)](2026-10-15-shared-network-core.md) | 简体中文
>
> **执行代理：** 必须使用 superpowers:executing-plans，沿用用户已选择的直接执行方式，逐任务落实复选步骤；最后对整个分支进行一次独立评审。

**目标：** 实现可复用、确定性的 Go Core，计算网络来源到逻辑入口的权限并集、独立期限与撤销、显式拒绝及解释。

**架构：** 标准库包 `pkg/core` 完整校验并复制规范化输入，生成不可变引擎。调用方提供时间和可信的逻辑入口身份；Controller/Agent 后续复用语义，本增量不推断目的地址、Binding 或实际生效状态。

**技术栈：** 现有 Go 1.27.0 模块 `github.com/PicketX/PicketX`，Apache-2.0；不新增模块依赖。

**设计依据：** [已批准的 M1.1 设计](../specs/2026-10-15-shared-network-core-design.zh-CN.md)。

**状态：** 已执行、验收并合并 [PR #4](https://github.com/PicketX/PicketX/pull/4)；[主干 CI](https://github.com/PicketX/PicketX/actions/runs/38661859967)通过。计划设计基线为 `ec23f7467d66ee9fc5655e2ed98c6e3ba89e019c`；执行从计划合并提交 `e64ccd36f0ddd7362e551dcfed593941df34e862` 开始。

## 全局约束

- Core 仅使用 Go 标准库；不引入 HTTP/ORM/Casbin/Linux、不启动 goroutine、不操作网络和数据库、不调用 `time.Now`。
- 仅处理网络逻辑查询；入口类型默认 `network`，动作默认 `connect`；不支持的要求必须报错，不能丢弃。
- 普通 IPv4 规范化为 /32、IPv6 为 /128；显式 CIDR 掩码化。掩码化前拒绝非法输入、域名、带 zone 或 IPv4-mapped 地址。
- 直接传入的前缀必须有效且已规范化；所有匹配来源的贡献参与计算，不能只取最长前缀。
- 入口选择为整个 Resource、非空名称列表或非空标签选择器；无隐式 Resource/Agent/global ALL、标签继承或把非法空值当作省略。
- 明确 bounded/permanent 模式；零模式非法；生效含起点、失效不含终点；调用方提供非零时间；编译和重启不能延长绝对期限。
- 完整校验候选，包括失效和撤销数据；确保不可变所有权与排序去重的确定性解释。
- 每条贡献保留自己的来源与期限；`NextChange` 表示重新计算，不表示延期或已生效。
- 不涉及外部协议、UID 持久化与生命周期、广泛 Resource 选择器、Binding/物理规划、传输、审批/存储/UI 或续期集成；不得把 GPL Agent 代码复制到 Apache 包。

## 评审关注点

- 显式空值：空选择、动作和 Exists values 必须失败，不能扩大权限；任务 1–2。
- mapped/zoned 来源与跨地址族重叠：掩码化前拒绝非规范形式，保持地址族隔离；任务 1–2。
- 未生效的非法贡献：已过期、已撤销、选择器无匹配的数据仍必须使候选失败；任务 2。
- 调用方或返回值修改：不能改变已编译策略和后续决策；任务 2–3。
- 未来生效、精确失效与重放：保留独立期限，正确返回最近的相关变化；任务 1–3。

## 文件地图与依赖

| 文件 | 职责 |
| --- | --- |
| `pkg/core/source.go`、`source_test.go` | 网络来源解析与规范校验 |
| `pkg/core/selector.go`、`selector_test.go` | 标签条件校验与匹配 |
| `pkg/core/validity.go`、`validity_test.go` | 明确的期限模式与时间边界 |
| `pkg/core/selection.go`、`selection_test.go` | 网络 Endpoint 规范化与整体/名称/标签选择 |
| `pkg/core/network_types.go`、`network.go`、`network_test.go` | Snapshot/查询/结果类型、完整校验和不可变决策 |
| `pkg/core/conformance_test.go`、`example_test.go`、`doc.go` | Gitea 跨场景契约、公开示例与集成边界 |
| `.github/workflows/ci.yml`、`README.md`、`README.zh-CN.md`、`docs/README.md`、`docs/README.zh-CN.md` | 格式检查纳入 pkg，记录真实实现范围 |

任务 2 消费任务 1 原语，任务 3 消费引擎与原语；现有 Echo 存活检查和模块依赖保持不变。

---

### 任务 1：来源、选择与期限原语

**文件：** 创建地图中的四组原语与测试。新增 Go 源码/测试包含 `// SPDX-License-Identifier: Apache-2.0`。

**接口：**

- `ParseSource(text string) (netip.Prefix, error)`；`ValidateSource(prefix netip.Prefix) error`。显式 CIDR 先解析，先拒绝 mapped/zoned 再掩码化；不回退处理空白或域名。直接前缀必须已经规范化。
- `SelectorOperator string`：仅 `In`、`NotIn`、`Exists`、`DoesNotExist`；`Requirement{Key string, Operator SelectorOperator, Values []string}`；`LabelSelector{MatchLabels map[string]string, MatchExpressions []Requirement}`。
- `NormalizeSelector(LabelSelector) (LabelSelector, error)` 复制 map/slice，拒绝空选择器/空 key/未知操作符；In/NotIn 必须有非空 values，Exists/DoesNotExist 的 values 必须为 **nil**。集合重复值可去重；相等匹配区分已有空字符串值和不存在的 key。
- `(LabelSelector).Matches(labels map[string]string) bool` 按 AND 计算校验过的条件，非法选择器返回 false；不修改或继承标签。
- `LifetimeMode uint8`：`LifetimeBounded=1`、`LifetimePermanent=2`；`Validity{Mode LifetimeMode, NotBefore, ExpiresAt time.Time}`；`NormalizeValidity(Validity) (Validity, error)` 返回 UTC 时刻。零模式失败；bounded 必须有终点；permanent 禁止终点；提供起点时必须早于 bounded 终点。
- `(Validity).ActiveAt(at time.Time) bool`；`(Validity).NextBoundary(at time.Time) time.Time`。非法期限/零查询时间返回 false/zero；边界为严格晚于查询时刻的最近起点或终点，zero 表示没有。
- `Endpoint{Name, Type string, Labels map[string]string}`；`NormalizeEndpoint(Endpoint) (Endpoint, error)` 要求名称；空类型默认 network，其他类型失败；复制标签，不继承。
- `SelectionMode uint8`：omitted=0、all=1、names=2、labels=3；`EndpointSelection{Mode SelectionMode, Names []string, Selector *LabelSelector}`。
- `NormalizeSelection(EndpointSelection) (EndpointSelection, error)` 仅将完全零值规范化为 all；omitted/all 携带 names/selector 非法。names 要求非空列表、非空名称和 nil selector；labels 要求有效非 nil selector、nil names。显式非 nil 空 slice 或混用模式失败。
- `SelectEndpointNames(endpoints []Endpoint, selection EndpointSelection) ([]string, error)` 校验入口与选择，拒绝重复定义/不存在的指定名称，返回排序去重名称；有效标签无匹配或 all 遇到空目录时返回 nil，不能扩展匹配。空目录不是通配符。

- [x] **先写来源测试。** `TestParseSource`：`192.0.2.10`→`192.0.2.10/32`，`2001:db8::10`→`/128`，`192.0.2.10/24`→`192.0.2.0/24`；拒绝域名、`%eth0`、mapped 地址和 `::ffff:192.0.2.10/64`。`TestValidateSource`：非法/未掩码化/mapped 前缀失败。
- [x] **先写选择器与入口选择测试。** `TestSelectorOperators`：缺失 key 的 NotIn 为 true，追加 Exists 后为 false；In 内部或、条件间 AND；DoesNotExist；已有 `access:""` 与缺失 key。`TestSelectorRejectsInvalidValues`：nil/空选择器、空 key、未知操作符、空 In/NotIn、Exists/DoesNotExist 非 nil 空 values 失败。`TestEndpointSelection`：零值/显式 all 等价，新增入口后 named 不变，selector 跟随标签，无匹配返回 nil。`TestSelectionRejectsInvalidInputs`：空/混合模式、重复入口、缺失名称、不支持类型失败。
- [x] **先写期限测试。** `TestValidityBoundaries`：起点 `2026-10-15T08:00:00Z`，终点 `10:00Z`；起点前 false，起点 true，终点−1ns true，精确终点 false；permanent 在 12:00Z 后仍有效。`TestValidityRejectsInvalidModes`：零/未知模式、bounded 无终点、permanent 有终点、终点≤起点、零查询时间不能允许。`TestNextBoundary`：起点前→起点，起点→终点，终点→zero；等价时区规范化到 UTC，时刻不变。
- [x] **运行 RED。** 仅补齐编译所需类型和签名，行为暂缺；运行 `go test ./pkg/core -run 'Test(ParseSource|ValidateSource|Selector|EndpointSelection|SelectionRejects|Validity|NextBoundary)' -count=1`。预期规范化、空值与时间边界的行为断言失败，不是 import/依赖失败。
- [x] **实现原语。** 使用 net/netip 与标准 maps/slices/time；先保留输入存在性，再规范化；返回规范对象不能别名共享输入。
- [x] **运行 GREEN。** `go test ./...`、`go vet ./...`、`go build ./...`、`git diff --check`；预期全部退出 0，原语断言通过。
- [x] **提交。** `feat(core): add source selection and validity primitives`。

### 任务 2：不可变的 Grant/限制决策

**文件：** 创建 `pkg/core/network_types.go`、`network.go`、`network_test.go`；消费任务 1 的接口，不改名或改签名。

**接口：**

- `Resource{Name string, Labels map[string]string, Endpoints []Endpoint}`；允许空目录和无入口 Resource；重复/空 Resource 名称及重复/空入口名称失败。
- `Scope{ResourceName string, Endpoints EndpointSelection, Actions []string}`。nil Actions 默认 connect，非 nil 空列表、空/不支持动作失败，重复 connect 去重；引用即使在未生效贡献中也必须存在；Resource 标签不参与 Endpoint 匹配。
- `Grant{ID string, Source netip.Prefix, Scopes []Scope, Validity Validity, Revoked bool, SubjectType string, Conditions []string, Renewal string}`；`Deny` 是同字段的独立类型。这些是演进中的规范化 Go 输入，不是外部协议字段。
- SubjectType 空/source 支持，其余失败；Conditions 长度必须为 0，任何条件均因不支持而失败；Renewal 空/disabled 接受，其余失败；不添加条件或续期解释器。
- `Snapshot{Resources []Resource, Grants []Grant, Denies []Deny}`；ID 非空且在各贡献类型内唯一；同一 ID 可分别存在于 Grant 与 Deny 域。
- `Compile(Snapshot) (*Engine, error)` 完整校验后返回非 nil 不可变引擎，复制并规范化解析入口，不使用时间；非法输入返回 nil 引擎与 error。删除 Grant 不删除目录；非法替换不修改旧引擎。
- `Query{Source netip.Addr, ResourceName, EndpointName, Action string, At time.Time}`；空 Action 默认 connect；非法/mapped/zoned 来源、未知入口/动作、零时间返回 error。
- `DecisionReason string`：`allowed`、`no-active-grant`、`explicit-deny`；`Contribution{GrantID string, Source netip.Prefix, Validity Validity}`；`Decision{Allowed bool, Reason DecisionReason, GrantIDs, DenyIDs []string, Contributions []Contribution, NextChange time.Time}`。
- `(*Engine).Decide(Query) (Decision, error)` 收集当前非撤销匹配贡献；deny 获胜仍保留匹配 allow 解释。ID 排序去重，Contributions 按 GrantID 排序；无匹配返回 nil ID/贡献列表，无未来边界返回 zero NextChange。结果不能暴露引擎所有的数据；nil/未初始化引擎返回 error，不能 panic 或允许。
- NextChange 取非撤销、来源/范围/动作匹配的 Grant/Deny 的严格未来最小边界，包含未来起点；已过期、无匹配、不相关、已撤销贡献不能虚构边界；不能视为新到期时间或已生效字段。

- [x] **先写完整校验测试。** `TestCompileRejectsInvalidSnapshot`：重复名称/ID（各域内重复失败，同一 ID 的 Grant/Deny 配对有效）、缺失 Resource/Endpoint、空 scopes/actions、非法来源/期限均返回 nil 引擎及 error。`TestCompileValidatesInactiveContributions`：撤销、过期、选择器无匹配贡献的非法引用/要求仍失败。`TestCompileRejectsUnsupportedRequirements`：Grant 与 Deny 中的 user/service、http、read、任何 condition、unique-only renewal 均失败。`TestCompileDefaults`：省略/显式 network、connect、disabled 决策等价；空目录无可查询入口；有效非空标签选择器无匹配时不贡献 allow。
- [x] **先写决策测试。** `TestDecisionPermissionUnion`：`192.0.2.10` 同时匹配 `a-host` /32 与 `b-prefix` /24，解释含两个 ID 及原始来源/期限；其他来源仅匹配 b-prefix；IPv6 等价，地址族不串联。`TestDecisionDenyPrecedence`：有效 deny 覆盖两者，未来/失效/撤销 deny 不覆盖，NotIn 不能取消另一 Grant。`TestDecisionRevocation`：撤销 a-host 保留 b-prefix，全部撤销 no-active-grant，永久 Grant 撤销也不允许。`TestDecisionNextChange`：allow 起点 08:30Z、deny 起点 09:00Z、allow 终点 10:00Z、deny 终点 09:30Z；08:00Z→08:30Z，08:30Z→09:00Z，09:00Z deny 获胜且 NextChange 为 09:30Z，09:30Z 恢复 allow 且 NextChange 为 10:00Z。`TestDecisionRejectsInvalidQuery` 覆盖所有查询错误和 nil/零引擎。
- [x] **先写所有权、确定性与并发测试。** `TestCompileOwnsInputs` 修改原始标签、入口/选择名称、selector values、scopes、Grant ID/来源/期限/撤销状态和 Deny，旧决策不变。`TestDecisionOwnsResults` 修改返回 ID/贡献，再次查询不变。`TestDecisionDeterministic` 重排等价目录/贡献/scopes/selector values，排序解释相同。`TestDecisionConcurrentRead` 多 goroutine 查询同一引擎并仅修改各自返回值，固定结果保持一致、race 无报告。`TestInvalidReplacementPreservesEngine` 拒绝非法新候选，稍后旧引擎仍按原期限失效。
- [x] **运行 RED。** 仅创建可编译类型/签名；`go test ./pkg/core -run 'Test(Compile|Decision|InvalidReplacement)' -count=1`。预期权限/原因/解释/校验的行为断言失败，无 import 错误或 nil-pointer 测试错误。
- [x] **实现 Compile/Decide。** 在自有目录解析精确 Resource 和入口成员，过滤前完整校验未生效数据；可按入口建索引，但计算所有匹配来源贡献，各边界独立。无最长前缀独占、全局缓存、goroutine 或副作用。
- [x] **运行 GREEN。** `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`git diff --check`；预期退出 0，所有权与并发断言稳定。
- [x] **提交。** `feat(core): evaluate immutable network grant contributions`。

### 任务 3：Gitea 契约、公开示例与真实覆盖

**文件：** 创建 `pkg/core/conformance_test.go`、`example_test.go`、`doc.go`；更新 CI 格式检查和双语根 README/文档索引；消费任务 1–2 接口，不改变签名。

**接口：**

- `ExampleCompile` 演示 ParseSource→Compile→Decide 查询 Gitea HTTPS 逻辑入口；使用固定输出，无 HTTP/数据库/Agent 依赖。
- `doc.go` 说明来源与申请人分离、可信逻辑身份、显式时间、不可变快照替换、nil/空输入差异、不支持要求必须失败、Go API 演进且不是外部协议承诺。
- CI 格式命令由 `gofmt -l cmd internal` 改为 `gofmt -l cmd internal pkg`；保留 vet/race/build 验收。

- [x] **先写跨场景测试。** `TestGiteaIndependentDeadlines`：dev-gitea 的 https/ssh、来源 192.0.2.10；HTTPS 10:00Z 失效，SSH 12:00Z 失效；09:00Z 两者允许，10:00Z 仅 SSH，12:00Z 全部失效，IPv6 重复。追加 11:00Z 失效的 HTTPS prefix Grant，HTTPS 维持至 11:00Z，SSH 的 12:00Z 期限不变；10:00Z 后解释不含已失效 HTTPS Grant ID。
- [x] **先写更新与契约测试。** `TestGiteaOverlapAndRevocation`：删除/撤销 host 贡献，prefix 保留；显式 deny 仅拒绝 HTTPS，SSH 保留。`TestCatalogSelectionEvolution`：追加 git-admin/重标入口后编译替换；whole 扩展、named 固定、selector 跟随入口标签，旧引擎不变。`TestEquivalentProviderInputs`：独立构造 Controller/static 风格规范输入，省略/显式默认值及等价时区期限产生相同决策；`TestRecompilePreservesDeadlines` 两次编译同一绝对期限快照，在到期前和精确终点查询两引擎，要求期限/决策完全一致，不能生成新的持续时间；不声称已实现 provider/decoder 或内核桥接。
- [x] **运行新测试与示例。** `go test ./pkg/core -run 'Test(Gitea|CatalogSelection|EquivalentProvider|RecompilePreserves)|ExampleCompile' -count=1`；已有引擎应满足契约。新增未覆盖行为先观察断言失败再最小修复；任务 2 已实现的契约无需故意制造失败。确认最长前缀独占、Resource 共用最长 TTL、含到期终点等错误会被固定断言捕获。
- [x] **补充公开说明与格式覆盖。** 双语 README 记录已实现逻辑 Core 与待实现 M1 规划/工作流；保留 M0 实测边界和外部 Schema 待设计状态；不能将逻辑允许表述为数据包/应用实际放行。
- [x] **运行完整验收。** `test -z "$(gofmt -l cmd internal pkg)"`、`go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`git diff --check`；预期退出 0。`go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./pkg/core` 的非标准库项只能是 `github.com/PicketX/PicketX/pkg/core`；go.mod/go.sum 不变。现有 Go CI 覆盖 pkg/core，验证真实结果，不能只核对配置。
- [x] **提交。** `test(core): verify Gitea logical authorization contract`。
- [x] **最终评审。** 使用 requesting-code-review 派发一次新的整个分支评审；Important/Critical 项集中修复并测试，延期 Minor 记入记录；提交实现 PR，附真实 unit/race/build/依赖证据。没有用户的集成决定，不合并。

## 固定断言样式

以下片段分别用于任务 1–3 的既定测试，fixture 辅助构造函数留在测试文件中，不是新增公开 API：

```go
// TestParseSource
p, err := ParseSource("192.0.2.10/24")
if err != nil || p.String() != "192.0.2.0/24" { t.Fatalf("prefix=%v err=%v", p, err) }
// TestValidityBoundaries: v 已规范化，end 是固定终点。
if !v.ActiveAt(end.Add(-time.Nanosecond)) || v.ActiveAt(end) { t.Fatal("exclusive expiry") }
// TestDecisionPermissionUnion: engine 含 a-host /32 和 b-prefix /24，q 查询 .10。
d, err := engine.Decide(q)
if err != nil || !d.Allowed || !reflect.DeepEqual(d.GrantIDs, []string{"a-host", "b-prefix"}) { t.Fatalf("decision=%+v err=%v", d, err) }
// TestGiteaIndependentDeadlines: q 查询 https，endHTTPS 为 10:00Z，且没有额外 HTTPS prefix Grant。
q.At = endHTTPS
https, err := engine.Decide(q)
if err != nil || https.Allowed || https.Reason != "no-active-grant" { t.Fatalf("https=%+v err=%v", https, err) }
q.EndpointName = "ssh"
ssh, err := engine.Decide(q)
if err != nil || !ssh.Allowed { t.Fatalf("ssh=%+v err=%v", ssh, err) }
```

## 完成条件与下一阶段

全部命名测试与公开示例执行、全量 Go unit/race/vet/build 通过、包依赖保持标准库且最终评审问题处理后，才验收 M1.1。刻意的接口/设计解释写入执行记录。

本计划交付逻辑 Core。下一份聚焦设计是物理 Endpoint/Binding 规划和 Core-to-M0 契约验证：先解决目的地址/实施范围、所有权冲突与生命周期编辑，再集成 Agent。持久化/审批、可靠发布、传输/状态及 Gitea 门户/CLI 在后续增量完成。

## 执行记录

- 三个实现提交分别交付原语、不可变引擎、Gitea 契约；9 项原语与 14 项引擎测试先观察行为 RED，再实现 GREEN。5 项跨场景测试和公开示例验证已有引擎；实际 first-Grant-only/含到期终点的临时变异均被 Gitea 固定断言拒绝，之后恢复正确代码。
- 全部 29 项 Core 测试/示例及 3 项现有 Controller 测试通过，没有跳过测试；命令包本身没有测试。Go unit/race/vet/build、格式及 diff 检查通过；Web npm ci/typecheck/lint/build 通过。Core 非标准库依赖只有自身，go.mod/go.sum 不变。
- 独立整分支评审：0 Critical、0 Important；评审者独立执行非缓存 race/vet/build/格式/依赖检查。远端 [CI #38608827993](https://github.com/PicketX/PicketX/actions/runs/38608827993) 已在提交 `03efd46722dcf245f354ff56e96a3a03d8ec2b29` 成功：Go/Web 两个任务通过，Go 日志确认实际执行 pkg/core race 测试。后续验证记录文档变更不修改运行时或测试源码。
- 延期 Minor：`Validity.NextBoundary` 对早于公元 1 年的非零查询时间，可能将未提供的 NotBefore 作为未来边界返回 zero，遮蔽实际到期边界；正常运行时间不受影响。后续可加入 IsZero 检查与历史时间回归测试。
- 执行调整：任务 3 的最终评审/发布步骤置于全分支完成关卡，遵循 executing-plans；影响仅流程顺序，评审仍先于发布。
- 范围裁定：物理目的地址/Binding/内核实施留后续，因此逻辑允许不证明数据包放行；持久化/审批/可靠发布/实际状态留后续，因此尚无完整浏览器到 Agent 工作流；外部 Schema/解码留后续，因此不承诺稳定协议；规模容量未作承诺，须实测后才能给出部署容量保证。这些维持已批准边界。
