# 结构化来源 Grant 管理（alpha）

[English](GRANT_MANAGEMENT.md) | 简体中文

[管理工作台](BROWSER_CONSOLE.zh-CN.md)的授权页使用现有单一数据库权威，
提供管理员操作，不提供自助审批或已验证人员身份。请先在期望配置页发布
Resource/Endpoint/Binding 和可信 HTTPRoute。资源/绑定 CRUD、审批和新鲜节点
应用状态仍是后续工作，应用自身凭据仍然必需。

1. 配置资源后打开授权页，显式“使用最新 Grant 修订”。输入唯一名称、实际来源
   IP/CIDR，选择固定 UID 的资源与具名 Endpoint、期限、原因。CIDR 向整个匹配
   网段共享权限，不等同于人员身份。
2. 签发时服务端生成独立随机 UID，并一次固定绝对 UTC 到期时间。表单提供
   15分钟/1小时/8小时/24小时，API 限60..86400秒。不修改、续期或恢复已有 Grant，
   同名签发被拒绝。
3. 列表状态以“最后成功读取时刻”计算，并展示绝对期限与 UID。期望有效不等于
   实际可访问：deny 优先，各 Grant 取并集，此页不能确认节点执行。
4. 完成或清空签发草稿后，选择撤销，核对来源和身份，填写原因并确认。仅撤销
   这个 Grant，其他 Grant 仍可能允许访问；NetworkDeny 和其他对象保持不变。

读取/翻页不会悄悄推进已有操作的 CAS 基线。412 保留输入和原因，写入阻止至
显式使用最新修订；更新基线仍保留输入，重试前必须核对资源 UID 和入口。
传输/5xx 失败仍可能已提交，应先检查最新状态，不自动重试写入。切换页签保留
草稿，离开/退出登录提醒。YAML 编辑器保留独立修订基线，结构化修改后需显式
重载。输入不保存到 localStorage/sessionStorage。viewer 只读，后端每次请求均
校验当前可信角色和凭据撤销。

## API

下列 bearer 路由使用独立管理凭据和当前 publication:read/write RBAC。
同名 `/console/api/v1` 路由使用既有 SCS 会话、准确 Origin 和原生 CSRF，
不会暴露于本地授权监听器。[生成契约](api/swagger.json)。

| 方法/路径 | 契约 |
| --- | --- |
| GET `/api/v1/grants?limit=50&after=UID` | UID 升序的期望 Grant、已有资源/入口目录、权威/修订、服务端读取时间、排他游标和 has_more。默认50，最大100。后续页必须带首页强 If-Match，陈旧412、缺失428；无游标刷新回首页。 |
| POST `/api/v1/grants` | 严格 JSON：name/source/resource_name/resource_uid/endpoint_names/duration_seconds/reason，必需强 If-Match。201 返回 uid/revision/expires_at 和新 ETag。 |
| POST `/api/v1/grants/{uid}/revoke` | 严格 JSON reason，先校验强 If-Match 再判断幂等。200 返回 uid/revision 和 ETag；不存在或非 Grant UID 为404。当前状态重复撤销不产生新修订/审计。 |

拒绝未知、重复和 null 字段，写入限16KiB。原因非空且最多512字节，新名称最多
128字节，每次最多256个具名 Endpoint。各路由错误400/401/403/404/412/413/415/
428/429/503 见生成契约，响应均 no-store。目录受既有4MiB/4096对象输入限制，
页面最多50条 Grant，续页不能混合修订。导入的永久、未来、过期、已撤销、多作用域、
标签/全部入口动态选择的 Grant 均可读取和撤销，动态范围不会伪装成固定当前入口。

## 权威与证据边界

服务端转换已接受发布，原样保留 facts，经共享完整校验/不可变 UID 检查和同一
修订事务/审计/分发所有者接受；没有第二份 Grant 表或后台文件回写器。保留已接受
UTF-8策略，签发仅追加一个紧凑 JSON 文档（合法多文档 YAML），撤销按解析器坐标
局部编辑目标布尔标志，再严格重解码并核对完整结果仅有预期语义变化。这避免大注释
等导入策略因缩进/转义扩张而无法撤销。新状态仍须满足既有4MiB/对象上限，新增对象
或标志仍需余量；其他对象语义、标签、选择器及独立期限保留。文件模式不会新增管理
写入，不修改业务数据库版本或 Agent/协议/内核。

测试包含真实 SQLite 审计失败回滚、陈旧/幂等 CAS、当前 RBAC、严格 DTO/绑定修订
分页；真实 TCP 管理 API→gRPC 快照及 IPv6 独立撤销/期限持久恢复；真实 Chromium
表单→SCS/CAS→本地决策→Nginx→测试 HTTP 应用，覆盖重叠 Grant/撤销、冲突草稿、
失败、viewer/移动端和审计。准确源码/head/tree、原始 CI 与独立评审验收见
[检查点](AUTONOMOUS_DELIVERY.md)及 PR17，不等同于新增 Agent 内核或完整产品验收。
