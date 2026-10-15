# M2A：静态网络配置入口

> [English (default)](2026-10-15-static-configuration-design.md) | 简体中文

- 状态：实现、本地验证及整批独立审阅完成；远端验收记录见 PR。
- 范围：可复用严格解码/编译器和可运行配置检查 CLI；Agent 安装、目录监听/LKG、传输、内核应用、七层和管理端后续交付。
- 基线：产品、架构、配置模型；本仓库不引入 GPL Agent 源码。

## 目标与决定

让用户无需构造 Go 结构体即可编写、检查完整网络策略；复用 Core/networkplan。配置有效仅表示期望状态有效，不证明执行成功。单个多文档 YAML 文件构成完整候选，由发布者原子替换；不把防抖当作多文件事务，本批不做目录监听。

使用 `picketx.io/v1alpha1`：Resource、Grant、NetworkDeny、EnforcementBinding。NetworkDeny 仅表示运行时来源限制，不是申请资格/审批策略。拒绝未知版本、类型、字段和用户 status；拒绝显式 null、重复键、非字符串键、锚点/别名、合并及自定义标签。限制 4 MiB、4096 对象、64 层节点。空输入失败；唯一 `Configuration` 对象的 `spec: {empty: true}` 是显式完整撤回，不能与其他对象混合。

metadata 必须包含非空 name 和显式不透明 uid，可有字符串 labels/annotations。同类型名称唯一，候选内 UID 全局唯一；UID 为 1–128 个 ASCII 字母/数字/点/下划线/连字符。资源引用必须同时提供 name/uid；先解析再编译，避免同名资源重建继承未修改的 Grant。Core/networkplan 资源使用 Resource UID，Grant/Deny/Binding 贡献使用各自 UID；内嵌 Endpoint 名为稳定条目键。保留贡献 ID 的物理重定向由 Replace 拒绝。改名保留 UID 并更新引用名；跨代 UID 换类型失败。持久化墓碑和新授权签发是否合法仍由发布者负责，不推断审批。

Resource Endpoint 必须有 name、TCP/UDP 协议和 1–65535 端口，type 缺省 network；可选 addresses 必须为非空 IP/CIDR 并集。Binding 必须有资源、Agent 选择和显式 enforcement profile/class/stage/destination boundary/handling。Agent 和已解析执行上下文仅来自独立运维/运行时事实，不允许策略对象提供；真正执行前仍需适配器现场校验。保持规划器现状，不支持广域 Resource selector。Endpoint 名称/标签、Agent 名称/标签选择复用现有语义；省略 Endpoint 限制表示整个资源，null/空限制失败。

Grant/NetworkDeny 必须有单个 `subject: {type: source, cidr: ...}`、非空具名资源 scopes、validity。validity 为 bounded 加绝对 RFC3339 expiresAt，或 permanent 且无 expiresAt；可选绝对 notBefore、revoked 默认 false。省略 actions 缺省 connect，显式空 actions 失败。未知条件/续期字段直接失败，不静默丢弃；来源不代表申请人或门户身份。

解码器拥有输入。编译完整验证所有对象，包括未匹配、已撤销和已过期对象。Replace 使用前一 Candidate 和 networkplan.Replace，失败不返回任何替换或撤回。暴露不可变 Plan、独立 Input 副本、对象标识及规范对象 JSON 的 SHA-256。对象顺序、显式 network/connect 缺省不影响哈希；不保证其他前缀拼写/selector 列表顺序等语义归一。重放始终保留绝对期限。显式空配置保留可信运行时库存但撤回所有保护，与保留 Binding、仅无 Grant 不同。

`picketx config check --file policy.yaml --facts facts.json [--at RFC3339]` 使用已选 Cobra v1.10.2；JSON 输出到 stdout，诊断到 stderr，错误非零退出。facts 的结构字段要求精确拼写（包括 Unicode），拒绝重复/null，标签 map 保持区分大小写。facts 是严格 JSON，含 authorityID、agents、contexts，后两者使用现有演进中 Go 规划器表示；只作诊断输入，不是稳定注册/运行时设置 API。没有引入环境/文件配置优先级或 Koanf 加载器。输出 `state: validated`、`enforcementConfirmed: false`、哈希、对象身份、指定时刻每 Agent 投影。facts 限 4 MiB，拒绝未知字段/尾随 JSON。无特权操作或网络监听。

本批节点解析器选择稳定维护路径 YAML v3.0.5，而不是 v4 候选版；Cobra 固定已选 v1.10.2，保留依赖许可。策略 DTO 与 Core、HTTP、数据库、运行时事实分离。显式记录新的 alpha 契约，不宣称原先待定协议已经批准或完成。

## 验收

覆盖真实双栈 Gitea HTTPS/SSH、独立期限、deny 优先、未来边界、选择字段存在性、未知/重复字段、错误类型/限制、固定引用、同名重建、UID 唯一/换类型、完整候选失败、重定向拒绝、显式空撤回、输入输出独立、顺序/缺省哈希、绝对期限重放。CLI 用临时文件真实运行解码和规划，错误不能输出成功 JSON。执行 test/race/vet/build、静态 Linux 二进制检查和固定时刻示例，整批独立审阅，再推送并检查 CI。
