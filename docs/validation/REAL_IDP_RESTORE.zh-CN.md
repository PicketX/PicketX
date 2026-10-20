# 真实 IdP 下的 Controller 停服恢复

现有安装版真实Keycloak HTTPS流程保留外部IdP，干净停止Controller后，
完整备份并恢复到不同路径。原真实提供方验证及签名协议夹具systemd恢复
门禁均保留；不宣称IdP数据库恢复、在线备份、主机重启、Agent恢复、HA、
任意提供方或生产拓扑兼容。

私有目录0700、文件0600；备份和恢复清单及字节哈希一致后，仅修改
controller.yaml和identity/provider.json内两层路径。停用原路径，禁止
重新初始化、换authority/key或只恢复数据库。异常仅输出固定诊断，不泄露
私有文档差异、路径或原始错误。

实际静态Controller以空PATH和正常CA验证重新启动。必须观察到旧Cookie401、
持久publication/accounts/audit及authority/key/bindings不变，只有校验为
当前读取时间的accounts.observedAt不参与持久值比较；实际PKCE/nonce及
同一人员/权限的新会话；新发布revision3和人员审计；权限降级、绑定撤销，
重新绑定后已观察到撤销的旧Cookie仍无效。签名夹具不替代此真实证明。

保留四个归档名，在controller-install-evidence新增
real-idp-restore-report.json及1440截图。严格公开收据只包含准确源码、
提供方、公开人员ID/版本、私有清单数量/摘要和已观察布尔值，不含私有路径、
配置、subject、code、token或密钥。实际进程、提供方状态和NSS信任清理后
才允许输出成功。

新增30项回归先红后绿，完整62项Node通过。本地Go仅有既有Unix套接字EPERM
导致TestAuthorizationRuntimeTCPUnixAndShutdown失败，不算原生成功。
独立评审、精确提交完整CI/原始日志/四份真实归档/静态二进制/源码/许可证/
截图、带预期头合并和主线复核均须完成；最终PR不可变收据覆盖本文待验状态。
整体产品尚未宣称完成。
