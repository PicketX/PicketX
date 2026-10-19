# 资源接入（alpha）

## 编辑受保护 Endpoint

在受支持的具名主机 Binding 上选择“编辑受保护 Endpoint”，核对固定 Resource UID、
Agent、完整执行设置及当前保护名单。选择1–64个既有名称、填写原因并确认风险：
增加保护可能拒绝服务流量，移除保护可能暴露流量。这不是撤销 Grant，也不迁移应用。
取消不写入。

PUT `/api/v1/bindings/{uid}/endpoints` 及会话/CSRF 对应路由接受
`{endpoint_names:[...],reason,acknowledge_scope_change:true}`。仅支持单固定 Resource、
单明确 Agent、host-local/original-destination 的非空具名选择；全部/标签动态选择与
多目标返回409 `binding_not_editable`，继续使用高级编辑器。清单仅为受支持形状增加
可选 `endpoint_edit` 投影，原始选择条件仍可查看。输入按 Resource 入口顺序规范化。
UID/名称/元数据、Resource/Agent/执行设置、Grant/Deny、facts 和其他源文档不变；
所选 Binding 文档规范化，其注释/格式可能丢失。继承严格 JSON、鉴权、强 CAS、
完整编译及事务审计，200返回UID/修订/ETag，不表示 Agent 应用。陈旧412先于查询，
缺失或错误类型404。

使用最新修订保留名单草稿和原因，展示最新值并重置确认。固定作用域改变、形状不再
支持、目标或已选入口不存在时，须取消后重新打开；异常状态/回执保留草稿并阻止盲目
重试。表单禁止未修改名单提交。原生浏览器验收使用真实 API/SQLite/审计，负响应注入
仅验证恢复行为，不模拟执行。

## 使用前修改网络参数

选择未被引用资源的“编辑网络参数”，只改既有 Endpoint 的协议、端口和可选双栈
目的地址；资源 UID/名称及 Endpoint 名称、类型和标签不变。核对最新目标、填写
原因并确认后保存；取消不写入。此操作不配置/迁移应用，也不确认 Agent 已应用。
所选文档规范化，注释/格式不保证保留；其他对象/源文本及 facts 不变。

PUT `/api/v1/resources/{uid}/network` 及会话/CSRF `/console/api/v1` 对应路由接受
`{endpoints:[{name,protocol,port,addresses?}],reason}`。须包含所有既有入口且不重复，
不可增删/改名。200 返回 UID/修订/ETag，语义无变化保留修订。任何 Grant（含撤销、
过期、零匹配选择器）、Deny、Binding、HTTPRoute 引用返回409；陈旧412先于目标查询，
缺失/错误类型404。继承严格 JSON、鉴权、CAS、完整编译及事务审计。显式重基准保留
草稿、展示最新值并清除确认；同名重建不会替换原 UID。异常状态/回执阻止盲目重试。
引用中的网络编辑、Endpoint 集合变更、Binding 高级作用域编辑、身份/审批及联合验收仍待完成。

[English](RESOURCE_MANAGEMENT.md) | 简体中文

[浏览器工作台](BROWSER_CONSOLE.zh-CN.md)的资源页读取真实数据库发布，创建
Resource/Endpoint 和明确主机 Binding。创建资源**不授予访问**；Binding 是期望
保护部署，不证明 Agent 已应用。[Grant 签发/撤销](GRANT_MANAGEMENT.zh-CN.md)
独立操作；[HTTPRoute](HTTP_AUTHORIZATION.zh-CN.md)已有结构化工作台页面。
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
5. 退役保护部署时，在对应条目选择**移除 Binding**，核对 UID、资源/Agent
   选择和执行设置，填写原因并确认可能导致服务失去保护，然后提交或取消。
   这**不是撤销 Grant**：其他 Binding 和已有 Grant 不变。最后一个 Binding
   被移除会撤下 PicketX 保护，而不是只撤销允许规则；接受修改不证明 Agent 已应用。
6. 清理未使用定义时，选择**移除资源**，核对精确 UID 和入口，填写原因并确认
   仅删除定义。Grant（包括已撤销/过期）、NetworkDeny、Binding、HTTPRoute
   的任何引用都会阻止删除，不级联修改或暗中撤销。请先在对应页面或期望配置中
   人工审查引用；移除定义不会停止应用。

每次只有一份资源、Binding 或移除草稿。切换表单类型或清空时先确认；切换工作台页签
保留草稿。刷新仅改变观察清单，不改变草稿 CAS 基线或选择目录；显式更新基线
保留输入并更新选项，重试前应核对。412、传输失败或5xx写入失败会阻止重提，
直到显式更新基线；失败不证明回滚，不自动重试。离开/退出提醒未保存草稿。
Viewer 可读、后端拒绝写入；不使用 localStorage/sessionStorage 保存数据。
高级编辑器、授权页各自保留修订基线，跨页变更后需显式重载。
移除草稿更新基线后保留原因，但重置风险确认并展示同 UID 的最新详情；目标已
不存在时禁止提交。取消明确丢弃该移除草稿。成功写入后读取失败不会自动重提。

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
| POST `/api/v1/bindings/{uid}/remove` | reason、acknowledge_unprotection:true；200返回 uid/revision 和 ETag。不存在或非 Binding UID 返回404；先检查修订，陈旧412。 |
| POST `/api/v1/resources/{uid}/remove` | reason、acknowledge_removal:true；200返回 uid/revision 和 ETag。不存在/类型不符404；存在任何引用409 resource_in_use；先检查修订，陈旧412。 |

写入必须带当前强 If-Match（缺失428、陈旧412），严格 JSON16KiB，拒绝重复、
未知、null字段，原因非空且最多512字节，新名称最多128字节，既有引用名称最多
1024。地址数组可省略，但不能null/空；具名入口选择非空且唯一。响应no-store。
保留策略4MiB/4096对象边界，清单受既有输入上限约束，不静默截断。
支持创建/读取、移除 Binding 和未引用资源，**并非完整 CRUD**。

已知 alpha 边界：首次发布前（修订0），移除不存在目标目前返回400
invalid_publication而非404，不会产生修改；工作台在此状态没有可选目标。

保留既有策略字节和事实，追加紧凑 JSON（有效 YAML）；首个资源替换唯一的
显式空 Configuration。根据当前事实查找精确上下文并原样复制契约，客户端不能
提交或扩大执行设置。既有编译器要求每个 Agent 的 profile 唯一。完整 Replace
执行不可变身份校验、修订 CAS、事务审计与分发；审计失败整体回滚，SQLite重开
恢复相同源码、投影和审计。数据库权威不受文件轮询回写，文件模式不获得管理写入。
移除使用解析器文档边界，保留存活文档字节；重新严格解码并核对所有对象，确保
只移除目标 Binding 或未引用资源。结构化移除要求 UTF-8；不支持的编码会被拒绝，不改变发布。
行首显式 `...` 终止符及其同行注释随 Binding 移除；其后的空行、注释和指令保留
原始字节，包括唯一 Binding 替换为显式空 Configuration 时的尾部注释。被移除
文档内部的注释随文档移除，语义策略对象和事实仍保留。浏览器仅在收到目标 UID、递增修订及匹配的强
ETag 回执后报告接受；格式异常的成功响应保留草稿并阻止重试，直到显式核对。

## 证据与剩余项

API测试覆盖双栈部署/不自动授权、原文与事实保留、严格校验、上下文冲突、RBAC/
CSRF、CAS、审计回滚、持久重开。浏览器验收覆盖真实资源→Binding→Grant/撤销、
冲突草稿、失败读取、Viewer拒绝和375px布局，并保留真实Nginx/应用授权套件。
精确通过/失败、源码树、评审及CI见[检查点](AUTONOMOUS_DELIVERY.md)和PR18。

引用中的资源/入口编辑、入口删除、Binding高级作用域编辑、上下文注册、人员身份/申请审批及最终Linux
联合流程仍待完成。结构化HTTPRoute创建和真实期望/应用节点观察已交付。
本段不声明新增内核执行证据或产品整体完成。
