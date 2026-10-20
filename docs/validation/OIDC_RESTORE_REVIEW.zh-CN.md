# 安装版 OIDC 恢复评审

oidc_restore_review 仅一次只读全分支评审72ab5d0..ef55027，共11文件，含设计/计划/
账本及既有service/CI。结论With fixes：0 Critical、1 Important、1 Minor。
评审者独立执行21个Node测试、模块语法、diff及合成泄露复现，检查已提取service
报告/图片。原生CI、原始日志、四份实际归档/静态/源码证据由root验证，不是平台approval。

## root同一轮阻塞修复

Important保留：provider对象及bindings字节断言失败会把完整私有actual/expected
附入公开stack，字面凭据替换无法覆盖未知generation/subject/其他私有字段。这是
证据保密缺陷，不是已知生产授权绕过。先保持行为抽取（21绿），再新增两个合成
不等文档回归，实际因stack含私有marker而RED；最小修复改深比较布尔值＋固定提示。
独立等值通过、不等仍拒绝，两侧subject/principal/generation/issuer/client/secret
均不入stack；合并Node23通过、无跳过。不再评审，不把后续root修复冒充独立批准。

Minor延期：installed-oidc-restored-1440.png实际宽1280，按真实尺寸汇报，不声称
1440 viewport；没有改UI布局。

## 全部保留判断的root裁决与错误代价

| 评审边界 | root裁决 | 错误代价 |
| --- | --- | --- |
| 生产IdP/MFA、HTTPS/Secure Cookie | 仍待实际部署验收，签名夹具不是生产身份源 | 虚假部署就绪 |
| 安装Agent/内核/应用联合闭环 | 无Agent改动，保留单独真实闭环门禁 | 身份误当执行 |
| 未独立重跑全Go/原生systemd | root必验最终完整native，Unix EPERM不算绿 | 隐藏失败/race |
| 四归档/来源/静态/许可证 | root检查实际下载字节、完整清单、源码/ELF | 错版本交付 |
| 通用备份/外部路径/并发变更 | 明确不支持，仅完整停服实例＋受控父目录 | 超出契约的不安全恢复 |
| 既有OIDC验证器全面安全评审 | 只审抽取等价和新增断言，不泛化安全保证 | 无证据担保 |
| 未观察的binding移除重加 | 本场景只证实观察到的撤销，保留generation边界 | 旧Cookie复活假设 |
| 普适视觉/无障碍 | root检查实际恢复/移动/中文图，无UI改动或普适声明 | 未见布局/无障碍缺陷 |
| 最终合并/main/合并后归档 | 必须expected-head及新main原始证据再释放 | 合并陈旧或未验证源码 |
| 历史日志绝对无泄露 | 修复本次可复现路径，不作历史全面保密声明 | 其他遗漏披露 |

其他裁决：用户免例行暂停但不免验证（错则信任边界未审）；最终集成不是启动评审
的循环前置条件（错则提前完成）；现有账号UI默认requester并勾选admin，原生
RED39371306721后改为精确字面并集，不弱化生产验证（错则权限判断错误）。本地
issuer smoke因配置代理失败，不绕过或编造成功（错则假协议证据）；保留既有宿主
工作区和他人修改（错则数据丢失）。

候选39371678322/remote2909449/tree9cd46b全部3job/step通过，原始Go260顶层/660
总测试无fail/skip/race；root核验4实际ZIP哈希/大小、244文件清单/静态/许可证、
10私有恢复文件、4个uid999/caps0/NNP1进程及严格新旧service/入口报告。这只是
评审修复前候选，不是最终head验收。最终不可变提交/PR/CI/源码/归档/main证据及
即时释放写PR40元数据，优先于源码pending；整体继续，实际IdP/HTTPS/安装闭环待做。
