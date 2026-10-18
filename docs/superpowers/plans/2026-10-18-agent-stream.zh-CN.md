# Agent 同步流基础实施计划

> 使用 executing-plans 按任务内联执行，沿用已授予的继续、推送、合并授权；最后进行一次全分支独立评审。

目标、协议/依赖精确版本、限制与验收以 [英文计划](2026-10-18-agent-stream.md) 和 [中文设计](../specs/2026-10-18-agent-stream-design.zh-CN.md) 为准。

任务 1：先测试双节点隔离、完整目标/上下文/来源、独立期限与纳秒精度、确定性摘要、篡改/可变副本、空撤回与非法输入；确认缺少实现导致失败，再生成固定版本 protobuf 并实现 Build/Verify。完整测试通过后提交协议与投影。

任务 2：先以真实 gRPC/protobuf 流测试凭据拒绝/轮换/撤销/重绑定、Hello/version、完整发布/重连/删除节点、原子拒绝、重复流、旧报告竞争、复制隔离、限制/超时、关闭及正常 TLS 证书校验；再实现 Controller Service。身份由服务端凭据决定，快照只含该节点，报告仅是 Agent 声明。测试慢端发送时撤销、旧流报告覆盖、新增凭据接收现有发布等边界。

上下文映射与自查已在英文计划记录：新建公开 proto、生成代码、agentsnapshot、internal/agentsync、生成脚本及中英集成说明；修改模块锁定、CI 和实际交付状态。复用 configuration.Candidate、networkplan.Inventory、core.Validity，不修改现有仅存活入口、数据库或 Agent 内核代码。

验收执行全量 Go、race/vet/build、静态 CLI、Web npm 锁文件安装/typecheck/lint/build 和生成代码一致性。持久化单调 revision、生产监听、注册管理和 PicketXD Provider 后续实施，本次不声明已可部署 Controller Mode。
