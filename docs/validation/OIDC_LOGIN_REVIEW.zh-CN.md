# OIDC 登录评审及验证

一次独立全分支只读评审adfa1b1..31b6285，结论With fixes：0Critical、2Important。
评审者独立跑四包race通过并检查固定SCS/Echo/go-oidc，不代表全量native/归档通过。

两项阻塞均由root同一轮回归优先修复，不二次评审、不伪造平台approval：

1. 跨站回调不携带Strict SCS Cookie，原会话会遗留。真实Go回归捕获旧Cookie，
   start后仍200（RED）；修复为在已有同token锁保护的start里先销毁原SCS，
   再输出外跳URL。Strict不变，增加真实跨站浏览器前后旧Cookie重放401。
2. 新管理库的显式空Configuration不能追加Resource。native39356233386/
   Web118411541723及实际SHA核验截图显示invalid_publication，生产拒绝正确。
   夹具改为用Resource替换空标记，保留revision2精确回读和真实人类审计。

后续准确head需完整native/raw/四归档/源码/expected_sha合并/main再验证，最终证据和
即时租约释放写入PR39；这里的pending不是验收结论。

延期Minor为传输故障专项测试广度：1MiB/5秒/拒绝重定向/正常TLS已检查实现，
但尚无各discovery/token/JWKS超限、慢响应、重定向及不可信证书的专门注入回归。
不是已知绕过，也不宣称跑过这些故障；下段真实提供方验收应保留该项。

## 对全部保留判断的裁定与风险

| 范围 | 裁定 | 若判断错误的代价 |
| --- | --- | --- |
| 全量normal/race | 本地仅继承Unix EPERM，root要求真实native全量及原始日志 | 隐藏失败/race |
| 失败的浏览器全场景 | 修复后完整真实Chromium及旧场景通过前禁止合并 | 未执行的撤销声明 |
| 后续head | root先失败后通过修复及准确最终门禁，不冒充评审者批准 | 后续回归 |
| 真实IdP/MFA/同意/故障策略 | 部署提供方验收待做，夹具不是生产身份 | 提供方不兼容/虚假就绪 |
| HTTPS/Secure Cookie | 不外推旧HTTPS证据，新OIDC部署仍待验证 | Cookie/代理错误 |
| 其他引擎/辅助技术/布局 | root检查EN/ZH375/1440，只声称Chromium实跑 | 未发现兼容/无障碍问题 |
| 传输故障 | 实现已检查，直接测试广度延期如上 | 诊断泄露/DoS |
| 安装OIDC/Agent/应用/恢复 | 明确待做，保留嵌套私有路径说明 | 错误恢复/不完整闭环 |
| 外部IdP撤销/logout | 当前只本地撤销和令牌绝对expiry，已说明 | 误判外部会话有效期 |
| 文件原样恢复 | 必须轮换绑定generation；数据库持久版本另有无中间请求实测 | 旧Cookie复活 |
| 归档/notice/remote/合并/main | root独立核对实际字节、静态ELF、清单、源码、日志、expected head | 交付错误/陈旧版本 |
| 依赖法律/全安全/HA/kernel | 不作泛化保证，本轮不改Agent/kernel | 超出证据的保证 |

额外执行裁定：用户明确省略例行设计/计划确认，不省独立评审/门禁（错误则未评审身份边界）；
本地Chromium0MiB损坏下载改用真实native门禁（错误则无UI证据）；注入必须先于路由
注册，测试保持运行时顺序（错误则测到虚构热重载）；空标记替换只修夹具不放宽解码
（错误则隐藏解析问题）。UI技能保留既有配色/组件，新增真实组织登录/帮助/加载/失败、
英中文和清除secret；不换栈、不假数据。整体目标仍活跃，下一段为部署IdP及安装联合/
停止状态恢复，不重做已完成的基础Provider或打包。
