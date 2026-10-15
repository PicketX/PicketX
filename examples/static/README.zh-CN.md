# 静态配置检查

> [English (default)](README.md) | 简体中文

这些文件仅用于只读诊断，不是已部署 Agent 或现场网络事实，不会开放防火墙端口。示例故意使用固定绝对日期，到期后检查不会产生有效权限。

在仓库根目录运行：

```bash
go run ./cmd/picketx config check --file examples/static/gitea.yaml --facts examples/static/facts.json --at 2026-10-15T09:00:00Z
```

输出包含 `state: validated`、`enforcementConfirmed: false`、仅针对策略的哈希、名称/UID 映射、诊断及每 Agent 完整规划投影。09:00 UTC 时 IPv4 来源有 HTTPS 权限，独立 IPv6 来源有 SSH 权限；10:00 仅 SSH 有效；12:00 两者失效，但两个入口仍受保护。不传 `--at` 则按当前时刻求值。facts 和投影使用现有演进中的 Go 诊断表示，不是稳定传输协议。

`empty.yaml` 检查显式完整撤回；空/截断文件失败。移除 Binding 表示撤回保护；仅移除全部 Grant 仍保持保护、拒绝新连接。命令本身不执行撤回。

每个 YAML 文件是一次完整发布。后续运行时提供器必须原子发布、失败保留前一有效状态；本命令不实现目录监听或 LKG。人工 facts 不能替代后端现场观测。

详见 [M2A 设计](../../docs/superpowers/specs/2026-10-15-static-configuration-design.zh-CN.md)。资源引用同时固定 name/uid；新 Resource UID 需要显式更新引用，这是权威配置变更，不推断审批。广域 Resource selector、条件、续期、流程对象、status、协议身份或不支持的执行要求均拒绝，不会忽略。
