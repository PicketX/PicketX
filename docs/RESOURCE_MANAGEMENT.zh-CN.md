# 资源接入（alpha）

[English](RESOURCE_MANAGEMENT.md) | 简体中文

[浏览器工作台](BROWSER_CONSOLE.zh-CN.md)的资源页读取真实数据库发布，创建
Resource/Endpoint 和明确主机 Binding。创建资源**不授予访问**；Binding 是期望
保护部署，不证明 Agent 已应用。[Grant 签发/撤销](GRANT_MANAGEMENT.zh-CN.md)
独立操作；[HTTPRoute](HTTP_AUTHORIZATION.zh-CN.md)仍使用高级配置编辑器。
本段不引入第二权威或数据表。

## 操作流程

1. 先在期望配置中录入可信 Agent 能力和主机执行事实。此页不注册 Agent，也不从
   心跳推断事实。
2. 打开资源页，显式使用最新资源修订，选择新建资源。输入唯一名称、1–64 个具名
   TCP/UDP 单端口 Endpoint 和审计原因；端口为1–65535。可选地址限制支持双栈
   IP/CIDR，以换行或逗号分隔，每入口最多64项。留空表示省略，使用 Binding
   目的范围，不表示全部拒绝，也不凭空推断全局地址范围。
3. 创建后新建 Binding，选择固定 UID 的资源、1–64 个明确具名入口和既有
   `(agent_name, context_id)`。提交前核对 profile、类别、阶段、目的范围和处理
   方式；仅支持 host-local/original-destination。完整编译仍检查 Agent 能力、
   物理重叠和引用。
4. 打开授权页，显式使用最新 Grant 修订。新资源及入口可用于独立绝对期限授权，
   无须手写 YAML 即可签发和撤销，审计记录包含操作原因。

每次只有一份资源或 Binding 草稿。切换表单类型或清空时先确认；切换工作台页签
保留草稿。刷新仅改变观察清单，不改变草稿 CAS 基线或选择目录；显式更新基线
保留输入并更新选项，重试前应核对。412、传输失败或5xx写入失败会阻止重提，
直到显式更新基线；失败不证明回滚，不自动重试。离开/退出提醒未保存草稿。
Viewer 可读、后端拒绝写入；不使用 localStorage/sessionStorage 保存数据。
高级编辑器、授权页各自保留修订基线，跨页变更后需显式重载。

清单展示完整存储的 Binding 选择条件，包括导入的动态全部/标签范围，不伪造
解析结果。每页每类最多50个资源、Binding 和配置上下文；加载、失败、空与过时
状态明确，时间戳仅表示读取时间，不是执行新鲜度。

## API 与不变量

Bearer `/api/v1` 与会话/Origin/CSRF `/console/api/v1` 共用当前
`publication:read`/`publication:write` 检查，见 [OpenAPI](api/swagger.json)。

| 方法/路径 | 契约 |
| --- | --- |
| GET `/api/v1/inventory` | 完整有界资源/入口、Binding、上下文数组，以及权威/修订/读取时间和强 ETag；拒绝查询参数。 |
| POST `/api/v1/resources` | name、endpoints[{name,protocol,port,addresses?}]、reason；201返回服务端生成的 uid、revision 和 ETag。 |
| POST `/api/v1/bindings` | name、resource_name/resource_uid、endpoint_names、agent_name/context_id、reason；201返回服务端生成的 uid、revision 和 ETag。 |

写入必须带当前强 If-Match（缺失428、陈旧412），严格 JSON16KiB，拒绝重复、
未知、null字段，原因非空且最多512字节，新名称最多128字节，既有引用名称最多
1024。地址数组可省略，但不能null/空；具名入口选择非空且唯一。响应no-store。
保留策略4MiB/4096对象边界，清单受既有输入上限约束，不静默截断。
这是创建/读取，**并非完整 CRUD**。

保留既有策略字节和事实，追加紧凑 JSON（有效 YAML）；首个资源替换唯一的
显式空 Configuration。根据当前事实查找精确上下文并原样复制契约，客户端不能
提交或扩大执行设置。既有编译器要求每个 Agent 的 profile 唯一。完整 Replace
执行不可变身份校验、修订 CAS、事务审计与分发；审计失败整体回滚，SQLite重开
恢复相同源码、投影和审计。数据库权威不受文件轮询回写，文件模式不获得管理写入。

## 证据与剩余项

API测试覆盖双栈部署/不自动授权、原文与事实保留、严格校验、上下文冲突、RBAC/
CSRF、CAS、审计回滚、持久重开。浏览器验收覆盖真实资源→Binding→Grant/撤销、
冲突草稿、失败读取、Viewer拒绝和375px布局，并保留真实Nginx/应用授权套件。
精确通过/失败、源码树、评审及CI见[检查点](AUTONOMOUS_DELIVERY.md)和PR18。

编辑/删除、上下文注册、结构化HTTPRoute、人员身份/申请审批、真实期望/应用状态
及最终Linux联合流程仍待完成。本段不声明新增内核执行证据或产品整体完成。
