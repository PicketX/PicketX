# M0：Linux 数据面验证设计

> [English (default)](2026-10-14-linux-data-plane-validation.md) | 简体中文

- 日期：2026-10-14
- 状态：执行设计待评审；不声称已通过内核验收
- 实现仓库：[PicketX/PicketXD](https://github.com/PicketX/PicketXD)
- 上位依据：[架构](../../ARCHITECTURE.zh-CN.md)、[配置边界](../../CONFIGURATION_MODEL.zh-CN.md)、[ADR 0003](../../decisions/0003-linux-backends.zh-CN.md)
- 计划：[实施计划](../plans/2026-10-14-linux-data-plane-validation.zh-CN.md)

## 1. 目标

建立可重复执行的 Linux 集成测试，验证选定 Go Backend 能否实现 PicketX 的网络执行契约。在完整 Agent、Controller 流程实现前，给出真实数据包/连接观测，以及明确的通过、失败、不支持证据。

首个产品闭环仍是 Gitea：浏览器审批、来源范围内的 HTTPS/SSH 可达性、执行状态及到期/撤销。M0 是其数据面前置验证，不代表完成该闭环。

## 2. 已核实的起点

2026-10-14，PicketX main 为 `dc62c73fc3017f2fb2db1f60e8cd45eec02d27a6`：Go 1.27.0、Echo 存活检查、React 骨架。PicketXD main 为 `2b4400d05c9c7348ca6ffc436f63e0819a4d1dc7`，仅有 LICENSE。

PicketXD 实际 LICENSE 仍为 Apache-2.0，而既定 Agent 方向是 GPL-3.0-or-later/OEM。在引入 Agent 实现前，通过初始化变更对齐空 Agent 仓库的社区许可证和声明。当前文档 PR 不修改该 LICENSE，也不授予额外 OEM 权利。

当前执行环境没有 Go 编译器，没有有效/边界 Linux Capability，不能创建用户/网络命名空间，无法提供真实 M0 内核证据。后续测试在明确具备权限的 Linux 机器或隔离 CI Runner 上运行。

## 3. 范围与边界

包含 IPv4/IPv6 TCP/UDP 准入、显式拒绝、来源/目标/端口区间与组合集合、原生元素超时、绝对期限恢复、失败 nftables Batch、自然到期、定向严格撤销、NAT 元组匹配和命名空间所有权。

不包含生产配置解析、稳定 Manifest/protobuf/API Schema、用户/RBAC/Session、Controller 通信、审批界面、动态活动续期、事件订阅、生产 LKG 存储和 HA。必需的事件观测能力仍是后续任务；选择删除能力不删除该需求。

使用测试内部的规范化快照，明确地址、协议、端口范围与绝对期限。重叠场景提供代表预期权限并集的更新前/后快照，验证 Backend 与连接选择，不声称实现完整生产 Grant Compiler；生产并集/defaulting 在 M1 通过共享 Core 一致性测试验证。

实现只放在 PicketXD，不把 GPL Agent 代码复制到 Apache-2.0 的 Controller/Core 仓库。Fixture DTO 保持验证套件内部使用。

## 4. 依赖与环境基线

使用 Go 1.27.0 或经审查的兼容补丁、`github.com/google/nftables` v0.3.0，以及初始 `github.com/vishvananda/netlink` v1.3.1；固定 go.mod/go.sum。变更依赖版本前记录失败或缺失行为。默认运行时不新增 C 库或 nft/conntrack 可执行文件依赖。

测试可使用 Linux `ip`/`unshare` 和诊断工具；在特权阶段前构建、下载依赖。只有 `PICKETX_KERNEL_TESTS=1` 时启用内核测试。未启用时可明确跳过；启用后权限缺失、初始化失败、必需内核行为不支持都必须失败，不能当作验收通过。

在隔离的父网络命名空间中运行，再创建测试客户端/服务端命名空间和链路。使用文档专用地址 IPv4 192.0.2.0/24、IPv6 2001:db8:1::/64，以及本地 TCP/UDP Echo Fixture，不连接生产 SSH/Gitea。测试不修改宿主机自有防火墙/路由。

## 5. 所有权与更新顺序

在隔离执行命名空间内，仅持有 `table inet picketx` 与 `picketx_` 对象；基线 Hook 为 PREROUTING priority -110，明确目标地址匹配。测试连接标记为 `0x5049434b`，不冻结生产 mark 分配。

先完整提交 nftables 候选 Batch 并确认结果，再在严格收回时定向清理 conntrack。Batch 失败保留此前可用规则，不因未成功应用的候选执行连接删除。

按新旧有效快照计算真正收回的范围。只删除失去有效权限的自有连接；保留独立授权、范围外和未标记连接。NAT 匹配使用对应 pre-DNAT 策略边界的 original tuple，并明确验证 reply tuple。删除 conntrack 不等于发送 TCP Reset 或保证关闭应用 Socket。

查询中断、删除错误或观察到目标连接仍残留，都视为失败/未完成，并返回部分结果证据。不以清空宿主机整个 conntrack 表作为恢复手段。

## 6. 场景验收

| 场景 | 必须观察到的结果 |
| --- | --- |
| IPv4/IPv6 默认拒绝 | 未授权来源无法向保护入口新建 TCP/UDP 访问 |
| 范围内允许 | 授权来源可访问指定入口；其他来源/端口仍拒绝 |
| 重叠授权 | 删除精确主机贡献后，其他有效 CIDR 贡献仍允许访问 |
| 独立期限 | 网站权限到期，独立授权的数据库端口仍可访问 |
| Deny 优先 | 同范围的显式拒绝优先于允许 |
| 集合组合 | 在实际内核验证 CIDR、目标、协议、端口区间组合 |
| 原生超时 | 无 Controller 进程时，元素超时后停止新访问 |
| 自然到期 | 已授权 TCP 流量继续；新连接拒绝 |
| 严格撤销 | 收回连接的后续流量被阻断；无关/仍授权连接继续 |
| TCP 不复活 | 删除状态后，旧 ACK/PSH 不能绕过有效新 SYN 授权恢复访问 |
| UDP 重新检查 | 删除后下一数据报形成新伪连接，必须符合当前策略 |
| 无效 Batch | 候选失败，此前网络行为仍可用 |
| NAT 范围 | 删除命中预期 original/reply 映射，保留无关映射 |
| 重启/期限 | 用同一绝对截止时间重新应用快照，不重置有效期 |
| 权限不足 | 已启用内核测试时，在任何规则变更前明确失败 |
| 部分清理 | 查询/删除未完成不能表现为成功撤销 |

通过带明确 Socket/测试截止时间的轮询观察结果，不用固定 sleep 证明到期。宽裕测试时限只避免调度抖动，不定义生产过期延迟承诺。

## 7. 证据与完成门槛

归档 Go test JSON 与机器可读环境/场景报告：内核版本、Go/库版本、实际执行场景、结果、权限失败和删除数量。单元、内核、跳过、不支持状态分别报告。

只有必需场景在真实 Linux 成功执行并评审结论，M0 才完成。跳过内核测试的绿色 Job 不足以验收。记录实际验证的内核范围，不宣称全部发行版或旧内核均兼容。

套件是可复用工程验证，不是生产 Agent 二进制。审阅测量结论及必要的明确设计调整后，再进入 M1 Schema/Core/业务闭环实现。

## 8. 工作方式

评审本设计及关联计划后，按任务顺序执行：先失败测试、后代码，聚焦提交、真实检查、代码评审与 PR。此小规模、有依赖的任务序列建议由当前会话直接执行；只有要求时才委派。后续前端工作使用已安装 UX/UI Skill，并遵循仓库中的设计决策。
