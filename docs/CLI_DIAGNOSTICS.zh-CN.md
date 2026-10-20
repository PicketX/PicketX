# Controller CLI诊断

安装包中的picketx status和picketx doctor读取现有Controller管理API。使用具备
publication:read的独立管理密钥（viewer即可）；Agent/入口Token和浏览器Cookie不能代替。

```sh
picketx status --controller https://controller.example.test --key-file /private/management-key
picketx doctor --controller https://controller.example.test --key-file /private/management-key --agent node-a --format json
```

参数只传文件路径，不传密钥值。文件必须绝对路径、普通文件、非符号链接、0600或更严格，
内容32–256个可打印非空格ASCII字符，可有一个末尾LF；controller-init的bootstrap-key可用。
父目录须由可信运维者控制。不增加凭据环境变量发现、DB直读或另一套配置权威。

HTTPS正常校验CA/主机名，可用--ca-file增加PEM CA；HTTP仅接受IPv4/IPv6字面回环地址。
URL不接受用户信息、路径、查询、片段；不跟随重定向、不使用环境代理、不关闭TLS校验。
总--timeout默认5s，须大于0且不超过30s，无自动重试。鉴权/读取/数据无效时固定错误、
无成功报告；不输出policy/facts、原始异常、密钥或私有路径。Bearer只发固定管理GET，
readyz不带密钥。
非法flag及位置参数同样固定报错而不回显；publication响应上限按两个受支持4 MiB
字段的最坏JSON转义计算，解码后仍校验各字段上限，不采用无限制读取。

status读取成功返回0，即使无节点或断连；doctor还要求ready200、至少一个所选凭据范围、
全部为新鲜且匹配期望的APPLIED报告，否则输出诊断并返回1；读取错误也返回1。
--agent精确选择当前凭据范围，不存在即失败；--format默认text，可选json。

reported-applied仅表示当前流新鲜、版本/哈希匹配的Agent声明；applying尚未匹配应用，
failed为新鲜失败报告，stale为过期/旧流，disconnected为断连，unknown为已连接但无报告，
unbound表示无发布目标。空列表不表示部署成功。

先后读取publication/observations/publication，权威或revision变化、期望版本矛盾即失败；
报告剩余新鲜度减去本地读取耗时，不用Agent时间延长，不因刷新而重置。多次API采样
不是原子租约。JSON始终enforcement_confirmed=false；不是独立内核/报文、持续健康或
某条撤销连接的证明。doctor不修复/重载状态；原静态config check和部署命令保持原语义。

真实CI用安装包CLI空PATH检查初始/重启revision、断连诊断、错误用途密钥拒绝与停服失败；
实际Controller/gRPC测试覆盖APPLIED、待应用新revision、断连及旧流重连。归档新增
cli-diagnostics-report.json；准确源码/CI/main和评审限制以交付PR最终证据为准。

