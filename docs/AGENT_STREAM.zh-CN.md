# Controller Agent 同步流基础

[English](AGENT_STREAM.md) | 简体中文

M2D1 提供 Controller 内部服务、公开类型化 protobuf 和 Go 客户端；测试使用真实 protobuf/gRPC 流、TCP 回环及正常证书校验的服务端 TLS。[M2D2](CONTROLLER_RUNTIME.zh-CN.md)已装配Controller入口、持久发布及运行配置；PicketXD Provider仍待接入，不代表已远程驱动内核。运行 `go test -count=1 -race -v ./internal/agentsync ./pkg/agentsnapshot`。远程Provider/LKG与注册管理后续接入。

## 装配契约

在 Controller 内构造 `internal/agentsync.New(authority, Limits{...})` 并通过 `agentv1.RegisterAgentSyncServer` 注册。公开客户端为 `pkg/protocol/agent/v1alpha1`，协议在 `proto/picketx/agent/v1alpha1/sync.proto`。包不打开监听；装配者集中启动/停止、调用 Service.Close、限定 graceful shutdown 等待，超时调用 grpc.Server.Stop。

所有限制必须显式正数：凭据和活动 Agent 作用域各不超过 4096，快照帧不超过 4 MiB；接入尝试（含认证失败）采用服务本地一秒窗口，消息（含 Hello）采用独立每流一秒窗口。Hello 和发送均超时。不与普通 HTTP 共用计数；进程重启会重置，不是集群配额。RPC 接入限制不代替装配者管理认证前连接、HTTP/2 并发及监听暴露。

服务端配置 grpc.MaxRecvMsgSize/MaxSendMsgSize(agentsync.MaxWireBytes)，客户端匹配收发限制。TLS 使用标准 grpc.Creds(credentials.NewTLS(...)) 和客户端 grpc.WithTransportCredentials(credentials.NewTLS(...))，可信根和 ServerName 正常验证，不能跳过。明文客户端显式使用 insecure.NewCredentials()，没有保密性和服务端身份校验，只适用于有意控制或隧道保护的链路。不强制 mTLS/内部 CA。测试验证错误根和错误名称被拒绝。

凭据完整替换记录为 ID、AgentRef、SHA256；明文 Token 应至少由 32 个密码学随机字节编码产生、无空格并安全分发，服务只保留摘要。线缆接受 32–256 字节可打印 ASCII，长度不等于熵。禁止记录明文 Token/metadata。只接受一条 authorization: Bearer <token>。身份和节点范围只由服务端凭据推导。轮换、删除、重绑定终止已有流；同 Agent 新认证流取代旧流。非法或超限替换完整拒绝。这里的撤销只在当前进程生效，集群声明需要权威存储向所有 Controller 传播。

## 发布和报告

Publish(candidate, revision) 接收不可变已编译配置和严格递增正 revision；authority 固定。完整替换重新校验共享规划层的保留身份/改指向限制，并禁止 UID 改种类。校验/大小/revision 失败不改变当前发布。跨重启 revision 和回滚防护需后续权威存储，发布成功也不代表审批或执行成功。

每个注册 Agent 只得到自己的完整目标清单。后注册凭据收到当前发布；未知或删除节点收到明确空清单。消息保留所有权、上下文/地址、能力、Binding 来源、独立 Grant/Deny 身份、未来/过期/撤销贡献和纳秒级绝对期限。空清单表示完整撤回，断连本身不撤回。慢端可合并中间 revision；重连收到最新完整状态。SHA-256 覆盖清空摘要字段后的确定性 protobuf，只用于完整性/关联，不是签名或认证。Agent 仍须校验域语义、本地作用域和 Provider 身份。

Hello 版本 1；v1alpha1 仍是开发中的 alpha。报告必须匹配该流最后成功发送的 revision/hash，含合法 RFC3339Nano 时间及 RECEIVED/APPLIED/FAILED 状态；过时或非法报告终止流，客户端须处理状态替换/重连。Report(agent) 返回独立副本、本地接收时间和绑定报告原流的 Connected；新流建立后旧报告显示断连。Agent 时间不可信，新鲜度不能由其推导。APPLIED 仅为认证 Agent 的声明，不是 Controller 对内核的独立证明。逐目标执行/清理细节和能力协商后续扩展，尚无可用管理 UI。

## 生成与验证

PATH 固定 protoc 31.1、protoc-gen-go v1.36.11、protoc-gen-go-grpc v1.6.1，执行 scripts/generate-agent-protocol.sh；版本不符直接拒绝。CI 校验上游 Linux protoc 压缩包 SHA-256 `96553041f1a91ea0efee963cb16f462f5985b4d65365f3907414c360044d8065`，重新生成并拒绝差异。运行依赖由 go.mod/go.sum 固定，发布二进制不依赖这些生成工具。

测试覆盖节点完整隔离、独立期限、摘要/副本完整性、撤销/轮换/重绑定、流替换和报告竞争、空撤回、原子发布拒绝、共享改指向约束、接入/消息限制、Hello/发送超时、慢端合并与关闭。这些结果不证明远程 Agent/内核已接入或集群容量。
