# ADR 0003：Linux Backend 验证优先级

> [English (default)](0003-linux-backends.md) | 简体中文

- 日期：2026-10-14
- 状态：验证优先级已确认；内核验收与生产接入待完成

## 决策

nf_tables 配置优先验证 `google/nftables`，初始基线 v0.3.0。当前必需的定向严格撤销优先验证 `vishvananda/netlink`。`ti-mo/conntrack` 保留为 NEW/UPDATE/DESTROY 事件观测候选；选定删除 API 不代表解决了事件订阅。

默认运行时不新增必须安装的 C 库或 `nft`/`conntrack` 系统命令；诊断/集成测试环境可以独立安装这些工具。

## 边界依据

内核 conntrack 状态表达式和 mark 可由 nftables 配置，不需要用户态 conntrack 库。Agent 主动查询、观测或删除连接时才需要该库。当前设计要求严格撤销，不能退化成仅自然结束既有连接。

Kubernetes/kube-proxy、CNI plugins、OVN-Kubernetes 使用 vishvananda/netlink 定向删除；Antrea、Kindnet 有 ti-mo/conntrack 流量观测用途。这些是采用证据，不证明 PicketX 的到期、重叠和失败语义已经实现。

`ConntrackDeleteFilters` 先查询表、在 Go 中匹配，再逐项删除。需要验证连接表增长时的成本，并处理不完整查询与部分失败。业务必须先按权限并集/拒绝规则计算实际收回的权限，仅删除 PicketX 自有连接，保留其他 Grant 仍允许的连接。

## 验收

执行 [M0 设计](../superpowers/specs/2026-10-14-linux-data-plane-validation.zh-CN.md)与[实施计划](../superpowers/plans/2026-10-14-linux-data-plane-validation.zh-CN.md)。验证 IPv4/IPv6、区间/组合集合、到期、失败 Batch、定向严格/自然撤销、NAT 元组解释和重启。仅 Mock 或跳过内核测试不能满足 M0。

Backend 保持可替换。验证初始化时固定实际 conntrack 依赖；计划从已发布 v1.3.1 起步，功能缺失或错误报告问题必须成为验证结果，不能被忽略。事件观测接入另行作为聚焦任务推进。

## 参考

- [google/nftables](https://github.com/google/nftables)
- [vishvananda/netlink](https://github.com/vishvananda/netlink)
- [Kubernetes conntrack](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/conntrack/conntrack.go)
- [CNI plugins conntrack](https://github.com/containernetworking/plugins/blob/main/pkg/utils/conntrack.go)
- [OVN-Kubernetes 定向删除](https://github.com/ovn-kubernetes/ovn-kubernetes/blob/master/go-controller/pkg/util/net_linux.go)
- [Antrea Flow Exporter](https://github.com/antrea-io/antrea/blob/main/pkg/agent/flowexporter/connections/conntrack_linux.go)
- [Kindnet 连接事件](https://github.com/kubernetes-sigs/kindnet/blob/main/pkg/conntrack/conntrack.go)
