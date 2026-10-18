# 自动交付检查点

## PR18 main CI 修复（进行中）

心跳2026-10-18T15:43Z；负责人 `codex/resource-onboarding`；租约延长至16:30Z，
或修复合并且 main 全绿/明确交接时释放。PR18 已按预期 head 合并为
2de0d6a8c9e52ed40cd68b6486b7bd5fccd29bad，源码树9c59b871298cb5a86757357642e7bddab2def5e9；
最终分支 CI39140789757 的 Go167/429 和真实 Web/Nginx 均通过。
main CI39141222771 的 Web117715886102 通过，但 Go117715886276 的真实 Unix
权限断言失败：测试收到原始 Listen 回调后，可能先于运行时 chmod0600 检查权限。
这不是本地 EPERM，也不是验收通过。修复计划与上下文图见
docs/superpowers/plans/2026-10-18-unix-startup-assertion.md。
保留0600断言，改为在真实 HTTP 服务响应后检查，不改生产代码。
仍须完整原生 CI/重复压力测试/独立修复评审与 main 验证。
原租约继续有效，请勿竞争开发；最终 SHA/树/PR/CI 证据追加至 PR18 和修复 PR。

当前心跳2026-10-18T15:28Z：资源接入增量 `codex/resource-onboarding`，租约至16:00Z或合并且main全绿/明确交接。
Core PR17已验收：main92b2d5f5e1f00c1ae32e219fa1cf9bd45deceee5，树12e206a694ec648a91738cdbbce0cde69bb2ba72，
mainCI39137171776通过161顶层/423总测试及真实浏览器/Nginx/Unix/静态产物；旧租约已释放。Agent95a5f7f/CI39118687475不变且通过。
接手时两仓库无开放PR。本段设计/计划为2026-10-18-resource-onboarding，新增真实资源/Endpoint与显式既有上下文Binding创建及React目录。
保留唯一权威、CAS、审计和原始facts；不推断已应用。已有对象变更、节点登记、审批/身份、真实节点状态及组合Linux验收仍待后续。
本地完整Go基线仅Unix socket EPERM失败；不得跳过，必须以原生CI执行验收。租约内不要竞争实现。

PR18代码验收：本地1d272a14d495d970924cbb1c8a2e1d04df718046、远端f040e70aea47a3a0205c726aa3b30ec981c46fa7、
实际CI检出52f11cd70064a24b4c022af3f78dad1b29fe7cf9同树36a5b1161e30dbb32912e1b5fd8abd3407697c3d。
CI39140082943 Go117712526496/Web117712526381全绿：167顶层/429总测试，无fail/skip/race；真实Unix、
vet/build、固定生成/OpenAPI及全静态SQLite/管理Controller/CLI通过。本地完整普通/race-count1实际运行，
仅Unix EPERM失败，不声明本地全通过；Web锁文件安装/typecheck/lint/build本地与原生均通过。

真实Chromium资源/双栈入口→精确主机Binding→Grant签发撤销、412保留草稿、读取失败、Viewer拒绝、
375px创建/只读布局、facts不变和持久审计通过，保留所有真实Nginx/应用来源隔离、并集、deny、撤销、绝对期限与IPv6场景。
截图12140969029下载并校验SHA2561637544b82b004e55fecb0d52c3c3ba9484a6b3c230f6d2f9b23a83498676f7f，
桌面及两类手机表单/Viewer目视检查。UI实现前CI39139004218真实缺少Resources导航RED；API404及Grant条目/
可选地址schema测试RED→GREEN。后续只修测试：精确原因标签避免歧义；并发CAS必须真正改变语义而非幂等发布。

一次独立全分支评审无Critical/Important；独立跑TestInventory和配置/controller/管理/publication/networkplan
相关race、diff检查及Binding手机截图，未独立重跑完整浏览器/远端CI，由主执行者负责原始证据。
延后小项：表单/DTO按字符限制，领域按128 UTF-8字节限制，43个中文字符可能得到通用拒绝提示；后端安全拒绝，
后续补字节级行内提示与边界回归。PR17延后endpoint_names schema约束已补齐。

裁定及代价：资源/Binding单草稿防止暗中更新CAS（批量操作需多切换）；沿用现有样式并应用可访问错误汇总
（无匹配设计模式，布局个性化有限）；编辑删除仍高级配置（生命周期不够易用）；Agent注册/HTTPRoute表单延后
（初始部署仍需高级配置）；人员身份/申请审批延后（无申请人自助）；不宣称真实应用/组合内核完成（整体验收未完）；
不宣称PostgreSQL/多Controller可用（目前限单Controller SQLite）。

评审后仅交付文档收尾。仍须最终准确head完整CI、expected-head合并和main CI复核，执行后的精确SHA/树/运行记录
追加在PR18；仅合并且main全绿释放租约。下一步结构化HTTPRoute和真实期望/应用状态，再身份审批/生命周期与
浏览器→API→L4/L7→应用的Linux联合验收。此局部里程碑不暂停总任务。

最新心跳2026-10-18T12:27Z：PR15修复后代码已验收，远端ecddce298d5b18933680808593191e469fc68d22/树ab3b87c3f0cb440189257d2adab69d60a7e0dcfc与本地7b27b24、实际CI检出4fa673c一致。CI39127108548完整Go/Web成功：原始日志139顶层/368总测试、无fail/skip/race、vet/build/固定生成/OpenAPI与真实静态运行检查；真实Chromium包含只读焦点/选择/键盘滚动/禁止编辑和后端403。截图12135394967摘要0bcded1a0a9fd54a2072f83d2153660812cc14eee1ddcc2bee7fb8ae029097fd下载校验并目视复核。

一次独立全分支评审发现锁可用性和viewer键盘访问两项问题，均经单次修复流程RED→GREEN，无遗留发现。陌生Cookie不加锁、有效会话精确锁不碰撞、等待可取消且回收引用；退出重放保护仍通过。首次只读CI39126597617为RED；39126806452即时读取滚动状态失败，改为等待真实键盘滚动后，39127108548完整GREEN。评审独立运行普通/相关race，未复跑浏览器/远端CI/静态/Web，主执行者负责这些证据。

本次仅元数据收尾仍需最终准确head完整CI，再合并并复核main；精确最终head/树/CI/main结果在PR15中持久追加。合并且main成功即释放租约，不等13:30Z，不重做会话/配置页面。下一段推进可信HTTP请求授权/本地受控决策/真实入口，再补结构化资源、Grant签发撤销、状态和审批页面；浏览器→授权→代理→应用及Linux完整验收尚未完成，不暂停任务。Agent本段未修改，不宣称新增内核验收。

心跳2026-10-18T12:07Z：当前[Core PR15](https://github.com/PicketX/PicketX/pull/15)仍在开发/评审，租约13:30Z，合并且main CI通过释放。远端65096587c3769b3037e1b78486370955d08304f8/树c11e009074d7856a0a37599ff74f3ef9cfc5241e与本地/实际CI源码一致；CI39125526147 Go/Web成功，原始Go日志133/362无失败/跳过/race及静态运行检查。真实Chromium→React→SCS/CSRF→Go→SQLite验证发布/审计、冲突草稿、负面读取故障、viewer拒绝/手机布局、凭据撤销通过，下载截图校验摘要并目视检查。不是本地浏览器成功：本地下载包为空。最初CI39124774237缺登录控件为RED证据。补充静态越界、过期/换号/绝对期限及重复Origin/Bearer隔离回归，本地完整race136/365通过；最终准确head CI、独立全分支评审、合并/main复核仍待完成。尚不具备结构化易用管理/审批或L7，不暂停后续开发。

当前心跳2026-10-18T11:40Z：分支codex/browser-console，租约至13:30Z，合并且main CI通过或显式交接释放。旧PR14已合并且租约释放：Core main0ccfa8ff8522fb95d93b7898f8b0a1ed9dfcc65a，树3f49d6f51cc0cba29ada9f3a65a7d845a849e06d，main CI39121855312通过128/357、无失败/跳过/race。Agent仍为95a5f7f/CI39118687475。本段实现SCS会话/CSRF和真实发布/审计页面，规范与台账见2026-10-18-browser-console计划。采租时两仓库无开放PR。后续仍需结构化管理/审批、L7及浏览器→代理→应用验收，不暂停任务。以下旧管理段状态已被本段覆盖。

2026-10-18：用户已授权自动开发、测试、推送、PR、修复CI及评审/验证后按准确head合并，持续到易用管理端与基础七层授权完成；不重复请求例行设计/执行/推送确认。当前租约为Core PR14管理发布增量，分支codex/management-authority，到2026-10-18T12:30Z，或PR合并且main CI通过即提前释放。先检查实际远端/心跳，不启动竞争实现。旧PR12和Agent PR16租约均已释放，不等待旧时间戳。

已接受Core main0b2c5e919626f281750295cf6cb119ea7241ed99，树094555cc0a493cddbe855b4dac9fd38715da54de，CI39117067190（115/344）通过；Agent main95a5f7f62dceb1c369d08b8e23ffda2f7b978ce4，树a53a743e22254c0511b430e5b9e2ed0664180629，CI39118687475（190/482、112原生/9服务）通过。PR11非空目标后注册与PR12 stale UPDATE CAS回归均已完成，不重做。

详细租约、精确源码、验收和接续规则以 [英文检查点](AUTONOMOUS_DELIVERY.md) 为准。新运行先读 main/开放PR/CI，避免重复或并发覆盖；通过测试与独立评审再按准确head合并，并复核main CI。目标完成后暂停每小时任务。没有执行能力时报告阻塞，不把提醒/模拟或静态页面当成开发完成。

本段提供显式数据库权威、独立API Token主体、官方Casbin/GORM RBAC、完整配置读写、If-Match和事务变更审计，真实HTTP→SQLite→gRPC→重启测试通过。管理模式不轮询policy/facts文件，不回写覆盖API状态；来源绑定入库。200仅表示接受期望状态，不代表Agent应用。[运行说明](MANAGEMENT_API.zh-CN.md)。

代码远端head525497cf051d8f4937df020b53f5e20b52e311c9与本地评审树2576f0fd53aff92f59babf16c6a81296f4473b04一致；PR CI39120860780进行中。完整Go/race（127顶层/356总测试、无失败/跳过/race）、vet/build、Web锁文件检查、真实静态文件/管理模式Controller与CLI冒烟已通过。一次独立全分支评审无Critical/Important/Minor发现；评审独立跑完整普通和相关race测试，未复跑内核/静态产物/Web或远端CI。尚未合并。

裁定：云技能脚本不可直接执行，使用仓库持久台账，代价是手动维护；OpenAPI校验CLI触发未授权遥测后停用，改锁定Swagger Parser13.1.0/Ajv2020本地校验、禁用外部/文件/HTTP引用，不假定遥测许可，代价是另一套lint建议。

随后CI39120860780在setup-node失败：错用不存在的apps/web/.node-version（原始job117656009194），必需测试被跳过，不算验收。新增TestDocumentationCINodeVersionFilesExist先复现同一路径失败，再改为根目录.node-version；这只是已评审生产树之外的工作流/测试修正，未改生产行为，须重跑全部检查。

修正后代码验收通过：远端headf8536e6b447b9da5454dadbb9f235b492dd27e39，树27daea21709eb7682f8d7f0dc9b551e72a4abbb8；实际CI检出96108c071c0c009b4b95b54c9899307df07c376b同树。CI39121161170的Go/Web均通过。原始Go job117656881965确认128顶层/357总测试，无fail/skip/race，vet/build、协议/OpenAPI重生成、本地OpenAPI3.1校验、真实全静态文件SQLite/gRPC及管理Token/Casbin/HTTP CAS/审计/重启/SIGTERM、CLI绝对期限/撤回检查通过。Web job117656882175确认npm ci/typecheck/lint/build通过。未改Agent内核，不能把旧PR16证据当成本段新内核测试。

最终仅验收元数据/文档头仍须完整CI，然后按expected head合并并复核main；精确最终/main CI在核对后追加到PR14，合并且main全绿才释放租约。无未解决评审发现或延后小项。

下一段：SCS浏览器登录/CSRF与可信成员新鲜度、对象/Grant/审批API和真实React页面、基础HTTP可信入口及受控TCP/Unix决策接口，最后浏览器→API→授权→代理→应用与Linux联合验收。保持选定栈、IPv6、单来源、绝对期限和审批/下发/应用分离，不强制mTLS。管理页使用frontend-design/ui-ux-pro-max，体现加载/失败/空/过时状态。不长期只堆库或样例UI，目标完成前不暂停。
