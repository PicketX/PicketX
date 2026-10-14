# ADR 0001：RBAC 授权与定时策略同步

> [English (Default)](0001-rbac-authorization.md) | 简体中文

| 字段 | 内容 |
| --- | --- |
| 日期 | 2026-10-14 |
| 状态 | 已确认设计；运行时接入待实现 |
| 范围 | PicketX 管理及申请人授权，以及适用的可信逐请求身份集成 |

## 背景

Casbin 默认 RBAC 将权限规则（`p`）和分组规则（`g`）一起加载到内存，并建立角色关系。共享角色权限减少重复的 `p`，但用户成员关系 `g` 仍可能随用户数量增长。使用 GORM 持久化不会自动把鉴权变成逐条查询数据库。

官方扩展建议区分请求吞吐与策略容量。多副本可以分担请求，但全部加载同一份策略时不会降低每个进程的策略内存。过滤加载可以缩小 `p`、`g` 的装载范围。外部 Role Manager 是另一种集成边界，其查询、缓存与初始化行为取决于具体实现。

Argo CD 从可信身份声明中读取当前用户的组，并与共享授权规则匹配；Harbor 用当前用户的角色和权限构建评估器。它们提供集成参考，不是 PicketX 的容量保证。

## 决策

### 1. 从 RBAC 开始

采用 Casbin 和官方 GORM Adapter，首先实现 RBAC。实际需求涉及资源归属、设备或请求属性等条件时，再扩展 ABAC；当前不选定 ABAC 模型或额外引擎。

Casbin 判断角色、资源和操作权限，包括 PicketX 管理操作及访问申请资格。PicketX 按钮显示对应相同的操作权限，后端对每个受保护操作执行鉴权；这不会替代被保护第三方应用的内部权限。

Authorization Core 继续负责 IP/CIDR 匹配、独立 Grant 贡献、权限合集、绝对截止时间、Lease 续期、网络编译和连接撤销。不为每个数据包、连接或临时 Grant 展开一条 Casbin 策略，也不能由 IP Grant 的申请人推断协议身份。

### 2. 用户成员关系留在共享 Enforcer 之外

| 数据 | 归属与用途 |
| --- | --- |
| 用户→组、用户→角色关系 | 可信身份/业务层及其权威存储 |
| 当前用户适用的组/角色 | 后端解析的请求授权上下文 |
| 组→角色绑定、角色继承 | 共享 Casbin `g` 规则 |
| 角色→资源/操作权限 | 共享 Casbin `p` 规则 |

不把整个用户目录的成员关系复制到共享 Casbin 策略中；通过身份/业务边界只解析当前用户适用的组/角色。本地用户和 OIDC 用户可使用相同授权输入，登录 Session 的存储与库选型仍单独处理。

标识须稳定并区分用户、组和角色。存在租户范围时，解析并校验该范围，不能直接接受调用方声明的角色或跨租户成员关系。本决策不引入 Namespace，也不冻结多租户 Schema。

业务层需要授权封装/模型来消费可信主体上下文，单独引入 GORM Adapter 不会自动实现这一分工。完整评估所有适用组/角色，显式拒绝优先，默认拒绝；不能命中第一个 Allow 就绕过其他角色的 Deny。

若明确新鲜度及成员关系撤销行为，可在 Session 缓存当前用户的组/角色。Casbin 定时策略重载不会刷新这些成员关系。登录时计算完整权限清单不作为当前基线。

### 3. 持久化与进程生命周期

- 使用 `github.com/casbin/casbin/v3` 和 `github.com/casbin/gorm-adapter/v3`；运行时接入时固定并验证兼容版本，不向当前骨架加入未使用的依赖。
- 通过官方 Adapter 的 `NewAdapterByDB` 边界复用已有 `*gorm.DB`。SQLite、PostgreSQL 继续使用已选定的 GORM 官方驱动，不为 Casbin 再做一套业务驱动选型。
- 保留官方 Adapter 对其自身 Casbin Schema 的默认建表与 AutoMigrate 行为；这与业务数据库的版本化迁移分开，也不承诺任意历史数据转换。
- 启动时每个进程创建一个共享 `SyncedEnforcer`，加载所需策略范围，通过面向应用的授权接口注入。Casbin/GORM 类型留在集成实现中，不使用全局服务查找或共享可变的当前用户字段。
- 注册策略与登录 Session 独立；不为每次鉴权创建 Enforcer 或重新加载策略。
- 重复决策缓存是后续可选优化，当前不选定，也不能解决策略容量增长。

### 4. 仅定时同步

通过正常的带类型运行配置边界，提供可自定义的正数策略同步间隔；每个副本按该周期重载权威策略。当前 Casbin 同步不引入 Redis/Valkey Watcher、数据库通知监听器或 Dispatcher；独立的登录 Session 后端仍可使用数据库、Redis 或 Valkey。

其他副本的变更在下一次成功定时重载后可见。扩大间隔降低同步开销，也延长授权变更和权限撤销的传播时间，启动提示必须说明该取舍。重载失败须作为可观察错误，不能更新成功新鲜度；持续失败时不能把健康同步间隔描述为传播上限。

该间隔仅作用于 Casbin 策略刷新，不重新定义 Agent 期望状态同步、Grant 截止时间、Lease 活动检查、UI 轮询或连接撤销契约。

后续采用过滤加载时，每次定时重载保留相同过滤范围；不能因调用完整 `LoadPolicy()` 而意外加载全部全局策略。

### 5. 启动时按实际策略规模提供建议

启动时分别报告实际加载的 `p`、`g` 条数、合计数量及首次加载耗时。结合规则数量分级与加载耗时提供建议：

| 观测到的规模/成本 | 建议 |
| --- | --- |
| 策略较少且加载成本低 | 展示选定间隔，无需建议扩大 |
| 策略较多，或加载占同步周期较大比例 | 建议扩大间隔，并说明传播延迟 |
| 策略非常多，或装载/内存成本不可接受 | 同时建议检查共享角色建模及按租户/domain 加载子集 |

规则数量分级及诊断阈值可配置，建议不自动修改既定间隔。具体配置键、默认周期和数值阈值留待实现；官方示例与基准不是通用容量上限。

采用容量分级前，使用代表性模型测量常驻内存、鉴权延迟及重载成本。扩大间隔不会降低已装载策略的常驻内存。按钮级操作本身不会导致按用户数量复制规则，但组/角色绑定与权限规则仍可能增长。

## 实现与验证边界

本 ADR 记录选型，不表示认证、授权、存储或配置字段已经实现；当前运行骨架保持原状。

宣称运行时支持前，验证可信成员关系解析、多角色拒绝优先、UI 操作权限对应的后端校验、两种已选数据库、指定范围内的并发重载及可见刷新失败。分别验证 Session 成员关系新鲜度、Casbin 策略传播和 Agent Grant 到期行为。检查不同策略量和加载成本下的启动建议，不自动重写配置。

## 参考资料

- [Casbin RBAC](https://casbin.apache.org/docs/rbac/)
- [生产部署](https://casbin.apache.org/docs/production-deployment/)
- [性能优化](https://casbin.apache.org/docs/performance/)
- [策略子集加载](https://casbin.apache.org/docs/policy-subset-loading/)
- [Role Manager](https://casbin.apache.org/docs/role-managers/)
- [官方 GORM Adapter](https://github.com/apache/casbin-gorm-adapter)
- [Argo CD 授权](https://argo-cd.readthedocs.io/en/stable/operator-manual/security/#authorization)
- [Argo CD 身份声明鉴权](https://github.com/argoproj/argo-cd/blob/master/server/rbacpolicy/rbacpolicy.go)
- [Harbor 当前用户策略适配器](https://github.com/goharbor/harbor/blob/main/src/pkg/permission/evaluator/rbac/casbin_adapter.go)
