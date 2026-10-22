# 管理浏览器控制台

本文说明完整期望策略/事实文档、事务审计和工作台部署。[授权](GRANT_MANAGEMENT.zh-CN.md)与[资源](RESOURCE_MANAGEMENT.zh-CN.md)页面提供结构化管理员流程；[HTTP授权](HTTP_AUTHORIZATION.zh-CN.md)是独立限权入口集成。[本地人员登录](HUMAN_LOGIN.zh-CN.md)保留文件配置；[账户管理](MANAGED_ACCOUNTS.zh-CN.md)新增显式数据库人员权威及旧会话永久失效。OIDC 仍延后；申请与独立审批另段已交付。保存不等于节点应用，节点观测是时间点报告而非独立内核确认。

## 运行

按照[管理 API](MANAGEMENT_API.zh-CN.md)配置数据库权威及两个独立私有凭据文件。在 web 执行 npm ci、npm run typecheck、npm run lint、npm run build，运行 YAML 加入：

```yaml
browser:
  enabled: true
  origin: https://control.example.com
  assets: web/dist
```

路径相对 Controller 工作目录。既有 HTTPS 入口必须保留外部 Host；限制 Controller 直连网络访问。origin 使用浏览器 location.origin 的准确值，无尾斜杠、路径、userinfo、查询或片段，不写浏览器省略的默认端口。Controller 不负责 TLS，也不信任转发身份/TLS头。打开 /console/，用独立管理密钥登录；Agent Token 不能登录。密钥不写 localStorage/sessionStorage/Cookie/SCS 值。

仅回环开发可用 http://127.0.0.1:8080 或 http://[::1]:8080。Vite 开发可省略 assets，origin 设置 http://127.0.0.1:5173，npm run dev，/console/api 代理至127.0.0.1:8080并保留外部Host；禁止远端明文开发 Origin。未显式启用或非数据库权威时拒绝浏览器配置，不会回到文件轮询覆盖管理状态。

## 安全与生命周期

SCS2.9.0 单进程内存存储及原生过期清理；重启失效，不支持多副本共享。picketx_session Cookie 为 HttpOnly/SameSiteStrict、路径/console/，HTTPS 时 Secure，浏览器会话持久方式。服务端绝对8小时、空闲30分钟；读取只延长空闲，不延长绝对期限。登录换号并重新计时，退出销毁。仅对已存在认证会话，按准确Token使用引用计数锁串行完整读取/处理/保存；陌生Cookie不加锁，不同会话不碰撞。等待可取消，取得锁后重新读取，无持有/等待者时删除锁，防止在途空闲续期复活退出会话。仍需普通连接/请求体限制，不宣称通用抗拒绝服务保证。

服务端只存身份和凭据摘要，每个请求重新读可信私有文件、解析当前角色。删除/轮换密钥在下一请求撤销旧会话，复用ID但使用新凭据也不恢复；不追溯撤回已授权在途业务操作。角色下一请求生效，Casbin 策略仍按独立周期重载及失败关闭就绪语义。退出仅当前会话，要永久撤销全部旧会话请轮换凭据。[人员登录](HUMAN_LOGIN.zh-CN.md)增加同一新鲜度边界的密码入口，尚无OIDC及按资源委派。

所有控制台请求校验准确 Host，写操作还校验准确 Origin；转发头不能替代。Echo 原生 CSRF 覆盖登录/退出/写入：现代浏览器校验 Sec-Fetch-Site，旧客户端保留 HttpOnly CSRF Cookie 并在 X-CSRF-Token 回传 /console/api/v1/session 的 csrf。现代 bootstrap 值是 Echo 元数据标记，不是 Cookie Token；独立 Origin 校验仍必须通过。无 CORS。登录按实际本地对端限流，与 Agent 限制独立。

旧/api/v1仅Bearer，新/console/api/v1仅Cookie且拒绝任何Authorization头。响应no-store、CSP/no-referrer/nosniff。显式静态资源使用os.Root及扩展名白名单，禁止目录列表、符号链接越界或API回退首页，构建输出不提交。

## 操作与过时状态

读取并编辑完整文档，填写原因，用编辑基线强ETag保存。共享校验、事务审计记录操作者/原因/修订/摘要；viewer只读，前后端均限制。错误不回显密钥或原始文档；复杂校验目前只给策略/事实通用错误，尚无逐字段编译诊断。

刷新更新观测状态/权限，不推进编辑器CAS基线、不覆盖草稿。412保留策略、事实与原因，必须显式重新载入最新配置（有草稿时确认放弃）才能换基线；不自动重试或覆盖。切换页保留草稿，离开脏页触发浏览器未保存保护。退出/撤销清理本地配置和身份。读取失败保留上次数据并标记过时，超过一分钟提示刷新；这是手动刷新，不是实时监测。审计为升序排他游标，每页50条，不是执行/请求决策日志。

## 验证边界

npm run test:e2e 需要Go、构建dist和Playwright Chromium，启动真实Go/SQLite及临时私有凭据，驱动实际浏览器验证登录/CSRF写入、持久读回/审计、并发修改冲突与草稿、注入读取失败（仅负面UI测试）、viewer后端拒绝/手机布局、凭据撤销。CI只保存截图，不上传临时凭据/数据库。完整Go/race/vet/build/static/OpenAPI与Web锁文件检查保留；未据此宣称浏览器→七层→入口→应用或新Agent内核验收。
