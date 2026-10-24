# 数据库权威配置管理 API

> [English（默认）](MANAGEMENT_API.md) | 简体中文

本段已实现单 SQLite Controller、独立 API 主体凭据、Casbin RBAC、完整期望配置
读取/替换、修订号前提与事务变更审计。SCS/OIDC 浏览器登录、自助申请/审批、HTTP L7、
真实管理页面、PostgreSQL 和多 Controller 尚未实现，不能据此宣称产品交付完成。

## 唯一权威来源

显式运行配置文件默认 `source.mode: file` 保持兼容且没有管理路由；正常无配置启动见[本地开发](LOCAL_DEVELOPMENT.zh-CN.md)。API 使用新的私有数据库，显式配置：

```yaml
http: {listen: "127.0.0.1:8080"}
agent:
  enabled: true
  listen: "127.0.0.1:9090"
  authority: "controller:example"
source: {mode: database}
database: "var/managed/controller.sqlite"
credentials: "/etc/picketx/agent-credentials.json"
management:
  credentials: "/etc/picketx/management-principals.json"
  reload: "30s"
  warn_rules: 10000
  warn_load_percent: 50
  requests_per_second: 128
  burst: 256
poll: "1s"
shutdown: "5s"
```

数据库模式拒绝 policy/facts 文件路径，绝不轮询或回退文件策略；Agent 凭据仍独立轮询。
新库以 revision1 持久化显式空配置与 system:bootstrap 审计。业务 schema2 绑定来源模式；
旧 schema1 中已有发布按文件权威迁移，不能静默改为管理权威。保留旧库/来源，有意初始化
另一个管理数据库；尚无原地切换命令。不得为切换模式删除在线数据库；既有显式撤权和
Agent 恢复边界仍适用。

HTTP 本身不提供 TLS。绑定回环并使用可信 HTTPS 入口，或受控加密链路；不能在不可信明文
网络传 Token。管理与 Agent 凭据独立，不能复用 Token。API 不信任转发身份头，也没有
CORS、Cookie、Session 或登录接口。

## 主体和 RBAC

凭据文件由可信主机配置所有者控制，必须是0600 私有常规文件，最大1MiB。结构示意：

```json
{"principals":[{"id":"principal:operator","sha256":"<64位十六进制SHA256>","roles":["role:admin"]}]}
```

使用密码学随机 Token，原文保存在调用方私有秘密存储中，不进 Git、URL、进程参数或 shell
历史。文件仅存 SHA256；发送单个 Authorization: Bearer 头。Token 为32–256字节无空白的
可打印 ASCII。最多128主体、每主体16角色、ID 最长128字节。主体前缀 principal:，角色/
组前缀 role:/group:；后缀仅 ASCII 字母、数字、横线、下划线、点。重复ID/摘要/角色与
歧义、未知字段、null JSON 都拒绝。空主体列表有意关闭认证访问。

启动和每次请求都校验文件，移除主体/更改角色影响下一次请求；已授权请求不追溯取消。
应原子替换并保留权限。损坏/不可读返回503及 not-ready，不继续使用缓存认证成功。

单个 Casbin3.11.0 SyncedEnforcer 通过官方 GORM adapter3.41.0 复用 SQLite 连接。
主体角色来自可信凭据，不把用户目录成员关系写进 Casbin。共享 g 规则支持组→角色与角色
继承，所有适用主体一次计算，deny 优先。仅首次且策略为空时初始化：

| 角色 | publication:read | publication:write | audit:read |
| --- | --- | --- | --- |
| role:admin | allow | allow | allow |
| role:viewer | allow | 默认拒绝 | allow |
| role:blocked | deny | deny | deny |

权限覆盖完整发布，是全局操作权限，不是按资源委派或租户隔离。admin+blocked 仍拒绝。
Adapter 只自动初始化自己的 Casbin 表，业务迁移独立版本管理。仅按配置正间隔重载共享
策略，最长24小时；可信直接改库不是公共管理接口，只在成功重载后可见。重载失败停止
授权并使就绪失败，直至恢复。启动记录p/g数量、耗时；阈值给出扩大间隔/收敛范围建议，
绝不自动改配置。扩大间隔延长撤权传播，不减少常驻内存。

普通 HTTP 限流按实际 socket 对端 IP，不接受转发头；入口代理后的调用者共享代理桶。
Agent 建流/消息计数独立。SCS浏览器会话、CSRF和成员关系新鲜度属于后续段。

## HTTP 契约

响应 Cache-Control: no-store，错误为稳定 `{"code":"…"}`，不暴露秘密、SQL或内部编译
错误。所有操作后端鉴权：

| 路由 | 权限 | 结果 |
| --- | --- | --- |
| GET /api/v1/me | 已认证 | 可信ID/角色/当前可执行操作 |
| GET /api/v1/publication | publication:read | authority/revision/原始policy/facts字符串及强ETag |
| PUT /api/v1/publication | publication:write | 校验、提交完整期望状态与审计，然后下发 |
| GET /api/v1/audit?after=0&limit=50 | audit:read | 按ID升序的变更事件与next_after游标 |

PUT体为 policy（完整YAML）、facts（完整JSON）、reason（审计原因）三个字符串。facts须
遵循已有契约（含数值family数组），不能随意序列化内部Go结构。policy/facts各最多4MiB
字节，原因非空白且最多512个UTF-8字节，整个JSON最多12MiB。共享编译器的身份、替换
约束仍权威；重复/null/未知/别名字段及多文档拒绝。仅application/json，不绑定表单/查询。

GET示例ETag为 `"2"`，PUT须单个If-Match同值；弱标签、通配、多值、前导零非法。
缺少前提428，陈旧412（即使语义无变化）。当前版本的无变化写入保留原始输入/修订，不增
审计；并发不能互相覆盖。变更与actor/reason/revision/digest/UTC时间审计同一事务，审计
失败全部回滚。审计游标排除当前ID，limit为1–100/默认50；未知/重复/非法查询拒绝。
空页保留next_after，不代表总量或冻结历史快照。

401凭据无效；403权限拒绝；400输入/配置无效；413超限；415非JSON；429本地限流；503
凭据/RBAC、持久化、发布所有者或下发不可用。Agent凭据无效阻止写入，但保留管理查看。
持久化或提交后下发失败会关闭写入并停止进程，重启恢复原修订。提交后的503有意不承诺
未提交：恢复后先GET再决定是否按新ETag重试。200仅为接受期望状态，不代表Agent/内核
已应用。未增加心跳/状态API，既有RECEIVED/APPLIED边界不变。

## 文档和验证

Swaggo2.0.0-rc6生成[OpenAPI3.1](api/swagger.json)，按嵌入模块元数据核对工具（其
--version仅显示2.0.0）。确定性后处理补同源server及rc6遗漏的严格DTO约束。Go契约测试
检查路由/状态/认证/字段。锁定的Swagger Parser13.1.0/Ajv2020只校验本地文档，禁用
外部/文件/HTTP引用，无CLI遥测。暂未提供文档HTTP路由/渲染器；后续仍使用已选的本地
Swagger UI5.33.2，不改成CDN。

测试包含真实HTTP/TCP→SQLite→gRPC→重启、独立撤销、陈旧/无变化CAS、审计回滚、提交后
失败恢复。静态冒烟检查真实ELF并运行文件模式gRPC及管理Token/Casbin/API/审计/重启/
SIGTERM。这是Controller证据，不是Linux/L7/浏览器产品完成。精确head/CI和剩余项见
AUTONOMOUS_DELIVERY.md。
