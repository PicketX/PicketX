# 静态配置实施计划

> 用户已授权持续实施及推送，无需重复阶段确认；使用 executing-plans 本地执行，整批独立审阅。

目标：通过已有 Core/planner 检查完整网络配置；设计以[英文主计划](2026-10-15-static-configuration.md)及[中文设计](../specs/2026-10-15-static-configuration-design.zh-CN.md)为准。

1. **严格解码器**：pkg/configuration/types.go、decode.go 和测试。先测试未知/重复/null/空字段、别名、大小/层数/对象数、顺序与缺省哈希，再实现。YAML 固定 v3.0.5。
2. **完整规划候选**：compile.go、candidate.go 及测试。Facts 独立提供 authority/Agent/context；Compile/Replace、Plan/Input/Hash/Objects。覆盖双栈、UID 固定引用与重建、独立期限/deny/未来边界、重定向、类型复用、整批失败、空撤回、独立副本；复用 Core/planner。
3. **命令集成**：Cobra v1.10.2；picketx config check --file --facts [--at]。真实临时文件测试，JSON 明确 validated/未确认执行；任何失败无成功 JSON。补充 Gitea 示例、双语说明/索引、依赖许可，运行 test/race/vet/build 和静态二进制示例，再审阅推送并检查 CI。

全局边界：Apache/GPL 分离；严格拒绝未知/null；双栈同等、保留绝对期限；无内核、监听、传输、数据库、UI 或进程配置加载器。单文件原子发布，不宣称实现目录事务或 Agent 运行时。无迁移。

审阅重点：限制错误不能扩大授权；同名重建必须 UID 固定；CLI 失败不显示成功；调用者不能修改已接受状态；空/截断文件不能隐式撤回保护。逐项 RED→GREEN，全量检查，整批独立审阅。

执行记录：2026-10-15 基线 Go 测试通过；使用独立任务工作区中的 feature 分支。遵循用户持续交付授权，不重复请求设计/计划/工作区许可；新的 alpha 契约决定明确记录并在 PR 可审阅。目录代际发布/LKG/运行时留在后续 M2。

审阅/实测修复：Unicode 长 s 字段别名可绕过普通小写重复检查，已先复现失败再改为事实结构字段精确拼写校验，标签 map 仍区分大小写。静态二进制验证发现空撤回遗漏已知 Agent 空投影，已补失败回归再修复，未修改规划器 AgentRefs 语义。

本地验收完成：Go 全量/race/vet/build、npm ci/typecheck/lint/build 均通过（npm 只有环境 http-proxy 提示）。静态 Linux amd64 CLI 经 file/readelf 检查无动态解释器/共享库依赖；09:00Z 两项 allow/两个保护入口，10:00Z 一项/两个，12:00Z 零项/两个，显式空撤回仍报告已知 Agent 的零保护。相同构建/示例检查已加入 CI。整批独立审阅仅发现上述一项 Important，已复现并修复；远端 PR/CI 结果记录在 PR 中，避免证据提交自引用。
