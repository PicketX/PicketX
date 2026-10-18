# 可运行 Controller 发布服务（M2D2）

目标是把 M2D1 接入真实 Controller 入口，为后续 Agent Provider、管理端和七层授权提供后端；不声明远程内核或管理功能已经完成。用户已授权自主开发/合并，内联执行，不再停留于例行确认。

运行配置使用 Koanf2.3.8，显式参数>已存在环境变量>严格YAML>默认值；保留-listen，新增-config/-agent-listen。默认仍只有127.0.0.1:8080存活服务，Agent关闭。启用时必须提供authority、policy/facts、私有SQLite路径、凭据文件；拒绝未知/null/重复/多文档和非法值。Agent默认127.0.0.1:9090，轮询1秒、停止5秒；凭据/流128、帧4MiB、每秒接入128/消息32、Hello5秒/发送10秒。TLS证书/私钥成对配置，标准服务端TLS至少1.2；明文可选，不强制mTLS。

持久层使用既定GORM1.31.2、官方SQLite1.6.0、mattn1.14.52。显式版本迁移、不使用AutoMigrate；私有文件、FULL同步、DELETE日志、有限busy等待、单连接；单条完整已接受policy/facts记录、完整性摘要、固定authority、正有符号revision/CAS、溢出拒绝、语义未变不增版。先完整编译/替换及分发预检，持久提交成功后发布；运行模块串行管理凭据/发布，意外分发失败停止并由DB恢复。不声明HA/跨进程缓存一致性。

恢复已接受配置后，当前文件无效仍保留原发布并显示未就绪；首次启动无可恢复配置时必须输入有效。非法policy/facts/凭据重载不覆盖旧状态；有效后恢复。凭据更新独立于策略错误执行，保障坏策略期间仍能撤销Token。不重置期限，不记录Token或SQL载荷。本次不暴露无认证管理或状态详情API。

装配集中管理监听、DB、服务、轮询和关闭，启动失败全部回收。healthz存活，readyz只表示本模块就绪，不泄漏策略；HTTP/gRPC有大小和时间限制，取消后先停发布，再停服务/DB，有限graceful后强制结束，记录实际监听地址。

测试覆盖配置覆盖/严格拒绝、迁移/重启/CAS/损坏/authority/溢出、坏重载保留LKG、坏策略期间撤销凭据、后注册节点接收非空目标、监听回收与真实TCP流/报告、就绪和TLS配置。现有严格facts解码移入共享configuration，保留CLI回归。执行全量Go/race/vet/build、Web和协议生成；CGO静态Controller用netgo/osusergo/sqlite_omit_load_extension，检查ELF/ldd后实际启动SQLite和Agent监听、探测readyz、查数据库revision、SIGTERM停止，CI同测。PostgreSQL/Agent内核本次不作验收声明。
