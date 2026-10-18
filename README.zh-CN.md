# PicketX

[English (default)](README.md) | 简体中文

**简单易用、无需专用客户端的按需访问授权系统。**

PicketX 通过自助申请、策略和审批，为受保护资源提供有期限的访问权限。IP/CIDR 网络权限可由匹配来源的设备和应用共享，服务自身的认证仍然保留。后续身份级授权通过可信 Adapter 集成。

## 项目状态

当前仓库是**开发初始化骨架**，包含共享逻辑授权与物理规划库、基于 Echo v5 的 Go Controller 存活检查和中英文 React 起始页。尚未实现认证、审批、Grant 签发、数据库、Agent 同步或执行功能，不能用于保护实际资源。

控制面与 Web 位于本仓库；官方 Linux Agent `picketxd` 属于独立的 **PicketXD** 仓库。本次初始化未创建或修改该仓库。

后端技术栈已确定为 Go + Echo v5 + GORM，请求字段校验采用 go-playground/validator v10。API 文档生成采用 Swaggo v2.0.0-rc6，目标规范为 OpenAPI 3.1，文档展示与交互请求采用 Swagger UI v5.33.2。GORM/SQLite已通过显式迁移持久保存接受的发布记录；业务表、validator、Swaggo和Swagger UI仍待接入。

依赖组装采用显式构造函数注入、按模块组合和集中生命周期管理，不引入 DI 框架；设计边界见架构第 5.7 节。

运行配置加载选用 Koanf v2.3.8，通过带类型配置传入模块，并明确文件/环境变量/参数覆盖顺序。已接入Controller；只有显式变更的标准库参数才覆盖同一配置加载边界。

运行日志使用标准库 `log/slog`。当前骨架已向标准错误输出 JSON；模块 Logger 注入、格式选择、级别配置和独立持久化授权审计尚未实现。

`picketx` CLI 使用 Cobra v1.10.2，现提供只读配置检查。Controller已接入Koanf进程配置；经过认证的管理命令仍待实现。

SQLite 选用 GORM 官方驱动（v1.6.0 基线）和 mattn/go-sqlite3（v1.14.52 基线）。默认 Linux 发布包必须完整静态链接，无额外共享库安装要求。PostgreSQL 采用官方 gorm.io/driver/postgres v1.6.3，底层 pgx/v5 v5.10.0。Controller发布记录及静态SQLite实际运行检查已实现；PostgreSQL、业务迁移与发布打包仍待接入。

首期执行范围为 IP/CIDR 网络准入、现有入口提供 HTTP/HTTPS 授权、本地 TCP 或 Unix Socket 决策接口。NFQUEUE/镜像探测、透明代理/TLS 终止及原生数据库代理均后移；这些端点与集成尚未实现。

授权选用 Casbin RBAC 和官方 GORM Adapter；业务层提供当前用户的可信组/角色，共享规则在进程内鉴权，按可配置周期同步，启动时按规模/成本提供建议。ABAC 按需扩展。这些能力尚未实现，见 [ADR 0001](docs/decisions/0001-rbac-authorization.zh-CN.md)。

运行支持已选 SCS Session、Echo 内置 CSRF、普通/Agent 分别配置的限流、官方 gRPC/protobuf、独立 Agent Token 与可选服务端 TLS、Prometheus 指标、标准库周期任务；Controller 侧同步流基础已实现（见下文），其他运行时仍待接入，见 [ADR 0002](docs/decisions/0002-runtime-support.zh-CN.md)。

M0 Linux 数据面验证已在 [PicketXD](https://github.com/PicketX/PicketXD/pull/1) 实现并合并。[最终已测源码](https://github.com/PicketX/PicketXD/commit/13fdf2d9aacd79be74578ebe9c23af19a6b33d6a)在 Linux 6.17.0-1022-azure / Go 1.27.0 上通过 [17 项必需真实内核场景](https://github.com/PicketX/PicketXD/actions/runs/38547987642)。这是可复用验证基础设施，尚不是生产 Agent。已批准的 [M1.1 设计](docs/superpowers/specs/2026-10-15-shared-network-core-design.zh-CN.md)与[实施计划](docs/superpowers/plans/2026-10-15-shared-network-core.zh-CN.md)已有仅依赖标准库的逻辑引擎 [`pkg/core`](pkg/core)：校验来源、选择器、入口范围和明确期限，合并匹配 Grant、优先处理 deny，并解释独立贡献与未来变化。Gitea HTTPS/SSH 固定断言覆盖重叠、撤销、不可变替换和重放不延期。Controller 工作流尚未实现；逻辑允许不代表 Agent 已实际放行。

M1.2 已实现并经 [PR #7](https://github.com/PicketX/PicketX/pull/7) 合并。已批准的 [M1.2 设计](docs/superpowers/specs/2026-10-15-physical-network-planner-design.zh-CN.md)与[已执行计划](docs/superpowers/plans/2026-10-15-physical-network-planner.zh-CN.md)现有仅依赖标准库的物理规划层 [`pkg/networkplan`](pkg/networkplan)，消费 Core 独立解析视图。它校验明确 TCP/UDP 单端口定义、可信上下文边界/能力、Binding 笛卡尔积与物理冲突；输出不可变且独立持有的清单和显式时间 allow/deny 投影，并对完整替换提供约束及旧目标撤回来源记录。入口 nil 地址使用明确上下文边界，非 nil 空地址报错；离线目标仍保留，未绑定入口不受管理。NextChange 包含未来 allow/deny 边界，不延长绝对期限。M1.1 省略开始时间时的历史时钟边界问题已修正。

每份计划只有一个权威来源，不合并 Controller/static 计划。上下文事实不能扩大请求的类别/阶段/目的范围。改指向但保留同类型 Grant/Deny ID 时拒绝；新 ID 的审批关联及原期限保留仍由发布方负责。清单和撤回记录只是期望历史，不证明已经安装，也不是删除清单。独立 M1.3 PicketXD 桥接现已验证具体 guard/原始元组、夹具边界调度、实际应用历史、先确认应用再严格范围清理，以及明确空所有权清理。纯 Go 契约验证不证明主机/容器或数据包兼容性。用法见[包说明](pkg/networkplan/doc.go)及外部包 `ExampleCompile`。

M1.3 已在 [PicketXD PR #3](https://github.com/PicketX/PicketXD/pull/3) 完成真实内核验证，[最终推送版本 CI](https://github.com/PicketX/PicketXD/actions/runs/38750455517) 的 17 项 M0 + 19 项必需桥接场景全部通过。它验证隔离 transit 夹具中的共享规划应用和限定撤回；生产 host/container 部署仍待实现。

M2A 新增 [`pkg/configuration`](pkg/configuration) 和 `picketx config check`：严格版本化 YAML、固定资源 UID 引用、独立绝对期限、完整规划校验/替换及显式空撤回。参见[可运行 Gitea 示例](examples/static/README.zh-CN.md)。这是可用的配置检查入口；运行时提供器/LKG、Agent 部署与同步、七层及管理端仍未实现。

M2B 已在 [PicketXD PR #4](https://github.com/PicketX/PicketXD/pull/4) 合并：`picketxd prepare` 可持久恢复静态期望、重载完整配置并调度绝对时间边界。[最终 CI](https://github.com/PicketX/PicketXD/actions/runs/38760479190) 通过全部 43 项必需内核场景，包括 7 项主机 INPUT 场景。准备命令报告 `enforcementConfirmed=false`，生产执行仍待实现。[M1.3 实施计划](docs/superpowers/plans/2026-10-15-core-m0-bridge.zh-CN.md)保留并链接实际执行的 Agent 增量，后续沿[基本功能路线图](docs/BASIC_FUNCTION_ROADMAP.zh-CN.md)推进。

M2D1 已实现 [Controller Agent 同步流基础](docs/AGENT_STREAM.zh-CN.md)：公开类型化 gRPC/protobuf、节点完整投影、可撤销/轮换 Token、应用报告关联和有界生命周期；真实流测试覆盖 TCP 回环和正常 TLS 校验。下文M2D2接入生产进程与持久revision；PicketXD Controller Provider和注册管理仍待接入，尚不能声明完整Controller Mode可部署。

M2D2 新增 [可运行Controller发布服务](docs/CONTROLLER_RUNTIME.zh-CN.md)：严格Koanf配置、GORM/SQLite持久版本、独立凭据重载/撤销、可选服务端TLS及有界HTTP/gRPC生命周期。CI启动真正静态SQLite二进制，检查真实gRPC快照、就绪、持久revision和SIGTERM。Agent远程Provider、管理端和L7继续推进，进度见[自动交付检查点](docs/AUTONOMOUS_DELIVERY.zh-CN.md)。

## 本地开发

需要 Go 1.27+、Node.js 24 LTS 和 npm。

在仓库根目录启动 Controller：

```bash
go run ./cmd/picketx-controller
```

默认监听 `127.0.0.1:8080`，通过 `-listen` 修改地址。`GET /healthz` 仅表示进程存活，不表示授权功能已就绪。

另开终端启动 Web：

```bash
cd apps/web
npm ci
npm run dev
```

访问 Vite 输出的地址。前后端独立构建，Vite 将 `/healthz` 代理到本地 Controller。Controller 暂未托管 Web 静态资源。

## 验证

```bash
go test ./...
go vet ./...
go build ./...

cd apps/web
npm run typecheck
npm run lint
npm run build
```

## 目录与设计

| 路径 | 用途 |
| --- | --- |
| `cmd/picketx-controller` | 本地开发服务入口 |
| `internal/controller` | HTTP Handler 与测试 |
| `pkg/core` | 逻辑来源到入口的授权及已解析成员 |
| `cmd/picketx`、`internal/cli` | 只读 Cobra 配置检查 |
| `pkg/configuration` | 严格 alpha 策略解码与不可变编译候选 |
| `pkg/networkplan` | 有明确范围的每 Agent 物理意图及替换约束 |
| `apps/web` | React、TypeScript、Vite、shadcn/ui、Tailwind CSS、i18next |
| `docs` | 最新中英文产品、架构与配置模型基线 |

`pkg/core` 提供演进中的网络逻辑 Go API 与可运行的 `ExampleCompile`；公开外部协议包仍暂缓。参阅[产品设计](docs/PRODUCT_DESIGN.zh-CN.md)、[架构设计](docs/ARCHITECTURE.zh-CN.md)、[配置模型](docs/CONFIGURATION_MODEL.zh-CN.md)和 [Web 开发说明](apps/web/README.md)。

## 许可证

项目代码使用 [Apache-2.0](LICENSE)。复制的第三方组件保留原有声明，见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。PicketXD 采用独立的 GPL-3.0-or-later/OEM 授权设计。
