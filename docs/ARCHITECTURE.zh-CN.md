# PicketX 架构设计

> [English (Default)](./ARCHITECTURE.md) | 简体中文

| 字段 | 内容 |
| --- | --- |
| 版本 | v0.8.2 |
| 日期 | 2026-10-14 |
| 状态 | 已确认的设计基线；不代表已实现 |
| 项目 | PicketX |

**设计原则：**访问流程简单 · IP/CIDR 授权明确 · 权限生命周期可靠 · 业务零改造优先 · 执行能力可扩展

## 文档定位

本文档承接 [PRODUCT_DESIGN.zh-CN.md](./PRODUCT_DESIGN.zh-CN.md) 的产品基线，是 PicketX 的系统上位架构设计，统一定义产品边界、领域模型、功能架构、部署形态、Linux 数据面、控制面、授权服务、插件与 WASM、可靠性、安全性、许可证边界和分阶段实施路线。

后续 PRD、API、数据库模型、协议规范、实现 ADR 和运维手册均应遵循本文档；如需偏离，应以新的正式决策明确替代关系。

## 当前架构决策

| 决策项 | 当前结论 |
| --- | --- |
| 产品定位 | 简单易用、无需专用客户端，以 IP/CIDR 为网络授权对象的按需访问授权系统；独立使用或补充保护现有网络。 |
| 网络授权模型 | 聚合仍有效且匹配的来源授权，受显式拒绝与条件约束；申请人、审批人是授权来源，不是连接身份。 |
| 产品范围 | 不按组织规模限制适用范围；授权记录属于核心，应用行为记录不属于范围。 |
| 核心数据面 | nftables/Netfilter + conntrack 实现 IP/CIDR 准入；NFQUEUE、镜像检测响应、透明代理与 TLS 处理属于后续探索，不是默认 Agent 要求。 |
| 核心进程 | `picketx-controller`、`picketxd` 和 `picketx` CLI；Web SPA 可嵌入 Controller。 |
| 授权服务 | 提供通用 Authorization API；Nginx `auth_request`、Envoy `ext_authz`、Traefik `ForwardAuth` 等均为适配器。 |
| HTTPS 边界 | PicketX 在授权路径中不终止业务 TLS；TLS 终止与 HTTP 转发继续由 Nginx、Envoy、Traefik、Caddy 等承担。 |
| 扩展机制 | 公开外部插件协议与 WASM Policy Runtime；Native Plugin 与 External Plugin 使用不同的信任和许可证边界。 |
| `picketxd` 许可证 | 社区版本使用 GPL-3.0-or-later；另行提供 OEM Commercial License；贡献协议授予项目方实施经批准再许可所需的权利。 |
| 其他组件许可证 | Controller、CLI、公开协议与 SDK、External Plugin SDK 默认使用 Apache-2.0。 |
| 后端语言与仓库 | Go；PicketX 承载控制面/Core/Web，PicketXD 承载官方 Linux Agent。 |
| Controller HTTP 框架 | Echo v5（MIT）；HTTP 适配层保持精简，业务/Core 包独立。 |
| RBAC 授权 | Casbin + 官方 GORM Adapter；当前用户可信组/角色输入、共享组/角色规则，每进程一个 SyncedEnforcer；ABAC 按需扩展。 |
| RBAC 策略同步 | 仅定时重载，间隔可配置，启动按规模/成本提供建议；不自动改间隔，不依赖策略 Watcher。 |
| 登录 Session 与 CSRF | SCS v2，内存/GORM/Redis Store；Echo 内置 CSRF；接入待实现。 |
| 限流 | Echo 内置本地计数；普通 HTTP 与 Agent 通信独立配置。 |
| Agent 通信 | 官方 grpc-go/protobuf；独立 Agent Token；明文或服务端 TLS；mTLS 后续可选。 |
| 指标与周期任务 | Prometheus client_golang；Controller/Agent 分别采集；Go Timer/Ticker 与显式生命周期。 |
| Linux Backend 优先级 | google/nftables 配置规则；vishvananda/netlink 用于必需的严格撤销；事件观测另行评估。 |
| SQLite 驱动与发布 | 官方 GORM SQLite + mattn；Linux 默认完整静态链接，无额外共享库安装要求。 |
| Controller 持久化 | GORM（MIT）；官方 SQLite/mattn 与 PostgreSQL/pgx 驱动；SQLite 用于单 Controller，PostgreSQL 用于集群；迁移工具暂缓。 |
| CLI 框架 | Cobra（Apache-2.0）；轻量命令适配器，Koanf 配置与独立业务/Core。 |
| 运行日志 | 标准库 slog；集中创建并注入 Logger，文本/JSON 输出；授权审计独立持久化。 |
| 运行时配置 | Koanf v2（MIT）；文件/环境变量/启动参数明确覆盖，模块接收带类型配置；策略 Manifest 独立解析。 |
| 依赖注入与组装 | 手工构造函数注入、按模块组装、集中生命周期管理；不引入 DI 框架。 |
| API 文档 | Swaggo v2（MIT），固定 CLI 版本；从 HTTP DTO 和注释生成 OpenAPI 3.1；Swagger UI v5（Apache-2.0）展示文档。 |
| 请求参数校验 | HTTP 字段使用 go-playground/validator v10（MIT）；模型约束使用可复用 Go 函数。 |
| 配置契约 | 远程与静态输入共用版本化语义；Resource/Endpoint/Grant/Binding 与业务流程、执行模型分离；不引入 Namespace。 |
| 前端 | React、TypeScript、Vite、shadcn/ui、Tailwind CSS、i18next、npm；官方基础组件与项目组合分层。 |

## 目录

1. [产品定位与设计原则](#1-产品定位与设计原则)
2. [目标场景与非目标](#2-目标场景与非目标)
3. [业务架构与领域模型](#3-业务架构与领域模型)
4. [总体系统架构](#4-总体系统架构)
5. [控制面设计](#5-控制面设计)
6. [Agent 与 Linux 数据面](#6-agent-与-linux-数据面)
7. [网络策略与权限模型](#7-网络策略与权限模型)
8. [Lease、活动续期与连接生命周期](#8-lease活动续期与连接生命周期)
9. [Authorization Service](#9-authorization-service)
10. [插件与 WASM 架构](#10-插件与-wasm-架构)
11. [API 与协议](#11-api-与协议)
12. [数据与状态架构](#12-数据与状态架构)
13. [高可用、同步与一致性](#13-高可用同步与一致性)
14. [安全设计](#14-安全设计)
15. [可观测性与运维](#15-可观测性与运维)
16. [部署形态](#16-部署形态)
17. [许可证与代码边界](#17-许可证与代码边界)
18. [分阶段实施路线](#18-分阶段实施路线)
19. [风险与设计约束](#19-风险与设计约束)
20. [附录](#20-附录)

## 1. 产品定位与设计原则

PicketX 是一个简单易用、无需安装专用客户端的按需访问授权系统，以 IP / CIDR 为网络授权对象，通过预设策略、自助申请或审批，动态开放指定资源，并管理权限的有效期与回收。它支持同一授权来源下的多设备、跨应用共享访问，可独立使用，也可为现有 VPN 和应用认证体系提供补充保护，并逐步扩展协议级的细粒度授权。

网络门禁决定来源能否建立连接；协议适配器可增加经过认证的用户、会话或请求约束。申请人认证决定谁能申请授权，不会认证该地址之后的每一条连接。

同一授权来源下的共享访问是有意提供的能力，不要求逐设备登记。VPN 可以承担连通与加密，PicketX 增加资源级限时访问。授权不会创建路由或隧道。“无需客户端”指无需专用 PicketX 终端软件，浏览器和原有协议工具仍然需要。

整体设计遵循以下原则：

- **IPv6 是一等公民。**动态自助访问通常绑定 IPv6 `/128` 或 IPv4 `/32`；CIDR 主要是管理策略，而不是动态会话的默认表达方式。
- **业务零改造优先。**SSH、MySQL、PVE、内部 Web、厂商软件等应优先在主机网络层或现有代理的授权 hook 中实施控制。
- **不重复建设成熟基础设施。**转发、TLS、HTTP/2、HTTP/3、WAF 和代理能力继续由 Linux、Nginx、Envoy、Traefik、Caddy 等承担。
- **控制面声明式。**权威配置源产生 Desired State，Agent 保存 Last Known Good（LKG），nftables/conntrack 是 Applied State。
- **策略与执行解耦。**领域对象、策略编译、Netfilter enforcement、Authorization Adapter 分层实现。
- **安全取舍显式化。**安全与性能之间的取舍应成为有文档的数据面策略开关，而不是隐藏的硬编码行为。
- **可解释、可审计、可回滚优先。**PicketX 不演变为无边界的 DPI 平台。

## 2. 目标场景与非目标

### 2.1 目标场景

| 场景 | 期望结果 |
| --- | --- |
| 个人及开发资源 | 网站、管理后台、测试系统和部分数据库在需要时开放，平时限制访问。 |
| 多设备使用 | 一次授权覆盖流量匹配指定 IP/CIDR 的设备，不逐台安装 PicketX 客户端。 |
| 多个应用访问同一服务 | 同一来源授权可供 IDE、数据库客户端、开发程序、命令行和脚本共享，无需逐应用接入 PicketX。 |
| 临时测试与外部协作 | 测试人员、客户或协作者申请资源和时长，由有权限的人批准，到期自动回收权限。 |
| 现有 VPN 或内部网络 | 增加资源级限时授权，保留原有网络连接方式。 |
| 显式静态来源策略 | 管理员声明永久或绝对到期的 IP/CIDR 策略，支持独立 YAML 运行。 |

组织规模不属于产品定义。是否适用取决于访问流程、来源共享策略与运维要求。使用频率和付费意愿仍需验证，不作为既成市场结论。

现有入口提供 HTTP/HTTPS 授权与本地决策接口属于首期应用集成；其他协议 Adapter、丰富的外部 Provider、Cloud 和 OEM 保留为扩展方向，不作为首个可用版本的前置条件。

### 2.2 明确非目标

| 能力 | 是否核心 | 边界说明 |
| --- | --- | --- |
| VPN 隧道 | 否 | 不创建 overlay 或 tunnel；控制真实网络上的服务可达性。 |
| 完整 WAF | 否 | HTTP 元数据可以参与授权，但不负责通用 SQL 注入、XSS 或 body 检测。 |
| 完整 IDS/IPS | 否 | 包检查与异步检测响应暂缓，不能替代同步应用授权。 |
| 反向代理 | 否 | Authorization Service 不终止业务 TLS，也不转发业务流量。 |
| 应用内部 RBAC | 否 | PicketX 控制入口边界，应用保留业务权限。 |
| 应用行为记录 | 否 | 不记录完整 SQL、命令、页面操作或文件内容；授权记录仍是核心。 |
| Service Mesh | 否 | 不提供服务发现、负载均衡、重试、熔断等通用 Mesh 能力。 |

## 3. 业务架构与领域模型

领域模型区分流程操作者、获授权主体、逻辑服务、访问入口与执行结果。[配置与领域模型设计](CONFIGURATION_MODEL.zh-CN.md) 记录已确认的选择、绑定语义，并列出仍待细化的 Schema。

| 领域对象 | 职责 | 关键语义 |
| --- | --- | --- |
| Agent（Node） | 执行身份；Linux 官方实现为 `picketxd` | 注册或静态本地身份、标签、能力、期望/已应用状态 |
| Resource | 受保护的逻辑服务 | 包含多个具名 Endpoint，不是单个目标元组 |
| Endpoint | 可独立选择的服务入口 | 类型默认 `network`；协议、端口、可选目标 IP/CIDR 数组 `addresses` |
| Subject / Source | 带类型的授权接收者；Source 为 IP/CIDR 类型 | 同时支持 IPv4、IPv6；申请人身份不是数据包身份 |
| Scope | 资源选择、可选入口限制与动作 | 省略入口限制表示全部及未来新增入口；网络动作默认 `connect` |
| Grant | 一份独立权限贡献 | 一个 Subject、多个 Scope、共享有效期/条件、来源和撤销 |
| EnforcementBinding | 将资源入口部署到 Agent | 非空 `resources[]` 与 `agents[]`，选中入口应用到每个选中 Agent |
| Requester / Approver / AccessRequest | 控制面流程与来源 | 区分申请和批准内容；Agent 不执行审批流程 |
| Policy | 资格、执行限制或协议评估 | 评估上下文分开，具体 Schema 划分待后续确定 |
| EffectivePermissionSet / Lease | 派生权限与可选生命周期机制 | 不是独立授权来源；不要求每个 Grant 都有 SourceSession |
| AuthorizationContext / Decision | 请求时协议授权评估 | 按需提供可信身份与信任来源；不同于 AccessRequest |
| AuditEvent / ExecutionStatus | 生命周期证据与观察结果 | 审批通过、Grant 有效、确认实施分别记录 |
| Plugin | 扩展能力 | 能力、版本、沙箱与配置，详细 Schema 随实际集成需求推进 |

### 3.1 业务、配置与执行边界

控制面 Web/API/持久化模型管理身份、申请、审批与审计，经发布边界把获授权的业务结果转换成版本化配置契约。Agent 消费配置与来源标识，不消费数据库行或 UI 流程对象；执行计划、LKG 与状态是独立运行时模型。

ControllerProvider 与 StaticManifestProvider 使用相同的校验/默认值及本地编译、协调语义，同时仍是互斥权威来源。控制面全局数据与本地 Manifest 集合不必完全相同。共享 Go 包和协议一致性验证保证控制面与 Agent 独立演进时授权语义一致。

### 3.2 选择、部署与隔离

Resource 包含多个 Endpoint。每个资源选择项省略 `endpointNames` 和 `endpointSelector` 时，选择全部当前及未来入口；缩小范围时只能填写一个非空限制。显式空值或 null 无效，不设置 `allEndpoints` 字段。标签选择器使用 `matchLabels` 与 `In`/`NotIn`/`Exists`/`DoesNotExist`；`NotIn` 匹配缺失 Key，不是显式拒绝。

EnforcementBinding 组织多个 Resource 选择和多个 Agent 引用/选择器。Agent 匹配取并集去重，所有选中入口应用到所有选中 Agent，不按位置配对。执行参数不同需拆分 Binding。默认主机部署；网关实施需要显式且经过能力校验的转发契约。

Controller 只向每个 Agent 投影相关保护、权限、限制、截止时间及清理清单。重叠 Grant 先聚合，不逐 Grant 生成规则，同时保留来源与独立到期语义。最后一份 Grant 撤销后入口继续受保护；最后一份 Binding 删除则撤除相应位置的保护责任。

不引入 Namespace。对象身份、有权限的引用、贡献追踪与冲突校验实现配置隔离。后端无法区分的入口，不会因标签或 Resource 名称不同而获得流量隔离。目标地址数组、范围默认值、状态、校验目标与待决生命周期见专项模型文档。

> **ALL 的定义：**产品概念上的 ALL 表示所有 PicketX 管理的 Resource，不是主机全部流量。省略 Endpoint 限制仅选中显式 Resource 范围内的全部入口，不会隐式选中全部 Resource 或 Agent；本文不定义全局 ALL 的编码。

## 4. 总体系统架构

```mermaid
flowchart TB
    Clients["Web · CLI · Automation"] --> Controller["picketx-controller · REST · OIDC · RBAC · Policy · Lease · AuthZ"]
    Controller --> DB["SQLite / PostgreSQL"]
    Controller -->|"gRPC stream + Token; TLS optional"| AgentA["picketxd · Node A"]
    Controller -->|"gRPC stream + Token; TLS optional"| AgentB["picketxd · Node B"]
    AgentA --> KernelA["Netfilter · conntrack"]
    AgentB --> KernelB["Netfilter · conntrack"]
    Proxies["Nginx · Envoy · Traefik · Caddy"] -->|"auth request / ext_authz"| Controller
```

- **Northbound：**Web、CLI 和自动化工具调用 Controller 的 REST/JSON/OpenAPI。
- **Southbound：**Agent 使用独立 Token 认证，主动建立 gRPC/protobuf 长连接；支持明文与服务端 TLS。mTLS 是未来可选方式，不是首版要求。
- **Local Administration：**`picketx agent status`、`doctor`、`dump`、`reconcile`、`reload` 通过本地 Unix socket 调用。
- **Authorization：**代理与网关调用 Controller 或独立 AuthZ endpoint。
- **Enforcement：**只有 `picketxd` 修改 `table inet picketx`，Controller 不直接操作主机防火墙。

### 4.1 Authorization Core、Controller 与 Agent Runtime

PicketX 将可复用授权语义与部署、执行分离。核心模型为 **带类型的 Subject–Resource–Action–Conditions**。网络 Source 是一种主体，经过验证的用户或服务身份是其他主体类型。来源 Grant 继续遵守第 7–8 节 IP/CIDR 语义；身份授权和请求决策不能直接编译为 IP 放行，除非这明确符合策略本意。

| 层次 | 职责 | 边界 |
| --- | --- | --- |
| Authorization Core | 带类型领域对象、校验、Grant 贡献、条件、权限聚合、决策解释和与 Backend 无关的规划 | 可复用 Go 包；不依赖 Controller 数据库、HTTP 服务或 Linux 系统调用 |
| Controller | 认证、流程/RBAC、持久化、全局资源部署、版本、分发、状态汇总和 AuthZ API | 使用 Core；不直接修改 Linux 防火墙状态 |
| Agent Runtime | 权威 Provider、本地校验、同步、LKG、到期、Reconcile/重试、Adapter 生命周期和状态 | 支持 Controller 或 Static Manifest Provider |
| Enforcement Adapter | 能力校验、观察、转换和对自有目标的应用 | Linux 是首个官方实现；后续可增加设备、云和网关适配器 |

官方 Go 组件复用同一份核心语义。Controller 计算全局和各目标的期望，Agent 验证收到的约束并转换为本地操作。第三方 Agent 可使用其他语言，但必须通过协议与语义一致性验证。

### 4.2 仓库与实现语言

| 仓库 | 职责 | 默认许可证意图 |
| --- | --- | --- |
| `PicketX/PicketX` | Controller、Authorization Core、用户门户/管理 Web、管理 CLI、公开 Schema/协议和 SDK | Apache-2.0 |
| `PicketX/PicketXD` | 官方 Linux Agent，二进制/服务名 `picketxd`，本地 Runtime、Linux Adapter 与诊断 | GPL-3.0-or-later、可替代 OEM 条款、贡献者再许可机制 |

**两个仓库的后端与运行时以 Go 为主要开发语言。** Web 继续使用 React 与 TypeScript。此决策替代旧 Rust Workspace 方案，不预设未来必须用 Rust 重写。常规网络执行仍在 Linux 内核，Go 负责控制、同步和选定用户态工作；可选探测路径仍需单独测量性能。

Core 与公开协议包初期放在 PicketX 仓库，通过带版本的 Go Module 被 PicketXD 引用。依赖方向是 PicketXD 引用公开/Core 包，而不是 Apache 控制面引用 Agent 实现代码。禁止两仓库复制并各自维护核心规则。具体 Module 拆分和公开包名属于实现细节；公开协议保持语言无关。

### 4.3 Adapter 契约与真实执行状态

概念契约包含 `Capabilities`、`Validate`、`Observe`、可选 `Plan` 和幂等 `Apply`。输入是带版本、自有对象标识和绝对截止时间的 Desired State，而不是会丢失重叠 Grant 贡献关系的独立 `grant(ip)` / `revoke(ip)` 命令。

能力声明包含主体类型、IPv4/IPv6/CIDR、资源/动作范围、原生到期、本地到期补偿、Strict 连接撤销、原子替换和可观测确认。无法满足的要求应被拒绝，除非操作者明确选择已展示的较弱约束。用户身份要求绝不能静默降级成来源 IP 过滤。

Adapter 只修改自身拥有的对象，报告 Desired/Applied Revision、观测时间、错误和部分结果。原子性仅限目标支持的操作；跨设备以及防火墙/路由/conntrack 变更不是一个全局事务。多步骤执行失败时，在支持的范围内保留或恢复上一有效状态并报告实际结果；已经终止的连接不可回滚。

状态下发 Agent 与逐请求 AuthZ Hook 是独立集成路径。增加厂商集成通常先增加 Runtime 内的 Adapter，而非立即创建全新 Agent。WASM 是可选扩展，不是首期交付前置条件。

## 5. 控制面设计

### 5.1 Controller 职责

- 用户、组、OIDC 与管理/申请 RBAC；这些身份决定流程操作权限，不用于推断数据包归属。
- 基础自助与审批流程、独立来源 Grant、授权来源记录、有效权限聚合。
- Resource/Endpoint、Agent、Grant、EnforcementBinding、执行 Policy、Label 与 Selector 管理；业务存储模型保持独立。
- 将全局 Desired State 编译为 per-node Desired State。
- Agent 注册、能力发现、心跳、revision 同步和状态聚合。
- Authorization Engine：处理通用授权、`auth_request`、`ext_authz` 和 ForwardAuth。
- 授权生命周期记录属于核心；高级审批、策略模拟、WASM 和 External Provider 管理分阶段扩展。
- SQLite 支持轻量单 Controller，PostgreSQL 支持集群部署。

### 5.2 借鉴 Kubernetes，但不复制 Kubernetes

| 借鉴概念 | 在 PicketX 中的用途 | 明确不做 |
| --- | --- | --- |
| `spec` / `status` | 分离声明状态与观察状态 | 不实现通用 CRD 平台 |
| Labels / selectors | 通过显式 Binding 选择资源入口与执行 Agent | v1 不引入 Namespace 或嵌套 NodeGroup |
| `generation` / `observedGeneration` | 追踪期望与应用进度 | 初期不依赖 etcd |
| Reconcile | 最终一致性与漂移恢复 | 不做 scheduler |
| Watch / revision | 增量分发与唤醒 | 不把 `LISTEN/NOTIFY` 当作可靠日志 |

### 5.3 Controller HTTP 框架

**Controller HTTP 框架确定为 Echo v5（MIT）。** 它负责路由、请求/响应处理和中间件组合。Handler 返回错误并由统一 HTTP 错误处理器处理，适合申请与审批流程。初始骨架在 `go.mod` 中锁定 `github.com/labstack/echo/v5` v5.4.0；升级时检查发布说明和中间件主版本兼容性。

- Echo 类型仅用于 HTTP 适配层。业务服务、共享授权与配置包使用普通 Go 类型和 `context.Context`；PicketXD 不依赖 Controller 的 HTTP 框架。
- 使用独立的请求/响应 DTO。申请者、审批者的认证身份由服务端确定，不把客户端输入直接绑定到持久化或领域对象；显式选择参数来源。
- Echo 不内置校验引擎，采用第 5.5 节确定的 go-playground/validator v10；模型约束与授权检查留在可复用的领域/业务代码中，保证远程与静态配置语义一致。
- 实现业务接口时，通过统一 HTTP 错误处理器将业务错误映射为 HTTP 状态与稳定 API 错误码，内部错误留在服务端日志。当前存活检查骨架使用 Echo 默认 HTTP 错误处理，尚未定义业务错误契约。
- Echo 不决定持久化、OIDC/会话实现、OpenAPI 工具或 gRPC。GORM 在第 5.4 节独立确定，Swaggo 文档生成在第 5.6 节确定，其余工具继续分别选型；Agent 同步保持既定协议边界。
- 当前骨架仅暴露 `GET /healthz` 进程存活检查（以及 HTTP HEAD 语义），设置 `Cache-Control: no-store`。超时与优雅退出继续由现有 `http.Server` 管理。接入框架不代表已经实现认证、审批、Grant 或执行功能。

参考：[Echo 文档](https://echo.labstack.com/)、[版本策略](https://echo.labstack.com/guide/installation/)、[v5 迁移](https://github.com/labstack/echo/blob/master/API_CHANGES_V5.md)。

### 5.4 Controller 持久化

**Controller 数据库访问层确定为 GORM（MIT）**，独立于 Echo。初始接入基线为 `gorm.io/gorm` v1.31.2，核验日期为 2026-10-13。在首个持久化实现实际引用时，将依赖锁定到 `go.mod`/`go.sum`；当前存活检查骨架没有数据库连接或 GORM 运行时依赖。

- 保留 SQLite 用于轻量单 Controller 部署、PostgreSQL 用于集群部署。SQLite 与 PostgreSQL 官方驱动分别在第 5.4.1、5.4.2 节确定；连接参数和迁移工具仍独立设计。
- GORM 模型和 `*gorm.DB` 限于持久化实现。HTTP DTO、数据库模型与标准配置契约分离；共享授权/配置包及 Agent 契约保持 ORM 无关。
- 为相关审批、Grant 与发布记录写入定义明确事务边界。数据库提交不代表 Agent 已执行；最终表结构、并发控制和可靠发布机制随首个业务流程设计。
- 已部署实例采用经过审查、带版本的 Schema 迁移。生产升级不能仅依赖启动时 `AutoMigrate`；本次选型不决定迁移工具。
- 目标使用体验为 Controller 启动时自动初始化或按受支持的升级路径更新数据库，完成后再让业务服务就绪。迁移库暂缓选型，随首批持久化模型确定；Goose、golang-migrate 与 Gormigrate 继续作为候选。在实现运行时行为前，需要明确并发保护与失败处理。
- GORM 可以抽象部分 SQL 差异，但不能统一数据库并发语义。在声称支持 SQLite 和 PostgreSQL 前，必须分别验证业务一致性与迁移；时间存储、约束及更新行为应保持已约定的授权语义。

参考：[GORM](https://gorm.io/docs/)、[v1.31.2](https://github.com/go-gorm/gorm/releases/tag/v1.31.2)、[迁移说明](https://gorm.io/docs/migration.html)。

### 5.4.1 SQLite 驱动与独立发布要求

**SQLite 采用官方 `gorm.io/driver/sqlite`（MIT），底层使用 `github.com/mattn/go-sqlite3`（MIT）**。于 2026-10-13 核实的初始接入基线为 SQLite Dialector v1.6.0、mattn v1.14.52，保留已选 GORM v1.31.2 基线。首个持久化实现接入时固定并验证版本组合；本文不宣称该组合已通过 PicketX 测试。官方 Dialector 发布版 v1.6.0 原始依赖较旧的 GORM/mattn，近期主分支修复不会自动进入该标签版本。

- 默认发布的可执行文件不能要求用户另外安装 SQLite 或其他额外共享库。**Linux 默认发布包采用完整静态链接**，包含 SQLite 引擎和 C 运行库，不依赖目标机器的 libc/共享 SQLite。允许构建时使用 CGO，但不能因此发布存在未声明共享库要求的二进制。其他平台使用必需的原生系统接口，不增加单独安装 SQLite 运行库的要求。
- 默认使用驱动自带 SQLite 源码，不启用链接系统 SQLite 的构建标签。采用受控静态 C 工具链，例如 Linux 使用 musl，并为支持的架构配置对应编译器。具体发布脚本、平台支持与参数留待实现确定；当前骨架尚无数据库或静态 CGO 发布流水线。
- 对每个支持的 Linux 发布架构检查 ELF 解释器/共享库依赖，并在干净目标环境运行。仅本机编译成功不能证明独立发布；需在实际产物上验证数据库创建/重开、事务、约束与必需的 Schema 升级。
- 首个业务流程明确并检查 SQLite 连接参数：每连接外键约束、忙等待、连接池策略、日志/持久性行为和时间存储方式。选择官方驱动不替代并发与迁移语义设计。PostgreSQL 驱动在第 5.4.2 节确定；迁移工具继续暂缓。
- 本次不采用纯 Go SQLite 适配器。驱动特有类型/DSN 留在持久化适配层，授权/Core 不依赖数据库。接入依赖及嵌入引擎源码时保留实际第三方声明。

参考：[GORM 官方 SQLite 驱动](https://github.com/go-gorm/sqlite)、[v1.6.0 Module](https://github.com/go-gorm/sqlite/blob/v1.6.0/go.mod)、[mattn v1.14.52](https://github.com/mattn/go-sqlite3/releases/tag/v1.14.52)、[静态构建说明](https://github.com/mattn/go-sqlite3#cross-compiling-from-macos)。

### 5.4.2 PostgreSQL 驱动

**采用官方 `gorm.io/driver/postgres`（MIT），默认底层为 `github.com/jackc/pgx/v5`（MIT）的 database/sql 驱动。** 于 2026-10-14 核实的初始接入基线为 PostgreSQL Dialector v1.6.3，其发布 Module 依赖 GORM v1.31.2 和 pgx/v5 v5.10.0。持久化实现接入时固定并验证实际版本组合；本次文档选型不新增 PostgreSQL 运行时依赖，也不代表已实现数据库支持。

- 常规持久化使用 GORM；Dialector、DSN 与 pgx 特有细节限于持久化/基础设施适配层，业务/Core 和 Desired State 对象保持驱动无关。
- pgx 是纯 Go PostgreSQL 客户端，无需额外安装 libpq 或 C/共享库，符合独立发布要求；SQLite 仍使用前述受控静态 CGO 构建。
- 首个持久化流程设计连接池上限、连接/查询超时、TLS 验证、时区/时间表示、事务/并发行为，以及代理/预编译语句兼容性。选择驱动不等于确定这些参数，也不证明多 Controller 安全性。
- Schema 初始化/升级继续采用带版本的迁移。迁移工具仍暂缓，不因此允许生产升级只使用 AutoMigrate。

参考：[GORM 官方 PostgreSQL 驱动](https://github.com/go-gorm/postgres)、[v1.6.3 Module](https://github.com/go-gorm/postgres/blob/v1.6.3/go.mod)、[GORM 连接说明](https://gorm.io/docs/connecting_to_the_database.html)、[pgx](https://github.com/jackc/pgx)。

### 5.5 请求参数校验

**常规 HTTP 请求字段校验使用 `github.com/go-playground/validator/v10`（MIT）**，通过 Echo 的 Validator 接口集成。在独立请求 DTO 中表达必填、长度、枚举和数值范围。在首个业务接口实际使用时添加并锁定依赖；当前存活检查骨架尚未实现参数校验。

- 复杂模型约束使用可复用的 Go 校验函数。Endpoint 选择条件互斥、对象引用、地址/CIDR 解析与规范化，以及配置契约的省略/空值语义，不依赖 Echo 或仅用于 HTTP 的标签。
- 契约需要时，解析过程必须保留字段缺省、显式 `null`、空集合和合法零值之间的差异。通用 `required` 规则不能表达所有默认值或 PATCH/字段存在性规则。
- 流程权限、策略上限和涉及并发的状态检查保留在业务/持久化操作中。字段校验通过不代表有权执行操作。
- Controller 输入与 Static Manifest 输入共用模型校验函数。字段级 API 错误映射留在 HTTP 适配层；具体业务错误契约随首个接口设计。

参考：[validator](https://github.com/go-playground/validator)、[Echo 校验集成](https://echo.labstack.com/guide/request/)。

### 5.6 API 文档生成

**API 文档生成采用 Swaggo v2（MIT）**。基线为 `github.com/swaggo/swag/v2/cmd/swag` 的 `v2.0.0-rc6`，于 2026-10-13 核实为最新发布版本。虽然 GitHub 未标记为 prerelease，它仍是候选发布版。固定显式版本以复现生成结果；升级需审阅发布说明并核对输出。

- 使用 `swag init --v3.1` 和显式输入/输出路径生成 **OpenAPI 3.1** JSON/YAML。Go HTTP DTO 和 Handler 注释作为文档输入，明确参数、字段存在性/默认值、响应、稳定错误、示例和认证要求。代码/SDK 生成不是首期要求。
- 保留普通 Echo v5 Handler、validator v10 以及独立业务/Core 边界。CLI 不依赖 Echo 中间件包，也不改变运行时请求处理模型。API 文档不替代授权、校验或 Agent 配置/protobuf 契约。
- 随首个有文档的业务接口引入固定版本的构建工具、生成命令和产物位置。当前存活检查骨架没有 Swaggo 依赖、生成流水线、已发布 OpenAPI 文档或文档页面。仅提供静态 JSON/YAML 时，不必导入生成的 Go 注册包。
- 接入时在 CI 重新生成文档，使用兼容工具验证 OpenAPI 3.1 输出，并检测未提交的生成差异。核对文档中的路径/状态码、可空性、集合、枚举、认证和错误结构与实现一致；生成成功不代表契约正确。验证工具和具体 CI 命令留待实现确定。
- **文档展示与交互请求采用 Swagger UI v5（Apache-2.0）**，于 2026-10-13 核实的最新发布版本 v5.33.2 作为接入基线。Controller 提供固定版本的本地静态资源及生成的规范文件；Echo 专用包装库不是必需，使用时必须明确支持 Echo v5。运行时不依赖公共 CDN，关闭外部在线规范验证器。文档与 Try it out 请求继续遵守正常认证/授权；路由访问和生产暴露方式留待实现确定。当前骨架未接入 UI 包或文档路由。API 描述默认英文；本地化说明必须保持相同的路径、Schema 和授权语义。

参考：[Swaggo v2.0.0-rc6](https://github.com/swaggo/swag/releases/tag/v2.0.0-rc6)、[固定版本 CLI 与注释文档](https://pkg.go.dev/github.com/swaggo/swag/v2@v2.0.0-rc6)、[OpenAPI 3.1.0](https://spec.openapis.org/oas/v3.1.0.html)、[Swagger UI v5.33.2](https://github.com/swagger-api/swagger-ui/releases/tag/v5.33.2)、[Swagger UI 配置](https://swagger.io/docs/open-source-tools/swagger-ui/usage/configuration/)。

### 5.7 依赖注入与应用组装

**长期基线采用显式构造函数注入、按模块组装和集中生命周期管理**，不选用 DI 容器或代码生成框架。手工组装不是必须最终迁移到 Fx 的临时阶段；仅在实际模块组合或生命周期问题证明有必要时重新评估工具。

| 边界 | 职责与依赖方向 |
| --- | --- |
| Core / 配置模型 | 共享授权模型、选择器、有效期与校验；不依赖 HTTP、ORM、Controller 启动组装或 Linux 实现 |
| 应用服务 | 审批、Grant 和发布等用例；依赖 Core 及使用的存储/发布能力，不依赖具体适配器 |
| HTTP 适配器 | 解析/校验 DTO、获取认证身份、调用服务和映射结果；不直接执行数据库业务流程 |
| 持久化 / 外部适配器 | 使用 GORM 或外部 Client 实现所需能力；不依赖 HTTP Handler |
| 应用组装层 | 选择具体实现、创建共享资源并组合模块；业务包不能反向引用它 |
| 命令入口 | 解析启动输入并调用组装后的应用；避免 main 函数不断累积模块构造和业务代码 |

- 每个模块通过构造函数参数或专用 `Dependencies` 结构体接收必要依赖。在实际替换/适配边界定义小接口，通常放在使用方包；不为每个结构体创建接口。模块内部对象保持私有，仅暴露需要的入口和生命周期操作。不向所有模块传递整个应用或万能服务容器。
- 运行配置值与已创建的依赖分开。禁止通过可变全局服务、`init()` 注册或 `context.Context` 查找依赖。必要依赖缺失时启动失败；可选能力必须显式定义，不能因缺失而跳过授权。
- 连接池、HTTP Client 和共享服务属于进程范围，必须支持并发使用。当前身份、请求 ID 和截止时间通过方法参数/请求 Context 传递；事务由用例明确限定范围。当前申请人或事务不能存入共享 Service 字段。
- 构造函数组装对象，不偷偷启动后台循环。应用显式管理创建、准备/就绪、运行与关闭。先停止接入和依赖资源的任务，再关闭资源；失败时清理已创建/启动的部分组件，并设置关闭时限。对象创建成功不代表业务就绪或 Agent 已执行。
- 随实际功能增长拆分模块组装函数。在审查中约束导入方向，并随相关包接入 CI 依赖检查；Go 能发现循环导入，但不会检查所有架构边界。具体包名与生命周期 API 留待实现确定；本次不自建 DI 框架，也不提前创建无用途模块。

参考：[Kubernetes 配置组装](https://github.com/kubernetes/kubernetes/blob/v1.37.1/cmd/kube-apiserver/app/config.go)、[kubelet 依赖](https://github.com/kubernetes/kubernetes/blob/v1.37.1/pkg/kubelet/kubelet.go)、[Prometheus 组装与运行组](https://github.com/prometheus/prometheus/blob/v3.15.0/cmd/prometheus/main.go)、[Go 接口建议](https://go.dev/wiki/CodeReviewComments#interfaces)。

### 5.8 运行时配置加载

**运行时配置加载采用 Koanf v2（MIT）**，于 2026-10-13 核实的 `github.com/knadh/koanf/v2` v2.3.8 作为接入基线。实现运行配置加载时，添加并固定核心及实际需要的 Provider/Parser 模块；这些模块独立版本化。当前骨架仍通过标准库 `flag` 读取监听地址，尚未接入 Koanf。

- 支持运行配置 YAML、显式映射的环境变量和启动参数。覆盖顺序为 **显式传入的启动参数 > 已设置的环境变量 > 配置文件 > 默认值**。未传入参数的默认值不能覆盖文件/环境设置。逐字段定义空值语义，区分未设置与显式空值。
- Koanf 仅位于配置加载/应用组装边界。使用新实例加载，解析为带类型的 Go 配置，应用明确的默认值/校验，再通过构造函数将各模块需要的配置传入。业务/Core 包不引用 Koanf，也不查询全局配置注册表。保留配置键大小写敏感性，显式定义环境变量/参数映射。
- 拒绝未知配置键和非法值，不能默默忽略拼写错误或将错误值转换为合法默认值。显式指定文件缺失或非法时启动失败；搜索路径与可选文件行为留待实现确定。诊断可以描述配置来源，但必须脱敏敏感值。
- 运行配置描述进程如何运行，包括监听地址、数据库设置、Controller 连接、数据路径和功能开关，与 Resource/Grant/Policy/Binding Manifest 分离。Koanf 多来源合并不能把 Controller Desired State 与本地策略对象混合；Controller Mode 保持单一安全配置权威来源。
- 文件监听仅提供变更信号，不代表允许直接修改运行中的服务。显式定义可重载字段与需重启字段。重载时创建并校验完整候选，失败保留当前配置；通过生命周期负责人协调允许的变更，不让并发读取方共享可变 Koanf 对象。Manifest 目录重载继续使用独立的 Desired State 校验/Reconcile/LKG 流水线。

参考：[Koanf v2.3.8](https://github.com/knadh/koanf/releases/tag/v2.3.8)、[Provider、Parser 与合并顺序](https://github.com/knadh/koanf)、[OpenTelemetry Collector 配置实现](https://github.com/open-telemetry/opentelemetry-collector/blob/main/confmap/internal/conf.go)。

### 5.9 运行日志

**运行日志采用 Go 标准库 `log/slog`（BSD-3-Clause）**，随项目 Go 工具链版本维护，不添加外部日志依赖。初期使用标准 JSON/Text Handler；本次不选用 Zap、Zerolog 或第三方 Handler。

- 应用组装层创建 Logger，并向需要日志的模块注入 `*slog.Logger`；使用 `With` 添加稳定的组件字段。纯模型操作不必仅为符合分层形式而接收 Logger。默认 Logger 仅在进程边界为兼容性设置，不通过全局 Logger 重配置管理模块依赖。
- 开发环境默认文本输出，部署环境默认结构化 JSON。运行日志写标准错误，由宿主环境采集。当前骨架已通过启动时设置的默认 slog Logger 输出 JSON；模块注入、格式选择与运行日志级别配置尚未实现。
- 按场景统一 `component`、`request_id`、`agent_id`、`resource`、`grant_id`、`revision`、`provider_id` 和 `error` 等字段；仅在实际接入追踪时添加 Trace 关联。请求场景使用带 Context 的调用，但 Context 不会自动变成日志字段。不能把审批成功描述为已确认执行。
- 实现时通过带类型运行配置管理日志级别。`slog.LevelVar` 支持安全动态调整级别，但不代表所有运行配置均可热重载。脱敏凭据、Token 和敏感内容，不直接序列化整个请求/配置对象。
- 运行日志用于运维与排障；授权/审计记录是第 15.1 节定义的独立持久化业务证据。调整日志级别、采样或控制台输出丢失不能丢弃必需的授权记录。具体 Schema、保留周期和可靠持久化机制留待实现确定；选用 slog 不代表已实现审计存储。

参考：[Go 结构化日志](https://go.dev/blog/slog)、[slog API](https://pkg.go.dev/log/slog)。

### 5.10 RBAC 授权

**采用 Casbin 与官方 GORM Adapter，从 RBAC 开始，实际需要属性条件时再扩展 ABAC。** 身份/业务服务提供当前用户的可信组/角色；共享 Enforcer 保存组→角色绑定、角色继承及角色→资源/操作权限，不复制整个用户目录的成员关系。PicketX 按钮权限对应后端必须校验的操作，第三方应用自身权限仍独立。

复用已有 GORM 连接，保留官方 Adapter 对 Casbin 表的默认初始化/AutoMigrate；这与业务 Schema 的版本化迁移分开，也不承诺任意历史数据升级。每个进程创建并注入一个 `SyncedEnforcer`。注册策略与登录 Session 独立；IP/CIDR Grant 聚合、Lease 生命周期、网络编译与连接撤销继续由 PicketX Core 管理。

Casbin 策略仅通过定时重载同步，提供可配置的正数间隔。启动时报告实际加载的 `p`/`g` 数量与加载耗时，通过可配置的规模/成本分级建议扩大间隔或按范围装载，并解释对应的传播延迟。建议不自动改配置，扩大间隔不会减少常驻策略内存。Session 成员关系新鲜度单独约定；ABAC、决策缓存、具体配置键/默认值和规模数值阈值尚未在本次冻结。

持久化、多角色拒绝语义、规模建议和接入验收见 [ADR 0001](decisions/0001-rbac-authorization.zh-CN.md)。这是已确认选型，不代表授权已经实现。

### 5.11 运行支持组件接入基线

登录 Session 选用 SCS v2.9.0，提供内存、复用既有 GORM 的数据库、Redis Store；声明支持 Valkey 前验证实际互操作。CSRF 与内存限流使用 Echo v5 内置能力。普通 HTTP 与 Agent 通信分别配置限流参数/计数，需要时区分 Stream 建立和消息处理。

Agent 通信使用官方 grpc-go v1.84.0 与官方 protobuf，通过独立可撤销 Agent Token 认证。明文和服务端 TLS 可配置；启用 TLS 时正常验证证书。mTLS/内部 CA 签发不是首版依赖。指标使用 Prometheus client_golang v1.25.0，周期任务使用标准库，指标分别采集，模块生命周期可取消。

这些是决策，不是运行实现。Session 新鲜度、传输、代理、配额、标签基数和到期边界见 [ADR 0002](decisions/0002-runtime-support.zh-CN.md)。具体配置 Schema 与接入细节随首个实现流程确定。

## 6. Agent 与 Linux 数据面

`picketxd` 是唯一的特权数据面进程，并且必须能够脱离 Controller 独立运行。首期二进制范围为 Agent/Reconcile、nftables/conntrack 执行和受限本地接口。NFQUEUE 探测、NFLOG/镜像检测响应、REDIRECT/TPROXY 代理及业务 TLS 终止均后移，不是默认依赖；此决定替代此前默认全功能 Agent 的假设。

核心 nftables/conntrack 能力缺失属于 fatal；可选能力缺失只禁用对应功能。

| 模块 | 职责 |
| --- | --- |
| Desired State Provider | Controller Mode 从 Controller 接收状态；Static Manifest Mode 扫描本地 YAML；同一时刻只有一个权威 Provider。 |
| Controller Client | 按配置使用明文/服务端 TLS 建立 Token 认证的 gRPC Stream，接收 snapshot/delta，报告 applied status。 |
| Static Manifest Watcher | 监听目录，解析全部 YAML，完成校验并构造 Candidate Desired Snapshot。 |
| Desired/LKG Store | 保存最近可用配置、绝对过期时间、provider identity、`boot_id` 和恢复元数据。 |
| Reconciler | 将统一 Desired State 编译并应用到 PicketX 自有 nftables 对象。 |
| nftables Backend | 通过经验证的 Go Netlink Backend 操作 nf_tables。 |
| Conntrack Backend | 通过 Go ctnetlink Backend 观察和删除连接。 |
| Activity Monitor | 根据 conntrack 生命周期推导 Source/Lease 活动。 |
| Local Authorization API | 通过本地 TCP 或 Unix Socket 向可信集成提供受限逐请求决策，与管理操作分离。 |
| Drift Monitor | 结合 Netlink 事件和周期一致性扫描恢复漂移。 |
| Local Admin API | 通过 Unix socket 提供 status、doctor、dump、reconcile 和 reload。 |

### 6.1 Controller Mode 与 Static Manifest Mode

Controller 不是 `picketxd` 的运行前置条件。两种模式只改变 Desired State 的权威来源，并复用完全相同的校验、编译、协调、应用和 LKG 流水线。

```mermaid
flowchart TB
    Mode{"Configured mode"}
    Mode -->|controller| CP["ControllerProvider · gRPC stream"]
    Mode -->|static| SP["StaticManifestProvider · YAML directory"]
    CP --> Snapshot["Versioned Desired Snapshot"]
    SP --> Snapshot
    Snapshot --> Pipeline["Validate → Compile → Reconcile → Apply → LKG"]
```

以下规则属于强制约束：

1. 启动配置显式选择 `controller` 或 `static`。同一 `picketxd` 实例在任一时刻只能存在一个 Authoritative Desired State Provider。
2. **Controller Mode：**Controller 是 Security Desired State 的唯一权威来源。本地 Manifest 中的 Resource、Grant、EnforcementBinding、Policy、Authorization Rule 等安全对象永远不能与 Controller 状态合并，也不能覆盖 Controller 状态。
3. Controller Mode 仍允许本地 `/etc/picketx/picketxd.yaml`，但其中只能保存 Agent Runtime Configuration，例如 Controller endpoint、Agent 凭据和可选 TLS 配置、日志、数据目录、已启用的本地 API 监听、health/metrics endpoint 和 LKG 路径。
4. Runtime Configuration 与 Security Desired State 使用独立的配置域和模型；不表示引入 Namespace API 对象。
5. Controller Mode 下发现 Static Manifest 时，`picketxd` 默认忽略并输出显著告警；严格部署选项可以改为拒绝启动。
6. **Static Manifest Mode：**Manifest 目录是 Security Desired State 的唯一权威来源，Controller 不向该 Agent 下发策略。
7. 切换 Provider 时必须原子替换完整 Desired Snapshot，禁止把两个来源叠加或隐式合并。
8. LKG 必须记录 Provider 类型与 identity；一个 Provider 产生的 LKG 不能作为另一个 Provider 的权威状态恢复。

#### Static Manifest 语义

- 默认目录概念上为 `/etc/picketx/manifests/`，具体路径可通过 Runtime Configuration 修改。
- 目录下全部 `*.yaml`、`*.yml` 和 multi-document YAML 共同组成一份完整 Desired State Snapshot。
- Create、write、rename、delete 事件经过 debounce 后触发全目录重新扫描。
- 新快照必须完整完成解析、Schema Validation、Cross-resource Validation 和 Compile；只有完全成功的 Candidate 才能原子提交。
- 任意文件无效时继续保持上一版 LKG 和 Applied State，禁止半量应用。
- 对象标识经 API 版本规范化后为 `(kind, metadata.name)`，必须唯一；`apiVersion` 选择 Schema，不能用不同版本创建相同身份的第二个逻辑对象。重复对象导致 Candidate 失败，不允许“后加载覆盖前加载”。
- inotify/fsnotify 用于降低延迟，同时保留低频目录 rescan，防止事件丢失、编辑器 rename-write 和挂载文件系统差异造成漏更新。
- `SIGHUP` 与 `picketx agent reload` 可以主动触发扫描。
- `status` 与 `doctor` 展示 generation、source files、content hash、最后成功/失败时间和 Validation Error。
- Static 与 Controller Mode 使用相同的版本化领域 Schema，使 Resource、Policy 可以在两种模式间迁移。
- Git、Ansible、Salt、Puppet、cloud-init、Ignition、NixOS、OCI 解包或其他外部机制负责更新目录；v1 不在 `picketxd` 内嵌 Git 客户端。

当前模型的静态 Resource 示例片段（不是完整可运行策略）：

```yaml
apiVersion: picketx.io/v1alpha1
kind: Resource
metadata:
  name: ssh-admin
  labels:
    environment: production
spec:
  endpoints:
    - name: ssh
      network:
        protocol: TCP
        port: 22
```

Binding 与 Grant Scope 片段见[配置与领域模型设计](CONFIGURATION_MODEL.zh-CN.md)。单独声明 Resource 不产生访问权限，完整配置需要适用的保护绑定与获授权的权限。静态本地身份/绑定解析及完整 Grant/Policy Schema 仍需后续规范。

Static Manifest Mode 使用相同配置语义实现本地保护，限时 Grant 保存绝对截止时间，重启或重载不能重置截止时间。OIDC 自助、集中审批和跨节点聚合属于 Controller Mode。

未来可以独立设计 **Emergency Override / Break-glass**，但它不能演变成普通的 Controller + Static Merge。该能力必须具有独立 ownership domain、明确优先级、完整审计、受限范围和自动失效语义，不进入 v1。

### 6.2 nftables 所有权

- 固定使用 `table inet picketx`。
- 所有对象使用 `picketx_` 前缀；内部对象可使用 `picketx__`。
- 不修改 Docker、firewalld 或其他管理器拥有的 table。
- 推荐主 hook 位于 host namespace 的 `PREROUTING`，priority 约为 `-110`，即 conntrack 之后、DNAT 之前，从而保留外部目标地址信息。

### 6.3 Go Backend 方向

nftables 优先验证 `google/nftables` v0.3.0；当前必需的定向严格撤销优先验证 `vishvananda/netlink`。`ti-mo/conntrack` 保留为事件观测候选。使用内核 conntrack 表达式本身不需要用户态 conntrack 库；主动观测/删除才需要。

默认运行不依赖 libnftnl/libmnl/libnetfilter_conntrack 或 nft/conntrack 系统命令。M0 已在 Linux 6.17.0-1022-azure / Go 1.27.0 上验证固定版本 Backend，[17 项必需真实内核场景通过](https://github.com/PicketX/PicketXD/actions/runs/38547987642)，[Agent 验证变更已合并](https://github.com/PicketX/PicketXD/pull/1)。生产接入与其他内核兼容仍待验证。验证区间/组合集合、Batch 失败、IPv4/IPv6、绝对到期、重叠有效权限、NAT 元组和部分删除，保持 Backend 可替换。

参阅 [ADR 0003](decisions/0003-linux-backends.zh-CN.md)、[M0 设计](superpowers/specs/2026-10-14-linux-data-plane-validation.zh-CN.md)和[实施计划](superpowers/plans/2026-10-14-linux-data-plane-validation.zh-CN.md)。NFQUEUE/eBPF 依赖继续后移。

## 7. 网络策略与权限模型

有效规则元组为：

```text
Source CIDR × Destination CIDR/LOCAL × Protocol(TCP/UDP) × Port/Range
```

ALLOW 与 DENY 使用独立 membership set；在有效策略中 DENY 优先于 ALLOW。

- 使用 nftables concatenated interval set 表达 CIDR、目的地址、协议和端口范围组合，避免展开为海量离散规则。
- 同语义 membership 可以 auto-merge，但原始 Policy provenance 与 TTL 必须保存在 Controller，不能把 Kernel Set 当成审计数据库。
- 动态会话默认使用精确主机：IPv4 `/32`、IPv6 `/128`；CIDR 动态续期必须是显式高级策略。
- Policy 删除或过期时默认阻止新连接，已建立连接优雅存活。
- 撤销 Grant 时先重算有效权限并更新 Desired State；Strict 模式再使被收回访问对应的 PicketX conntrack 失效，保留无关访问及其他有效授权仍允许的访问。
- 多个 Permission 或 Lease 重叠时，过期同步方式由策略显式选择：高安全场景可即时重新编译；也可按固定周期批量刷新，以降低控制面和内核更新压力。

### 7.1 基于来源的权限聚合

对来源地址 `s` 与时间 `t`，找出 IP/CIDR 包含 `s`、有效期包含 `t`、自身条件与任何显式配置的 Lease/续约约束有效的所有 Grant。合并它们贡献的资源权限，再应用匹配的显式拒绝约束。受保护资源没有匹配的允许权限时拒绝；PicketX 不自动保护主机全部端口。

CIDR 重叠是正常情况：单主机和更大网段 Grant 可以同时贡献权限。活动续约策略中的最长前缀选择，不替代权限合集语义。某一 Grant 到期，不能删除其他 Grant 仍需要的共享内核元素，也不能把所有无关权限统一延长到最晚到期时间。

示例：同一 IP 的 Grant A 允许网站访问至 16:00，Grant B 允许数据库访问至 18:00。16:00 前两项权限均有效，之后仅剩数据库；如果另有 Grant C 仍允许网站，A 到期不会移除该项权限。新匹配的拒绝规则仍优先。

Compiler 保留贡献的 Grant ID，并按资源/规则元组计算有效期。批处理策略必须公开最大执行延迟，不能将延迟过期描述为即时回收。Controller 断连不能让限时授权永久有效；Backend 必须使用绝对过期或有界本地到期调度，满足声明的执行保证。

### 7.2 来源观测与网络拓扑

- 自助申请默认使用观测到的 /32 或 /128。CIDR 授权必须显式选择范围，并具备策略或审批权限；不把主机地址自动扩为网段。
- 门户展示授权来源、资源范围、到期时间和共享访问含义。同一 Wi-Fi 不保证来源地址相同。
- 以执行点看到的来源为执行依据。VPN 路由、SNAT、代理、IPv4/IPv6 选择和分流可能导致其不同于门户来源。接入必须建立可信映射，或明确报告不匹配。
- 仅配置过的可信代理可以提供转发来源地址；浏览器自行提交的 Header 不能证明来源身份。
- 网络授权不提供隧道、路由、加密或应用凭据；其他防火墙和应用认证仍可能拒绝访问。
- IP/端口执行无法区分共用端点的网站。网站或用户级控制需要相应协议适配器；初始 SNI/Host 分类本身不能证明用户身份。

### 7.3 跨应用共享网络权限

网络 Grant 只依据来源、资源、条件和生命周期决定可达性，不以客户端应用名称、进程 ID 或门户 Cookie 作为隐式匹配条件。来自匹配 IP/CIDR 的 IDE、数据库客户端、开发程序、CLI 和脚本可以共享同一服务的网络访问资格，无需逐应用实现 PicketX 认证。

Compiler 和 Reconciler 不为每个应用复制 Grant；不同连接分别进入正常连接生命周期，并受同一有效权限集合约束。授权记录保留原始来源与审批贡献，不因多个应用使用而推断调用应用或人员身份。其他应用若使用相同来源且匹配资源，也受相同网络权限覆盖。

门户登录用于授权流程，不要求业务客户端携带门户 Cookie。应用自身凭据、TLS 及可选协议级身份授权保持各自职责。容器、远程开发机或代理改变来源时，以执行点观测地址判断是否匹配；不从同一设备或相同开发者推导网络权限。到期、撤销和独立 Grant 叠加仍统一遵循第 7.1 与第 8 节。

## 8. Lease、活动续期与连接生命周期

### 8.1 Lease 与 Activity 分离

Controller Mode 下 Controller 是 Hard Expiry 的权威来源；Static Manifest Mode 使用绑定当前 Provider 的 Manifest 截止时间，不依赖 Controller，Agent 只报告观察到的活动。活动续期不能每包刷新 nftables timeout，也不能把数据库变成实时数据面。

| 续期策略 | 语义 | 推荐场景 |
| --- | --- | --- |
| `disabled` | 仅固定绝对过期 | 简单或高安全部署 |
| `unique-only` | 只有唯一可续期 SourceLease 匹配时才续期 | 推荐安全默认值 |
| `most-specific` | 续期最长前缀匹配 | 兼顾精确来源与管理 CIDR |
| `least-specific` | 续期最短前缀匹配 | 仅适用于特殊管理策略 |
| `all-matched` | 续期全部匹配前缀 | 最方便但风险最高 |

运行时 SourceLeaseIndex 建议使用 IPv4/IPv6 Patricia Trie 或 Compressed Radix Trie，一次路径查询即可得到 first、last 或 all matches。保留 100,000 个活跃动态来源作为后续规模探索目标，不作为首期产品验收门槛或已验证容量。

### 8.2 Conntrack 语义

- 合法新流设置 `ct mark = PICKETX_ALLOW`。
- Agent 订阅 conntrack `NEW`、`UPDATE`、`DESTROY`；事件丢失或重启后通过 dump/reconcile 恢复。
- TCP 仅允许初始 SYN（`SYN && !ACK`）创建新的 PicketX Authorization。conntrack 被删除后，旧 ACK/PSH 不能让授权“复活”。
- **Force Re-authenticate** 是有明确范围的来源会话失效操作，不同于撤销单个 Grant；必须说明收回哪些权限与连接、是否仍有其他 Grant 允许访问。它不能保证共享 IP 后的每台设备逐一重新认证，也不是 blacklist。
- 删除 conntrack 不等于发送 TCP Reset。Strict 网络撤销按规则设计阻止继续转发；协议会话终止需要相应 Adapter。Graceful 到期阻止新连接，已有合法连接可以继续。
- 活动仅可续约仍有效且具备续约资格的来源状态，不能越过 Hard Deadline；门户退出、来源会话失效和协议用户退出是不同事件。
- UDP conntrack 删除后，下一个数据报形成新 pseudo-flow，必须重新授权。

## 9. Authorization Service

Authorization Service 是首期应用执行集成，与网络准入 Backend 独立。它不代理业务流量，而是接收 Nginx、Envoy、Traefik、Caddy 等网关的授权回调，然后统一评估 PicketX Identity、Resource、Policy、WASM、Provider 与审计规则。

```mermaid
flowchart TB
    Client["Client"] -->|HTTPS| Proxy["Nginx · Envoy · Traefik · Caddy"]
    Proxy --> Backend["Application backend"]
    Proxy -->|"auth_request · ext_authz · ForwardAuth"| AuthZ["PicketX Authorization API"]
    AuthZ --> Engine["Identity · Resource · Policy · WASM · Providers"]
    Engine --> Decision["ALLOW · DENY · CHALLENGE"]
```

### 9.1 标准化授权上下文

| Context | 示例 |
| --- | --- |
| ProtocolSubject | 可信 user、groups、claims 或 service identity，以及认证来源；不能由 IP Grant 的申请人推断 |
| Source | Source IP、Lease、device context |
| Resource | Resource ID、environment、owner、labels |
| Request | Host、method、path、selected headers |
| Node | Region、environment、capabilities |
| External Attributes | CMDB、工单、值班状态、风险评级 |
| Decision Metadata | Reason、TTL、response headers、policy ID |

协议解析用于获取上下文，不会自动认证用户名。Adapter 必须区分声明身份与已验证身份，并把可信身份绑定到当前请求/会话。策略要求可信身份而无法获得时，身份条件不成立；不能退回申请人身份或仅凭 IP 放行。

资源同时配置两层控制时，网络权限和协议授权必须同时满足；协议 Allow 不扩大网络 Grant。资源也可只使用显式配置的执行集成。认证凭据与 TLS 终止由选定的可信组件负责，普通包检查无法读取不透明加密会话中的身份。

### 9.2 Adapter

- Nginx `auth_request`
- Envoy `ext_authz`（HTTP 或 gRPC）
- Traefik `ForwardAuth`
- Caddy `forward_auth`
- Generic HTTP Authorization API

PicketX 可以返回身份、组、Decision Reason、命中 Policy 等集成 Header；除非主动消费这些 Header，上游应用不需要理解 PicketX。

### 9.3 HTTPS 边界

PicketX 不需要读取加密后的业务流量。代理负责终止 TLS，并只把授权所需的身份和 HTTP 元数据提交给 PicketX。这样可以支持复杂授权，同时避免 PicketX 演变为 HTTPS Proxy、WAF 或 Service Mesh。

### 9.4 故障行为与缓存

- 每个 Resource 显式选择 fail-closed 或 fail-open；安全敏感 Resource 默认 fail-closed。
- Adapter 配置必须包含有界 Timeout。
- Allow 和 Deny Decision 只能在 Authorization Engine 返回的 TTL 内缓存，同时必须满足所选 Policy 的撤销保证。
- 授权记录保留必要的决策元数据、按需包含的可信身份来源、Policy ID、原因与 Correlation ID；默认不保存完整请求、凭据、任意 Header 或应用行为内容。

### 9.5 首期执行范围与后续探索

首期范围为 **IP/CIDR 网络准入 + 现有入口提供 HTTP/HTTPS 授权 + 本地授权接口**。网络 Grant 适用于支持的传输协议；首期应用协议集成为 HTTP/HTTPS。原生 MySQL/数据库代理与其他协议解析器不属于本次交付范围。

- 可信 HTTP 入口对每个受保护请求，根据实际资源/路由目标执行授权，包括复用连接上的后续请求。SNI 或首个请求不会授权连接上的所有后续网站；协议 Allow 不能覆盖必需的网络 Deny。可信身份和来源元数据来自受信集成，不能直接信任任意客户端 Header。
- 本地决策接口可通过本地 TCP 或 Unix Socket 提供相同授权语义，限制在相关资源/Provider 状态内，保护调用方访问，并与本地管理 API 分离。其他应用负责自己的协议认证、生命周期与转发。监听/DTO 契约留待设计，选定接口不代表已有运行端点。
- 现有入口管理 TLS 证书、终止和业务转发；仅持有证书/私钥不能旁路解密现代 TLS。Agent TLS 终止、透明代理与协议特有的 TLS 协商均暂缓。
- NFQUEUE、流量镜像与异步检测/连接响应属于后续探索，不作为另一种首期授权模式。镜像分析不能保证触发异常的操作已被阻止，加密、截断、丢失和请求复用均限制其保证。未来响应策略必须声明对象与结果：终止连接不同于拒绝单个 HTTP 请求/Stream，删除 conntrack 不等于重置 Socket，发现异常不会自动允许封禁整个 IP/端口或撤销无关 Grant。主动终止 Stream 需要具备该能力的入口集成，auth_request 本身不提供这一能力。
- 后移能力在采用前独立设计并测量资源预算；缺少这些能力不能导致首期 Agent 不可用，也不能弱化显式要求的策略。

参考：[Nginx auth_request](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html)、[HTTP/2 连接复用](https://www.rfc-editor.org/rfc/rfc9113.html#section-9.1.1)、[TLS 旁路解密限制](https://wiki.wireshark.org/TLS)、[conntrack 操作](https://netfilter.org/projects/conntrack-tools/conntrack-manpage.html)。

## 10. 插件与 WASM 架构

### 10.1 扩展分层

| 类型 | 进程边界 | 定位 | 许可证方向 |
| --- | --- | --- | --- |
| Native Plugin | 同进程、共享地址空间 | 极高性能、紧耦合数据面能力 | 默认 GPL-compatible；谨慎开放 |
| External Plugin | 独立进程，通过 Unix socket/gRPC | 企业集成、Provider、扩展逻辑 | SDK 使用 Apache-2.0；插件许可证独立 |
| WASM Policy | 沙箱运行时 | 授权和策略决策扩展 | 模块发行策略可以独立制定 |

公开生态优先使用 External Plugin 与 WASM。Native Plugin 仅用于 RPC/WASM 无法满足延迟或数据访问要求的能力，避免扩大信任范围和 Copyleft 边界。

Native 扩展初期采用受信任的内置 Go 模块，不承诺 Go 动态 `plugin` ABI。外部进程通过受限 RPC 接入，独立进程本身不等于安全沙箱，还需操作系统权限与凭据隔离。WASM 限制内存、执行量/时间和 Host API 能力；不能扩大基础策略明确拒绝的权限或延长硬截止时间。具体 WASM Runtime 尚未选定。

### 10.2 WASM 执行位置

WASM 应位于策略决策路径，而不是 Packet Fast Path。

```mermaid
flowchart TB
    Request["Normalized authorization context"] --> Base["Base policy"]
    Base --> Wasm["WASM policy chain"]
    Wasm --> Providers["Controlled external Providers"]
    Providers --> Result["Decision + reason + TTL + metadata"]
```

WASM Host API 可以提供：

- `get_subject`、`get_resource`、`get_request`、`get_source`
- `get_attribute` 与 labels
- 受插件范围限制的 `kv_get`（存储隔离，不是 Namespace API 对象）
- 通过受控 Provider Proxy 实现的 `call_provider(name, request)`
- `log` 与 `metric`
- 返回 `decision`、`reason`、`ttl` 和 metadata

默认禁止任意网络、文件系统和系统调用。所有外部访问必须经过受控 Host API，以便统一实施审计、超时、熔断和权限检查。

## 11. API 与协议

| 接口 | 协议 | 使用者 | 设计原则 |
| --- | --- | --- | --- |
| Northbound API | HTTPS REST/JSON/OpenAPI | Web、CLI、自动化 | 稳定、版本化、资源导向 |
| Southbound Protocol | gRPC/protobuf + Agent Token；TLS 可选 | Agent，包括 `picketxd` | Agent 主动连接、双向 stream、snapshot/delta |
| Authorization API | HTTP/gRPC Adapter | Nginx、Envoy、Traefik、Caddy 等 | 低延迟、明确 timeout 与 fail mode、可缓存 |
| Local Agent API | Unix socket | 本地 CLI 与诊断 | 默认不监听外部 TCP |
| External Plugin Protocol | Unix socket/gRPC | External Plugin | 公开稳定、能力注册、结构化 context |

所有公开协议必须显式版本化。向后兼容规则应进入对应协议规范，不能依赖实现中的隐式行为。

### 11.1 CLI 命令框架

**`picketx` CLI 采用 Cobra（Apache-2.0）**，于 2026-10-13 核实的 `github.com/spf13/cobra` v1.10.2 作为最新发布版接入基线。实现首批 CLI 命令时，添加并固定 Cobra 及直接使用的参数 Provider 依赖。当前仓库仅有使用标准库参数解析的 Controller 入口，尚无 `picketx` 命令树或 Cobra 运行依赖。

- Cobra 负责命令/子命令组织、参数解析、帮助、补全和命令文档生成。命令 Handler 保持轻量：转换输入，调用 API Client 或职责明确的应用操作，再格式化结果。Core 与应用服务不引用 Cobra；命令不能重复实现审批/授权规则，也不直接操作数据库/防火墙状态。
- 面向 Controller 的 CLI 命令通过版本化 Northbound API，遵守正常认证/授权。本地 Agent 诊断通过独立 Unix Socket API 和本地访问控制，不引用特权 Agent 实现包。具体命令名称、认证存储和 API Client 行为留待实现确定。
- 运行配置继续使用 Koanf。Cobra/pflag 集成不要求 Viper。仅显式修改的命令参数覆盖环境变量/文件设置，未指定参数的默认值不能改变第 5.8 节覆盖顺序。不因 CLI 框架示例引入第二套配置加载机制。
- 在组装边界为命令传入必要依赖，传递 Context/截止时间，并把错误返回入口。命令结果写标准输出，诊断/运行日志写标准错误。命令需区分已批准、执行中、已执行、失败和过时结果，不能因 API 写入成功推断已实施。输出格式和退出码契约随实际命令设计。

参考：[Cobra v1.10.2](https://github.com/spf13/cobra/releases/tag/v1.10.2)、[Cobra](https://github.com/spf13/cobra)、[Koanf 参数 Provider](https://pkg.go.dev/github.com/knadh/koanf/providers/posflag)。

## 12. 数据与状态架构

PicketX 包含三层状态：

```mermaid
flowchart LR
    Desired["Desired State · Controller or static manifests"] --> LKG["Agent LKG · recovery snapshot"]
    LKG --> Applied["Applied State · nftables · conntrack"]
```

- Reconciler 消费规范化后的本地期望状态，不依赖 Provider。控制面投影与静态 Manifest 共用版本化语义，不要求全局数据完全相同；见[配置与领域模型设计](CONFIGURATION_MODEL.zh-CN.md)。
- Static Manifest Mode 中，目录是声明式事实来源；LKG 只是最后一次成功编译和应用的恢复副本，不能反向覆盖 Manifest。
- LKG 记录 Provider 类型和 identity，避免模式切换或重启后恢复错误来源的状态。
- 核心 CIDR、TTL、重叠和 renewal 语义在 Go Core Domain Layer 实现，保证 SQLite 与 PostgreSQL 行为一致。
- SQLite 支持轻量单 Controller；PostgreSQL 支持多个 Controller 共享权威状态。
- 高频查询使用可重建的内存索引，不能为每次连接或续期扫描数据库。
- Controller 维护单调递增 revision；Agent 报告 observed revision 与 semantic hash，用于同步和漂移检测。
- 持久化保存绝对 `expires_at`，不保存只在一次 boot 内有效的相对 TTL。

## 13. 高可用、同步与一致性

- 不自行实现 Raft。
- PostgreSQL 模式下 Controller 尽量无状态，每个实例拥有自己当前连接的 Agent Stream。
- PostgreSQL `LISTEN/NOTIFY` 可用于唤醒期望状态分发；正确性依赖按 revision 重新读取。这与 Casbin 策略刷新分开，后者仅使用第 5.10 节的可配置定时重载。
- 单例定时任务可使用 PostgreSQL Advisory Lock。
- Agent 落后过多时直接接收 Full Snapshot，不无限维护复杂 Replay History。
- Agent 启动时读取 `boot_id`；宿主重启后从合法且 Provider 绑定的 LKG 重建 PicketX Table。
- nftables semantic hash 必须 canonicalize，并排除 handle、counter、remaining TTL 等易变字段。
- Controller 失联不会清除当前 Enforcement；Agent 根据 LKG 继续运行，直到状态按照绝对过期语义失效。

## 14. 安全设计

| 风险 | 架构响应 |
| --- | --- |
| Controller 被攻破 | Agent 凭据/RPC 范围、最小权限、审计、审批、可选多人变更控制；按配置启用 TLS |
| Agent 被控制 | 只有 Agent 获得 `NET_ADMIN`；Local API 由 Unix socket 权限保护 |
| Policy 误配置 | Dry-run、Policy Simulation、Revision、LKG、快速回滚 |
| Authorization 超时 | Per-Resource Fail Mode；安全敏感 Resource 默认 fail-closed |
| 后续探测/代理接入 | 独立设计并声明执行保证；所需能力缺失不能隐式绕过 |
| Plugin 风险 | External Process Isolation、WASM Sandbox、Host API Allowlist |
| 身份混淆 | IP/CIDR 是网络授权主体；流程参与者和可信协议身份分离，共享访问必须明确 |
| 重放或旧连接 | TCP 仅 SYN 创建新授权；Strict Revoke 删除 conntrack |
| 配置源混淆 | 同一时刻只有一个权威 Provider；LKG 绑定 Provider；禁止隐式 Merge |

后续应持续收紧权限分离。不需要 `CAP_NET_ADMIN` 的组件不应获得该能力；特权 Agent 的公开网络面应保持最小化。

## 15. 可观测性与运维

- **Metrics：**Policy Compile Latency、Apply Latency、Agent Connectivity、Authorization Decision Latency、Allow/Deny Count、Conntrack Reconcile、WASM Latency、Provider Error、LKG Age。
- **Audit：**Policy 修改、Lease 申请与审批、Authorization 命中 Policy、Strict Revoke、Mode 切换和 Reload 结果。
- **Logs：**结构化 JSON；按场景包含 Node、Resource、Policy、Revision、Provider Identity、Trace ID 和 Decision Correlation。
- **Health：**Controller Readiness/Liveness、Agent Local Health、Netfilter Capability Probe。
- **Doctor：**内核模块、nftables capability、conntrack、既有路由假设、本地 API 权限、配置源、Manifest Validation 和 Controller Connectivity。

### 15.1 授权记录与可见状态

记录申请人/审批人、来源 IP/CIDR、资源范围、Grant ID、申请及批准时长、策略/原因、Desired Revision、各节点 Apply 结果、到期与撤销结果。不能把来源流量归属于申请人。SQL、命令、页面历史和文件内容由应用或专门审计系统记录。

界面区分待审批、拒绝、已批准待执行、已生效、部分生效、失败、已过期和已撤销；展示最后成功观测时间、待执行变更和过时/离线节点。这些是展示语义，不冻结数据库枚举。不能仅凭审批写入成功就显示“访问已开通”；多节点收敛不是分布式原子事务。

### 15.2 前端技术与应用结构

已确定技术栈为 **React + TypeScript + Vite + shadcn/ui + Tailwind CSS + i18next + npm**。shadcn/ui 替代此前 Ant Design 6 Core 选型。初期直接使用官方组件默认外观，通过共享主题变量调整品牌风格。底层交互组件家族（Base UI、Radix UI 或官方支持的其他选择）、路由库、表单库及具体依赖版本不在本次决策中冻结。

同仓库不代表共享构建工具：Go 后端使用 Go Module，`apps/web` 使用自己的 `package.json`、锁文件和 Vite 构建。开发服务和构建任务分开，生产部署可将静态前端嵌入 Controller，或单独托管并正确配置 API 同源/代理与认证边界。当前未决定将 UI 拆成第三个 Git 仓库。

一个 Web 应用提供用户访问门户和管理区域，使用不同布局/路由，共享主题、UI 组件、翻译、API Client 和会话处理。门户覆盖资源、申请和当前权限；管理区域覆盖审批、策略、执行目标和授权记录。后端始终执行权限校验，隐藏路由或按钮不构成安全边界。

| 层次 | 目录约定 | 职责 |
| --- | --- | --- |
| 官方 UI 组件 | `src/components/ui/` | 尽量保持官方实现，通过公开参数/事件和组合使用 |
| 通用组合组件 | `src/components/shared/` | 项目分页、可复用表格、来源展示和数据新鲜度提示 |
| 业务组件/页面 | `src/features/` | 申请、审批、资源管理、API 操作和业务状态 |
| 主题与国际化 | `src/styles/`、`src/locales/` | 共享 Tailwind/CSS 变量及中英文资源 |
| API 与 Hook | `src/lib/`、`src/hooks/` | 类型化请求、取消、错误处理和轮询 |

以上是目录约定，不代表文件已创建。管理区域不另外引入第二套 UI 组件库。

### 15.3 组件功能、分页与业务逻辑

组件提供外观和文档声明的交互行为。页面通过参数和事件（例如 Button 的 `onClick`）打开弹窗或提交访问申请；业务逻辑位于官方组件源码之外。绑定点击事件本身不需要修改 `button.tsx`。

官方 Pagination 是可组合的分页导航控件；项目负责当前页、页码生成、总数/每页条数及数据请求。将其封装为一个通用项目分页组件，供各页面复用。维护这个组合组件，与逐个修改官方基础组件是不同的工作。

官方 Data Table 指南组合 shadcn Table 与 TanStack Table，实现分页、排序、筛选、选择等能力。TanStack Table 是可选的表格状态工具，与 TanStack Query 独立，不要求引入请求缓存。需要时根据官方指南组合一套项目数据表格。复杂表格确实存在集成和维护成本；shadcn/ui 不直接提供完整管理后台 CRUD 平台。

持续增长的授权记录和权限列表使用服务端分页/筛选/排序，切页时带当前查询参数请求 API。不能把已返回的一页再做本地分页，并当成完整数据集。筛选或删除后按需重置/修正页码；展示加载、空结果、失败与重试状态，阻止旧查询参数的迟到响应覆盖当前结果。

### 15.4 数据新鲜度与国际化

首期使用显式 API 请求与一个共享 `usePolling` Hook，可自行实现或采用经评估的开源实现。TanStack Query 不进入首期基线，也不要求 SSE/WebSocket。每次进入页面和相关参数改变时获取新数据；选定状态页面按可配置的有界间隔轮询，写操作成功后刷新受影响视图。

轮询必须避免请求重叠，取消或忽略过时响应，组件卸载时清理，并处理超时/失败。若隐藏页面时暂停，重新显示后立即刷新。明确展示加载/刷新中、最后成功刷新时间、错误/离线和过时状态。可保留旧数据辅助查看，但必须明显标注，不能伪装成刷新成功或执行已确认。API 响应只是某一时点的观测，不保证瞬时实时；前端刚完成请求也不能替代 Agent 自身的观测时间和版本。

i18next 管理英文（默认）与简体中文业务文本，包括标签、校验信息、分页文字和无障碍名称。数字/日期使用本地化格式，需要时接入所选日期组件的 Locale。shadcn/ui 不替代业务翻译管理，组合组件也需检查语言细节。API 错误返回稳定编码供前端翻译，避免泄露敏感信息或原始内部错误。

### 15.5 组件升级与维护边界

npm 依赖与复制到项目中的组件源码分开更新。升级 React、Tailwind 或底层交互库不会自动更新 `src/components/ui/`；运行最新版 CLI 本身也不会更新现有组件，需要执行具体变更命令。

提交当前工作后，在 Web 项目目录使用官方 CLI：

```bash
# 预览变更，不修改文件。
npx shadcn@latest add button --dry-run
# 查看源码差异。
npx shadcn@latest add button --diff
# 审查后替换选定组件源码。
npx shadcn@latest add button --overwrite
# 查看支持的特定迁移。
npx shadcn@latest migrate --list
```

`--overwrite` 会覆盖本地文件，不会自动合并保留自定义修改。`migrate` 仅覆盖已提供脚本的特定迁移，不是任意版本的一键升级。按需更新已使用组件，不为了升级而添加全部组件。检查 Git 差异、依赖/锁文件、受影响的组合组件与主题兼容性；执行类型/构建检查并验证受影响的申请/审批流程。保留 `package-lock.json`，CI 按锁文件安装；需要复现时记录本次升级使用的 CLI 版本。

业务逻辑不写入官方 UI 文件，主题集中管理，通用组合单独维护。这样可减少升级冲突，但仍需审查和验证更新。本次接受的取舍是：尽量直接使用官方组件，维护少量项目组合组件。

## 16. 部署形态

| 形态 | Controller | Agent | 适用场景 |
| --- | --- | --- | --- |
| Static Manifest | 无 | Standalone Agent 监听 YAML | 本机执行、GitOps、隔离运行 |
| Single Controller | 单 Controller + SQLite | 按已验证容量部署 | 浏览器门户、自助与基础审批 |
| Distributed Controller | PostgreSQL 支撑的 Controller | 多 Node | 跨环境集中访问管理 |
| HA Controller | 多 Controller + PostgreSQL HA | 按实测容量部署 | 可用性要求，不取决于组织规模 |
| Cloud | 托管 Controller | 客户 Node 主动连接 | 后续托管运行 |
| OEM | 厂商或 PicketX 控制面，或静态模式 | 嵌入式 Agent | 后续设备/平台集成 |

这些是部署选择，不是组织规模限制，也不是已经验证的容量承诺。

默认在服务主机安装 Agent。显式网关 Binding 可控制既有路由上的转发流量，不使 PicketX 成为路由器或应用代理。主机/容器路径覆盖及网关 NAT 匹配需要适配器专项校验，均不得暗中扩大绑定范围。Controller/Static Mode 决定配置权威来源，与实施位置独立。

容器化 Agent 如需控制 Host Firewall，应优先使用 Host Network + `CAP_NET_ADMIN`，避免直接使用完整 `privileged`。nftables 与 Netfilter 受 Network Namespace 约束，部署时必须明确操作 Host Namespace。

## 17. 许可证与代码边界

| 组件 | 默认许可证或授权 | 理由 |
| --- | --- | --- |
| `picketxd` | GPL-3.0-or-later + 可替代的 OEM Commercial License | 保持数据面开放，同时保留商业 OEM 路径。 |
| Controller | Apache-2.0 | 促进企业集成、二次开发和生态采用。 |
| Core / Web | Apache-2.0 | Core 不依赖 Agent 实现，Web 与控制面同仓库。 |
| CLI | Apache-2.0 | 避免在主要工具入口制造 Copyleft 摩擦。 |
| Protocol / Public SDK | Apache-2.0 | 允许任意语言和商业系统集成。 |
| External Plugin SDK | Apache-2.0 | 最大化插件生态采用。 |
| Native Plugin ABI | 位于 GPL Agent 边界内 | 同进程插件默认需要 GPL-compatible，除非未来另行完成法律与许可证设计。 |

### 17.1 贡献者机制

为了保留 `picketxd` 同时使用社区 GPL 和经批准 OEM Commercial License 分发的能力，贡献者保留其贡献版权，但应向项目主体授予为上述发行和经批准再许可所需的永久、全球、不可撤销、可再许可权利。

正式 CLA 或 Contribution Agreement 发布前必须由具备开源许可证经验的律师审核。本文记录的是产品意图，不构成法律意见。

### 17.2 第三方依赖与 OEM 边界

OEM Commercial License 只能覆盖项目有权再许可的代码。如果商业二进制仍链接 libnftnl 或 libnetfilter_conntrack，项目方不能替 Netfilter 版权人豁免其 GPL 条件。

Go Backend 方向不再默认引入这些 C 库，但“纯 Go”不是许可证结论。OEM 交付仍需逐项核验实际依赖、复制代码和授权权利；Backend 可替换性保留。

## 18. 分阶段实施路线

M0、M1.1、M1.2 与隔离 M1.3 桥接已实现并合并。[M1.3 实施计划](superpowers/plans/2026-10-15-core-m0-bridge.zh-CN.md)链接实际执行的 Agent 增量。M2A 提供静态配置检查；已合并的 PicketXD M2B 提供持久期望准备与主机 INPUT 验证。生产执行、同步、七层与可用管理端仍在[基本功能路线图](BASIC_FUNCTION_ROADMAP.zh-CN.md)中。保留实际测试证据，已定架构仅在发现具体冲突时重新讨论。

产品基线固定目标与边界，不代表所列能力均已实现。历史范围保留为架构选项；协议适配和企业集成按已验证需求分阶段推进。

| 阶段 | 目标 | 范围 |
| --- | --- | --- |
| M0 数据面验证 | 证明来源执行可靠 | nftables/conntrack、IPv4/IPv6、重叠 IP/CIDR Grant、Hard Expiry、Strict/Graceful 撤销、重启和 Linux Network Namespace 行为 |
| M1 可用访问流程 | 完成申请、批准、生效、到期 | 浏览器/OIDC 自助、基础审批、来源 Grant、权限合集、Resource 范围、状态和授权记录；Controller/SQLite/Agent/CLI 与 Static Manifest Mode；HTTP/HTTPS 入口授权和受限本地决策 API |
| M2 运维规模化 | 可靠多节点运行 | PostgreSQL、按配置传输的认证 gRPC Stream、labels/selectors、drift/reconcile、LKG 恢复、各节点状态；按需 HA |
| M3 协议扩展 | 按需增加控制精度 | 更多可信 Adapter 和协议用户/会话上下文；原生协议代理与探测逐个评估 |
| M4 扩展与交付形态 | 按实际需求扩展 | WASM、External Provider/Plugin、高级审批、Cloud/OEM 和商业打包 |

首期 Agent 范围替代此前全功能模型：NFQUEUE、镜像响应、透明代理和 TLS 终止均后移。HTTP/HTTPS 入口授权与本地决策接口属于首期范围，可在网络准入之后实现；各阶段均不要求实现通用代理、完整行为记录或所有协议解析器。

基础审批与授权记录属于 M1，不仅属于未来企业版本。大规模 Benchmark 属于工程探索，不是首个可用产品的用户采用或商业验收门槛。

首个端到端场景选择 Gitea：门户授权来源后，同一匹配来源的浏览器、Git CLI 与 IDE 可访问已声明的 HTTP(S)/SSH 入口，继续使用 Gitea 原有凭据。Git 客户端无需持有门户 Cookie；未授权来源及到期后的新连接应被拒绝，其他有效 Grant 与已有连接按既定语义处理。仅开 HTTP(S) 入口不隐式开放 SSH，反之亦然。

## 19. 风险与设计约束

| 风险 | 后果 | 控制措施 |
| --- | --- | --- |
| 范围膨胀为 Cilium、WAF 或 Envoy | 项目规模不可持续 | 维护明确非目标；只检查做出决策所需的最少上下文 |
| GPL 与 OEM 冲突 | Proprietary Distribution 被第三方依赖阻塞 | Backend 隔离、贡献协议、实际依赖审查与可替换 Netlink Backend |
| 把 IP 当成人员身份 | 错误归属共享来源流量 | 显式来源授权、申请来源记录、独立可信协议身份 |
| Authorization Latency | 影响每次 Web 请求 | Cache、Short-circuit、Local/Edge AuthZ、明确 Timeout Budget |
| Policy 重叠或冲突 | 误放行或误拒绝 | Deny 优先、Compiler Validation、Simulation、Provenance、Audit |
| 配置源冲突 | 运维人员误判实际策略 | 唯一权威 Provider、禁止 Merge、显著告警、Provider-bound LKG |
| 大规模状态 | Controller 或 Agent 内存压力 | Per-node Compile、Trie、Snapshot/Delta、100k 与 1M Benchmark |
| 后续探测/代理过载 | 丢包或服务中断 | 在字节/流/时间预算、容量和失败策略设计及测量后再采用 |

## 20. 附录

### 20.1 推荐双仓库 Go 布局

```text
PicketX/                       # Apache-2.0
  go.mod
  cmd/picketx-controller/
  cmd/picketx/
  pkg/core/
  pkg/protocol/
  pkg/pluginapi/
  internal/controller/
  internal/storage/
  apps/web/
  docs/

PicketXD/                      # GPL-3.0-or-later + OEM option
  go.mod
  cmd/picketxd/
  internal/runtime/
  internal/provider/
  internal/reconcile/
  internal/backend/nftables/
  internal/backend/conntrack/
  internal/inspector/
  internal/localapi/
  docs/
```

这是建议目录结构，Module 边界由实现 ADR 明确；未在本次文档更新中创建或迁移仓库。

### 20.2 推荐配置目录

```text
/etc/picketx/
  picketxd.yaml             # 仅保存 Agent Runtime Configuration
  manifests/                # Static Manifest Mode 的 Security Desired State
    resources/
    grants/
    bindings/
    policies/
    authz/
```

### 20.3 后续规范文档

本架构应通过独立的聚焦文档继续细化，而不是无限扩张本文：

- [配置与领域模型 v0.1.0](CONFIGURATION_MODEL.zh-CN.md)：当前模型决策与待决 Schema
- Static Manifest Schema 与 Validation Rule
- Policy Evaluation 与冲突语义
- Southbound Protocol 规范
- Authorization API 与 Adapter Contract
- nftables 编译与所有权规范
- Lease 与 Activity Renewal 规范
- Threat Model
- Plugin Protocol 与 WASM Host API
- Contribution Agreement 与 Licensing Policy

### 20.4 参考资料

- [GNU GPL FAQ](https://www.gnu.org/licenses/gpl-faq.en.html)：GPL 版本兼容、链接、多许可证和额外授权边界。
- [Apache License v2.0 与 GPL 兼容性](https://www.apache.org/licenses/GPL-compatibility)：Apache-2.0 与 GPLv3 兼容，但与 GPLv2-only 不兼容。
- [libmnl](https://netfilter.org/projects/libmnl/)：官方 Netlink Helper Library 与 LGPL-2.1+ 许可证信息。
- [libnftnl](https://www.netfilter.org/projects/libnftnl/)：官方 nf_tables 低层 API 与许可证信息。
- [libnetfilter_conntrack](https://netfilter.org/projects/libnetfilter_conntrack/)：官方 conntrack API 与许可证信息。

- [shadcn/ui introduction](https://ui.shadcn.com/docs)
- [shadcn CLI](https://ui.shadcn.com/docs/cli)
- [shadcn Pagination](https://ui.shadcn.com/docs/components/pagination)
- [shadcn Data Table](https://ui.shadcn.com/docs/components/data-table)
- [Go nftables candidate](https://github.com/google/nftables)
- [Go conntrack candidate](https://github.com/ti-mo/conntrack)
- [Go NFQUEUE candidate](https://github.com/florianl/go-nfqueue)

## 修订记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.8.2 | 2026-10-14 | 记录 SCS、内置 CSRF/限流、官方 gRPC/protobuf、Token 认证与可选 TLS、Prometheus、标准库周期任务；明确 Linux Backend 验证优先级，提出 M0 执行设计/计划；运行接入和内核验证待完成。 |
| v0.8.1 | 2026-10-14 | 确定 Casbin RBAC 与官方 GORM Adapter；当前用户成员关系与共享规则分离；仅定时策略同步、间隔可配置，启动按规模/成本提供建议；记录双语 ADR 0001；接入待实现。 |
| v0.8.0 | 2026-10-14 | 确定官方 GORM PostgreSQL/pgx 驱动；首期收敛为网络准入、外部 HTTP/HTTPS 入口授权和本地决策接口；探测、透明代理/TLS 与原生数据库代理后移；实际接入待实现。 |
| v0.7.9 | 2026-10-13 | 确定官方 GORM SQLite/mattn 驱动及默认独立发布、Linux 完整静态链接要求；PostgreSQL 驱动与迁移工具待选，接入和产物验证待实现。 |
| v0.7.8 | 2026-10-13 | 确定 Cobra v1.10.2 作为 CLI 接入基线；明确 Koanf 参数覆盖、API/本地诊断边界与输出规则；命令实现待接入。 |
| v0.7.7 | 2026-10-13 | 确定 slog 作为运行日志方案；明确 Logger 注入、格式、字段、动态级别及与独立授权审计的边界。 |
| v0.7.6 | 2026-10-13 | 确定 Koanf v2.3.8 作为运行时配置接入基线；明确覆盖顺序、带类型注入、重载与策略来源隔离；尚未接入运行代码。 |
| v0.7.5 | 2026-10-13 | 确定手工构造函数注入、模块组装与集中生命周期管理为长期基线；明确分层、依赖方向与作用域，不引入 DI 框架。 |
| v0.7.4 | 2026-10-13 | 确定 Swaggo v2.0.0-rc6 生成 OpenAPI 3.1 文档；确定 Swagger UI v5.33.2；记录固定构建工具版本、契约检查及本地文档服务边界。 |
| v0.7.3 | 2026-10-13 | 确定 HTTP 字段使用 validator v10，模型规则复用校验函数；保留启动自动升级数据库目标，暂缓迁移工具选型。 |
| v0.7.2 | 2026-10-13 | 确定 Controller 持久化使用 GORM；记录双数据库、模型分离、事务与版本化迁移边界，未添加数据库运行时行为。 |
| v0.7.1 | 2026-10-12 | 确定 Controller HTTP 使用 Echo v5；记录错误处理、校验及框架边界；迁移存活检查骨架。 |
| v0.7.0 | 2026-10-08 | 分离业务/配置/执行模型；明确多入口 Resource、目标地址数组、多资源/多 Agent 绑定、按节点投影和不引入 Namespace 的隔离；替换旧静态示例。 |
| v0.6.0 | 2026-10-01 | 统一 Authorization Core 与可扩展 Agent/Adapter；确定 PicketX/PicketXD 双仓库与 Go；前端改为 shadcn/ui/Tailwind，补充组件组合、升级、i18n 与新鲜度约束；加入 Gitea 验收。 |
| v0.5.1 | 2026-09-30 | 增加跨应用访问同一服务场景；明确来源授权不依赖客户端应用或门户 Cookie，以及协议认证和出口变化的边界。 |
| v0.5 | 2026-09-30 | 固化简单、无需专用客户端的来源授权定位；分离流程与协议身份；明确 Grant 聚合和共享访问；限定授权记录范围；基础审批进入首期流程；保留 Linux Backend、配置源隔离和许可证决策。 |
| v0.4 | 2026-09-02 | 之前的双语架构基线，原文保存在 历史文档归档。 |

文档优先级和归档清单见 [README.zh-CN.md](./README.zh-CN.md)。具体 API/Schema 仍由实现 ADR 细化；仓库仍处于初始化阶段，实际实现状态见根目录 README。
