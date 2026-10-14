# ADR 0002：运行支持组件基线

> [English (default)](0002-runtime-support.md) | 简体中文

- 日期：2026-10-14
- 状态：选型已确认；运行代码接入待实现
- 上位依据：[架构](../ARCHITECTURE.zh-CN.md)第 5.11、11 节

## 背景

Controller 与 Agent 需要 Session、请求保护、通信认证、指标和周期任务。这些决策补齐支持组件基线，不代表已实现，也不冻结完整配置 Schema。

## 决策

| 范围 | 已选基线 | 边界 |
| --- | --- | --- |
| 登录 Session | `github.com/alexedwards/scs/v2` v2.9.0 | 内存、复用已有 GORM 的数据库、Redis Store |
| CSRF | Echo v5 内置 Middleware | 浏览器/Cookie 认证请求；明确可信 Origin 与 Token 传递方式 |
| 限流 | Echo v5 内置内存 Store | 普通 HTTP 与 Agent 通信独立配置、独立计数 |
| Agent RPC | `google.golang.org/grpc` v1.84.0 和官方 `google.golang.org/protobuf` | Agent 主动建立双向 Stream；接入时固定 protobuf 工具版本 |
| Agent 认证 | 每 Agent 独立、可撤销的 Token | 从凭据确定身份和资源范围；TLS 可配置 |
| 指标 | `github.com/prometheus/client_golang` v1.25.0 | Controller 与 Agent 分别采集；指标暴露可配置 |
| 周期任务 | Go 标准库 Timer/Ticker | 可取消生命周期、间隔可配置、同一任务不重叠执行 |

### Session 与 CSRF

使用 SCS memstore、复用既有 GORM 连接的 gormstore，或 goredisstore。Valkey 必须验证实际客户端/服务端组合的互操作性；尚未确认独立的 Valkey Store。Echo v5 需要薄适配层，并针对选定版本验证。

提供绝对期限和空闲期限；登录更新 Session Token，退出销毁 Session。可信组/角色成员关系新鲜度、高效撤销用户全部 Session 由业务层负责；策略重载不会刷新 Session 成员关系。切换 Store 配置不等于迁移现存 Session。

相关浏览器路由使用 Echo 内置 CSRF，验证固定版本的实际行为，包括登录/回调范围。Cookie 安全配置匹配 HTTP/TLS 部署方式。Session 表初始化与升级行为在接入时明确；业务 Schema 继续要求版本化迁移。

### 普通请求与 Agent 通信

普通 HTTP 与 Agent 通信分别配置启用、速率、突发量和计数状态。认证后按已验证用户/Agent 身份生成键；认证前使用明确可信的来源地址提取方式。

Echo HTTP 请求限流不会统计长连接中的 gRPC 消息；Stream 建立和消息处理分别接入。内存计数按进程独立，不是集群全局配额。

### Agent 身份与传输

支持明文 gRPC 与服务端 TLS。每个 Agent 使用独立 Token，Controller 从已验证凭据确定身份，不接受调用者自行声明的 Agent ID。支持撤销和轮换凭据；禁用 Agent 时拒绝新调用并关闭既有控制流。

明文链路不提供保密性或服务端身份验证，Token 与数据在该跳可被读取。说明受控网络/已有加密隧道部署方式，公网直连建议使用 TLS。启用 TLS 时正常校验服务端证书。支持外部终止 TLS，但必须明确可信代理边界。

mTLS 属于未来可选认证方式，不是首版要求；本次不选择内部 CA、Agent 证书签发或自动续期子系统。

### 指标与周期任务

只通过已配置 Listener 和访问控制暴露指标，不暗含新增默认 Agent 公网端口。外部 Prometheus/Grafana 可选。标签基数保持可控；用户/IP/Session 标识放在日志或授权记录中，不展开成逐个时间序列。

Casbin 优先复用能够保留策略范围的周期加载能力；过滤加载的重载仍须保留过滤条件。其他模块持有可取消的周期任务，记录失败和耗时。需要跨 Controller 仅执行一次的任务，在引入时使用数据库协调。

Ticker 不是可靠持久化任务队列。清理记录不等于执行授权到期：Controller 不在线时，内核或 Agent 本地期限机制仍须符合已声明的过期契约。

## 接入时再明确的内容

具体配置键/默认值、迁移、外部 TLS 拓扑、Token 注册/轮换协议、protobuf 版本和 Agent 指标 Listener 随首个实现流程细化，不改变已有产品与领域边界。

## 参考

- [SCS](https://github.com/alexedwards/scs)
- [Echo Middleware](https://echo.labstack.com/docs/middleware)
- [gRPC 认证](https://grpc.io/docs/guides/auth/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Go time](https://pkg.go.dev/time)
