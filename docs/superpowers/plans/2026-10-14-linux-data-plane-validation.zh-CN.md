# Linux 数据面验证实施计划

> **面向执行 Agent：**建议当前会话直接执行，逐任务使用 superpowers:executing-plans；只有用户选择委派时才使用 superpowers:subagent-driven-development。步骤以复选框跟踪。

**目标：**建立可重复执行的 M0 Linux 验证套件，为 nftables 执行和 conntrack 定向严格撤销提供真实证据。

**架构：**在 PicketXD 实现测试内部 Fixture 与 Adapter，用隔离网络命名空间中的真实流量验证。快照代表预期有效权限，不是生产 Grant Compiler 或公开配置 Schema。

**技术栈：**Go 1.27.0、google/nftables v0.3.0、初始 vishvananda/netlink v1.3.1、Linux 网络命名空间、GitHub Actions。

**设计：**[M0 设计](../specs/2026-10-14-linux-data-plane-validation.zh-CN.md)

**状态：**待评审，任务尚未执行。下列文件路径均相对 PicketX/PicketXD。

## 全局约束

- 实现位于 PicketXD；添加 Agent 代码前，将空仓库实际 Apache-2.0 LICENSE 对齐到既定 GPL-3.0-or-later 社区方向。
- 仅在隔离测试命名空间内持有 `table inet picketx` 和 `picketx_` 对象；PREROUTING priority -110，测试 mark 为 0x5049434b。
- IPv4 192.0.2.0/24、IPv6 2001:db8:1::/64 是 Fixture 地址，不是生产默认值。
- 固定 go.mod/go.sum；默认运行时不依赖 C 库或 nft/conntrack 命令。
- 内核测试需明确 `PICKETX_KERNEL_TESTS=1`；已启用但权限/初始化缺失必须失败，不能跳过。
- 不实现/冻结 Manifest/protobuf/业务 Schema、Controller 通信、事件订阅或活动续期。
- 跳过内核测试不能满足 M0；报告实际验证的内核及依赖版本。

## 评审重点

- 权限不足或命名空间初始化失败：启用测试时，在规则修改前失败；任务 1。
- 重叠范围和非自有连接：保留仍有效权限及未标记连接；任务 2、3。
- 接近绝对截止时间的重启：不能重新发放原始完整 TTL；任务 2。
- NAT original/reply tuple 不同：只删除目标自有映射；任务 3、4。
- 查询中断或部分删除：保留明确失败/未完成结果及证据；任务 3、4。

---

### 任务 1：初始化内部验证 Fixture 和环境检查

**文件：**新增 `go.mod`、`go.sum`、`AGENTS.md`、`README.md`、`README.zh-CN.md`、`THIRD_PARTY_NOTICES.md`、`internal/validation/fixture/snapshot.go`、`snapshot_test.go`、`internal/validation/kerneltest/preflight_linux.go`、`preflight_linux_test.go`；修改 `LICENSE`。

**接口：**
- `fixture.Endpoint`：Destination netip.Prefix、Protocol uint8、PortFirst/PortLast uint16。
- `fixture.Rule`：Source netip.Prefix、Endpoint Endpoint、ExpiresAt time.Time；零截止时间只表示测试内部不过期 Fixture。
- `fixture.Snapshot`：Protected []Endpoint、Allow/Deny []Rule。
- `fixture.Validate(Snapshot) error`：拒绝无效/非规范前缀、同条规则来源与目标地址族不一致（快照可同时包含 IPv4/IPv6）、非 TCP(6)/UDP(17) 协议、倒序/零端口范围和空保护范围。
- `kerneltest.Preflight(ctx context.Context) (string, error)`：只有已启用进程可以创建/使用隔离测试命名空间时，才返回内核版本。

- [ ] 写 `TestValidateSnapshot`，覆盖有效双地址族及每种非法 Fixture。
- [ ] 写 `TestPreflightMissingPrivilege`：已启用但 Capability 缺失，在任何规则变更前返回错误；内部 `preflight(ctx context.Context, probe func(context.Context) error) (string, error)` 注入命名空间检查。
- [ ] 实现前运行这些测试，确认因验证/preflight 行为缺失而失败。
- [ ] 实现 Fixture Validation 和 Preflight；初始化 Go Module 与声明、对齐 LICENSE，文档仅描述验证命令和非生产状态。
- [ ] 运行 `go test ./...`、`go vet ./...`、`go build ./...`；确认 Fixture/Backend 运行代码不调用外部二进制。
- [ ] 提交：`chore(agent): bootstrap Linux data-plane validation`。

### 任务 2：验证 nftables 准入、集合、期限与 Batch 失败

**文件：**新增 `internal/validation/nftbackend/backend_linux.go`、`backend_linux_test.go`、`backend_integration_test.go`，以及 `internal/validation/kerneltest/topology_linux.go`。

**接口：**
- `nftbackend.New(conn *nftables.Conn) *Backend`。
- `(*Backend).Apply(ctx context.Context, snapshot fixture.Snapshot, now time.Time) error`：先验证，再准备一个自有 nftables Batch；忽略已到期成员，按绝对截止时间计算剩余 Timeout。
- `(*Backend).Close() error`：关闭连接；测试拓扑负责命名空间清理，不清理无关表。
- `kerneltest.WithTopology(t *testing.T, run func(*Topology))`：建立隔离客户端/执行命名空间、Echo 入口和确定性清理。
- `(*Topology).Probe(ctx context.Context, source, destination netip.Addr, protocol string, port uint16) error`：带截止时间的真实流量/连接探测。
- `(*Topology).OpenTCP(ctx context.Context, source, destination netip.Addr, port uint16) (net.Conn, error)`：保留已建立的 Fixture 连接以验证自然/严格到期。

- [ ] 写 `TestApplyRejectsInvalidSnapshotBeforeFlush`，使用小型内部 Batch 测试接口，断言非法输入不排队/提交变更。
- [ ] 写带 Tag 的集成用例：`TestDefaultDenyDualStack`、`TestScopedAllowDualStack`、`TestDenyPrecedence`、`TestConcatenatedIntervalSets`、`TestOverlapRetainsMembership`、`TestIndependentDeadlines`、`TestNativeTimeoutWithoutController`、`TestGracefulExpiry`、`TestRestartPreservesDeadline`、`TestInvalidBatchPreservesPreviousState`。
- [ ] 在具备权限的 Linux 上确认新用例因行为缺失而失败；未启用时跳过不算失败阶段或内核证据。
- [ ] 实现最小 Fixture Backend 和拓扑；无效 Batch 用真实非法 Netlink 操作触发，确认此前策略仍控制流量。必需集合/Timeout 组合不支持时明确失败。
- [ ] 执行单元检查和已启用的集成用例，检查 Go test JSON 中的实际执行与流量观测。
- [ ] 提交：`test(agent): validate nftables admission and expiry`。

### 任务 3：仅选择和删除失去有效权限的自有连接

**文件：**新增 `internal/validation/ctbackend/select_linux.go`、`select_linux_test.go`、`backend_linux.go`、`backend_linux_test.go`。

**接口：**
- `ctbackend.SelectWithdrawn(flows []*netlink.ConntrackFlow, before, after fixture.Snapshot, mark uint32, now time.Time) []*netlink.ConntrackFlow`：按 pre-DNAT 策略边界匹配 original tuple，要求自有 mark，排除经拒绝/期限判断后仍允许的连接。
- `ctbackend.Result`：Selected、Deleted、Remaining int。
- `ctbackend.New(handle *netlink.Handle) *Backend`。
- `(*Backend).Revoke(ctx context.Context, before, after fixture.Snapshot, mark uint32, now time.Time) (Result, error)`：查询、范围选择、删除、确认收回范围无残留；内部注入 dump/delete 测试接口以验证错误。

- [ ] 写 `TestSelectWithdrawn`，覆盖 IPv4/IPv6、NAT original/reply、主机/CIDR 重叠、其他端口/协议、新增拒绝、过期贡献和未标记连接。
- [ ] 写 `TestRevokeReportsInterruptedDump`、`TestRevokeReportsPartialDeletion`：error 必须非 nil，未完成计数不能作为成功清理返回。写 `TestRevokeNeverFlushesTable`。
- [ ] 实现前运行单元测试，确认连接选择/删除行为缺失导致失败。
- [ ] 实现范围匹配、vishvananda/netlink 接入、结果证据和删除后确认；库的实际限制作为验证发现，不忽略不完整查询，不清空整表。
- [ ] 运行 `go test ./...`、`go vet ./...`、`go build ./...`。
- [ ] 提交：`test(agent): validate scoped conntrack revocation`。

### 任务 4：在真实 Linux 验证组合契约并输出证据

**文件：**新增 `internal/validation/ctbackend/backend_integration_test.go`、`internal/validation/report/report.go`、`report_test.go`、`internal/validation/scenario/apply_linux.go`、`apply_linux_test.go`、`scripts/test-kernel.sh`、`.github/workflows/validation.yml`、`docs/validation/M0.md`、`docs/validation/M0.zh-CN.md`。

**接口：**
- `report.ScenarioResult`：Name、Status、Detail string；Selected、Deleted、Remaining int。
- `report.Report`：Kernel、Go string；Libraries map[string]string；Scenarios []ScenarioResult。
- `report.Write(w io.Writer, report Report) error`：输出 JSON，保留 pass/fail/skipped/unsupported 区别。
- `scenario.Applier`、`scenario.Revoker`：任务 2、3 定义的 Apply/Revoke 方法。
- `scenario.ApplyStrict(ctx context.Context, applier Applier, revoker Revoker, before, after fixture.Snapshot, mark uint32, now time.Time) (ctbackend.Result, error)`：Apply 失败时不得调用 Revoke。
- `scripts/test-kernel.sh`：要求具备权限的 Linux，在隔离父命名空间运行预编译 Tagged Tests，保留退出码，失败时仍清理。

- [ ] 写 `TestApplyStrictRejectsUnappliedState`：Applier 失败时返回错误，Revoker 调用次数为零。写 `TestReportPreservesIncompleteAndSkippedResults`，断言 JSON 不把失败、不支持或跳过转成通过。
- [ ] 写集成用例：`TestStrictRevokeDualStack`、`TestStrictRevokePreservesOtherGrant`、`TestStrictRevokePreservesUnrelatedFlow`、`TestOldTCPPacketsCannotResurrect`、`TestUDPAfterDeletionRequiresCurrentAllow`、`TestNATScopedDeletion`、`TestApplyFailureDoesNotTriggerDeletion`。
- [ ] 新用例针对缺失的组合行为确认真实失败，再实现更新顺序、报告与 Runner 行为。
- [ ] 新增单元 Job 与隔离 Linux 内核 Job，固定 Action revision；特权阶段前下载/构建 Module，归档 test JSON 和环境/场景报告，必需场景未执行则内核 Job 失败。
- [ ] 运行 `go test ./...`、`go vet ./...`、`go build ./...`，然后在具备权限的 Linux 上运行 `PICKETX_KERNEL_TESTS=1 scripts/test-kernel.sh`；逐项对照验收表与观测结果。
- [ ] 记录实际内核/版本、不支持行为、部分失败和经审查的依赖调整，不因跳过测试而标记 M0 完成。
- [ ] 提交：`test(agent): run Linux enforcement conformance suite`。

## 完成与下一阶段

在 PicketXD 创建实现 PR，附实际检查和剩余限制；合并前评审代码与证据。M0 结论决定生产 Backend 契约，之后 M1 逐步实现共享 Core/领域 Schema 与首个 Gitea 闭环。当前 Controller 文档 PR 不包含 Agent 代码，也不是内核验证结果。
