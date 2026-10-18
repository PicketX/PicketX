# 运行 Controller 发布服务

[English](CONTROLLER_RUNTIME.md) | 简体中文

M2D2 把类型化Agent流接入真实picketx-controller进程，提供严格运行配置、SQLite持久发布及凭据重载。它是可运行发布端；PicketXD Controller Provider、管理端和基础七层授权仍待后续接入。

在仓库根目录按 [英文说明](CONTROLLER_RUNTIME.md#start-a-controlled-local-example) 生成私有随机Token文件及仅含摘要的凭据文件，然后运行 `go run ./cmd/picketx-controller -config examples/controller/controller.yaml`。不要提交、输出或记录Token；var已忽略。示例policy含历史绝对期限，发布不会重置；facts是诊断示例，不是Agent现场证明。

配置优先级为显式变更参数>已存在PICKETX_*环境变量>严格单文档YAML>默认。-config优先，否则PICKETX_CONFIG指定文件；-listen和-agent-listen分别覆盖HTTP/Agent监听。未知/null/重复/别名/多文档、类型错误和非法值均拒绝。相对路径以工作目录为准。默认无配置只启HTTP127.0.0.1:8080，Agent关闭；例启用Agent127.0.0.1:9090。所有YAML字段、精确默认值及环境名映射见英文对照表；poll/shutdown默认1s/5s，凭据/流128，帧4MiB，接入/消息128/32，Hello/发送5s/10s。

healthz仅存活；readyz为配置模块就绪，不泄漏策略，不证明远程已执行，也不要求Agent在线。本次没有无认证管理API。TLS证书和私钥必须成对，启动加载，至少TLS1.2；客户端正常验证根和名称，不强制mTLS。明文没有保密性和服务端认证，只适用于有意控制/隧道保护的链路。运行配置/TLS变更需重启，policy/facts/credentials轮询重载。

凭据文件为完整credentials数组，私有0600，使用同目录临时文件后原子重命名。明确空数组可撤销全部，缺失/null/未知/重复字段拒绝；有效轮换/删除/重绑定终止旧流。坏凭据文件保留原有效集并使readiness失败，不能宣称无效编辑已撤销。凭据更新先独立执行，即使策略无效仍可撤销。

持久层使用既定GORM1.31.2、官方SQLite1.6.0/mattn1.14.52，显式版本1迁移，不用AutoMigrate。私有0700目录/0600数据库，单连接、DELETE日志、FULL同步、5秒busy界限；保存完整policy/facts、固定authority、摘要和正有符号64位revision。未来迁移、错误authority、损坏记录拒绝启动。先完整编译/Replace及分发预检，后持久提交再分发；语义未变不增版，旧身份改绑限制保留。revision CAS拒绝并发旧写者，但不提供HA缓存一致性；PostgreSQL尚未接入。

首次启动无旧记录时输入必须有效；已有记录后，坏当前policy/facts保留旧完整发布，readyz返回503，修正后恢复。绝对期限不重置，Agent断连期间仍须独立执行。持久化或提交后分发意外失败停止进程，从DB恢复；不能伪造应用成功。监听/失败回收集中管理，先取消轮询，再关闭服务/DB，HTTP/gRPC有限graceful后强制关闭；输入编译和SQLite busy界限独立于网络关闭预算。SIGTERM保留持久记录。

执行 `scripts/check-static-controller.sh`，构建netgo/osusergo/sqlite_omit_load_extension的CGO完整静态Linux二进制，检查无ELF解释器，实际启动SQLite及HTTP/Agent监听；真实gRPC客户端验证节点快照并发送RECEIVED（不伪造APPLIED），检查迁移/revision、healthz/readyz及SIGTERM退出。既有静态CLI绝对期限/撤回检查保留，CI同测；不据此声明Agent内核或管理/L7已完成。
