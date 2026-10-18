# 浏览器控制台设计与计划

基线 Core PR14/main0ccfa8f、Agent PR16/main95a5f7f，main CI 均通过且无开放 PR。完整逐项台账见英文同名文件。

本段实现独立管理密钥登录、SCS 服务端会话、CSRF 和真实配置/审计页面。密钥不写浏览器存储或会话；每次请求以服务端保存的身份/凭据摘要重新解析可信角色，删除/轮换下一请求失效，复用身份不能恢复旧会话。会话绝对8小时、空闲30分钟，登录更新标识、退出销毁、重启失效。

显式启用且要求数据库权威和准确外部 Origin；除回环开发外必须 HTTPS。检查 Host、写操作 Origin，采用 Echo 原生 CSRF，不信任转发身份/TLS头。Cookie HttpOnly/SameSiteStrict，HTTPS 下 Secure。旧 API 仅 Bearer、新 API 仅会话，登录限流。静态资源显式配置、禁止越界或 API 回退首页。

浅色运维工作台保留 React/shadcn/Tailwind/i18next，中英双语。真实配置修订、身份与审计；保存不等于 Agent 应用。轮询/错误/412冲突不覆盖草稿、不自动重试覆盖他人修改；只读前后端都限制。明确加载/失败/空/过时与最后读取时间，无样例数据。

顺序：身份测试 RED→GREEN；会话/CSRF/配置测试 RED→GREEN；真实 Chromium→Go→SQLite 测试 RED→GREEN 后实现页面；完整 Go/race/vet/build/static/OpenAPI、Web 锁文件验证、独立全分支评审、准确 head CI 与安全合并并复核 main。

尚不包含人员密码/OIDC、按资源委派、结构化 CRUD/审批、七层授权及入口集成，不宣称最终产品完成，不暂停后续开发。
