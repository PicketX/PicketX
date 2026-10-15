# 设计文档

英文为默认版本，中文承载相同决策。文档描述目标架构，不代表全部功能已经实现；当前初始化状态见[根目录说明](../README.zh-CN.md)。

- [产品设计 v0.4.1](PRODUCT_DESIGN.zh-CN.md) / [English](PRODUCT_DESIGN.md)
- [架构设计 v0.8.2](ARCHITECTURE.zh-CN.md) / [English](ARCHITECTURE.md)
- [配置与领域模型 v0.1.0](CONFIGURATION_MODEL.zh-CN.md) / [English](CONFIGURATION_MODEL.md)
- [ADR 0001：RBAC 授权与定时策略同步](decisions/0001-rbac-authorization.zh-CN.md) / [English](decisions/0001-rbac-authorization.md)

- [ADR 0002：运行支持组件](decisions/0002-runtime-support.zh-CN.md) / [English](decisions/0002-runtime-support.md)
- [ADR 0003：Linux Backend 验证优先级](decisions/0003-linux-backends.zh-CN.md) / [English](decisions/0003-linux-backends.md)
- [M0 验证设计（已执行）](superpowers/specs/2026-10-14-linux-data-plane-validation.zh-CN.md) / [English](superpowers/specs/2026-10-14-linux-data-plane-validation.md)
- [M0 实施计划（已执行）](superpowers/plans/2026-10-14-linux-data-plane-validation.zh-CN.md) / [English](superpowers/plans/2026-10-14-linux-data-plane-validation.md)
- [M1.1 共享 Core 设计（已批准）](superpowers/specs/2026-10-15-shared-network-core-design.zh-CN.md) / [English](superpowers/specs/2026-10-15-shared-network-core-design.md)
- [M1.1 Core 实施计划（已执行，待集成）](superpowers/plans/2026-10-15-shared-network-core.zh-CN.md) / [English](superpowers/plans/2026-10-15-shared-network-core.md)

产品文档定义目标，架构定义系统边界，配置模型细化已确认的对象/选择/绑定契约。待决 Schema 明确列出，示例 YAML 不是已经实现的 API。

历史文档单独保留，旧仓库内容仍可通过 Git 历史查看。后续行为变更通过聚焦 ADR 记录并同步双语文档。
