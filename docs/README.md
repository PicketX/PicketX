# Design documents

English is the default; Chinese editions carry the same decisions. These documents describe the target architecture, not implemented features. See the [root README](../README.md) for the development scaffold status.

| Document | English | 简体中文 |
| --- | --- | --- |
| Product v0.4.1 | [Product](PRODUCT_DESIGN.md) | [产品](PRODUCT_DESIGN.zh-CN.md) |
| Architecture v0.8.2 | [Architecture](ARCHITECTURE.md) | [架构](ARCHITECTURE.zh-CN.md) |
| Configuration model v0.1.0 | [Model](CONFIGURATION_MODEL.md) | [模型](CONFIGURATION_MODEL.zh-CN.md) |
| ADR 0001: RBAC authorization | [Decision](decisions/0001-rbac-authorization.md) | [授权决策](decisions/0001-rbac-authorization.zh-CN.md) |
| ADR 0002: Runtime support | [Decision](decisions/0002-runtime-support.md) | [运行支持](decisions/0002-runtime-support.zh-CN.md) |
| ADR 0003: Linux backend priority | [Decision](decisions/0003-linux-backends.md) | [Backend 决策](decisions/0003-linux-backends.zh-CN.md) |
| M0 validation design (executed) | [Design](superpowers/specs/2026-10-14-linux-data-plane-validation.md) | [验证设计](superpowers/specs/2026-10-14-linux-data-plane-validation.zh-CN.md) |
| M0 implementation plan (executed) | [Plan](superpowers/plans/2026-10-14-linux-data-plane-validation.md) | [实施计划](superpowers/plans/2026-10-14-linux-data-plane-validation.zh-CN.md) |
| M1.1 shared Core design (approved) | [Design](superpowers/specs/2026-10-15-shared-network-core-design.md) | [共享 Core 设计](superpowers/specs/2026-10-15-shared-network-core-design.zh-CN.md) |
| M1.1 Core implementation plan (executed, merged) | [Plan](superpowers/plans/2026-10-15-shared-network-core.md) | [实施计划](superpowers/plans/2026-10-15-shared-network-core.zh-CN.md) |
| M1.2 physical network planner design (approved, merged) | [Design](superpowers/specs/2026-10-15-physical-network-planner-design.md) | [物理网络规划](superpowers/specs/2026-10-15-physical-network-planner-design.zh-CN.md) |
| M1.2 planner implementation plan (pending review) | [Plan](superpowers/plans/2026-10-15-physical-network-planner.md) | [实施计划](superpowers/plans/2026-10-15-physical-network-planner.zh-CN.md) |

The product document defines outcomes, the architecture defines system boundaries, and the configuration model refines the agreed object/selection/binding contract. Unresolved schema details are explicitly tracked; illustrative YAML is not an implemented API.

Historical documentation is retained outside this clean source baseline; the previous repository contents remain in Git history. Future behavior changes should record focused ADRs and update both language editions.
