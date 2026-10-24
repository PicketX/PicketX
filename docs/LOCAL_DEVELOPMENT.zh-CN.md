# Controller 本地开发

[English](LOCAL_DEVELOPMENT.md) | 简体中文

Go 后端与 Web 独立运行。Controller 正常启动自行初始化 SQLite 和首个管理员，无准备脚本或开发 CLI，不要求 PicketXD/Nginx。

## 启动

需要 `go.mod` 要求的 Go、CGO/本机 C 编译器、Node24+ 和 npm。从仓库根目录启动后端：

```sh
go run ./cmd/picketx-controller
```

另一个终端：

```sh
cd web
npm ci
npm run dev
```

访问 **http://127.0.0.1:5173/console/**，用户名 **admin**，密码取自生成的私有 `var/controller/initial-admin-password`。日志只提示文件位置，不输出密码。可在账户管理修改初始密码；明文文件只保留最初供应副本，改密后可以删除。

无配置文件时启动完整单实例 Controller：SQLite、数据库账户、HTTP8080、Agent gRPC9090 和本地决策9091。Agent/调用方凭据初始为空，不授权任何人，需要直接测试时显式配置。无 Agent 不阻止就绪，就绪不证明执行。状态保存在已忽略的 `var/controller/`，不要提交或分享。

已有 `-config` / `PICKETX_CONFIG` 部署保留原配置语义，包括文件权威。希望使用无配置默认值时，清除继承的 `PICKETX_CONFIG`。显式参数覆盖已出现环境变量，再覆盖所选配置/默认值。错误或不可读的配置会失败，不回退成新安装。

`browser.origin` / `PICKETX_BROWSER_ORIGIN` 可选。设置时严格校验准确 Host/Origin；未设置或为空时，写请求与实际 Host 和 HTTP/TLS scheme 同源，保留 CSRF/会话保护。不信任转发头；HTTPS 终止入口应配置公开 HTTPS 地址，OIDC 必须显式配置。Cookie Secure 根据显式 HTTPS 或真实 TLS 决定。

## 首次启动与重启

创建新的默认数据库前，在0700私有父目录写入0600初始密码/恢复文件及独立凭据，不覆盖已有文件。数据库和账户审计复用现有迁移与账户 API。`initial-recovery-key` 保留现有恢复契约要求的独立管理密钥，管理员改密后仍须私有保存。

仅数据库身份目录从未提交账户操作（revision0）时创建 `admin`。重启保留密码、账户修改及禁用状态；密码文件缺失不触发账户重置。浏览器会话在重启后失效。中断初始化可使用待完成密码文件继续；若账户创建前丢失该文件，或已有数据库丢失必需凭据，应恢复私有输入，不自动重新生成。准备失败可能留下私有文件供检查，不自动重置目录。

## IDE 与 Vite

在 IDE 中直接运行 Go 包 `./cmd/picketx-controller`，工作目录为仓库根目录，按需设置 `CGO_ENABLED=1`。无需启动参数或初始化器，Web 单独运行现有 npm 脚本；不要求提交 IDE 专用配置文件。

Vite 保留普通 `/console/api` 到8080的代理及公开 Host，5173占用就失败。默认无需设置浏览器 Origin；显式绑定地址时才须准确匹配浏览器主机/端口。修改后端端口时同步修改代理。从 IDE 启动另一后端前先停止旧进程。

Web 由 Vite 热更新，Go 由开发者/IDE 重启。后端 HTTP 读写界限仍为10秒，Web请求15秒，长断点可能超时。中断写操作后先恢复执行、检查当前状态/审计，再重试；错误不证明回滚。

## 构建后运行与部署

停止旧后端后：

```sh
cd web
npm run build
cd ..
go build -o var/picketx-controller ./cmd/picketx-controller
PICKETX_BROWSER_ASSETS=web/dist var/picketx-controller
```

访问 **http://127.0.0.1:8080/console/**。未显式设置浏览器 Origin 时，按请求实际 Host 和 HTTP/TLS scheme 校验同源。复用状态及凭据，Vite 可以停止。同一正常初始化逻辑也适用于部署；生产也可使用显式供应的运行配置及现有 HTTPS 入口。本机构建不等于 Linux 完全静态发布验收。

## 排查

使用 `curl -i http://127.0.0.1:8080/readyz`、已有 Controller/Vite 日志及浏览器 Network。连接失败检查进程/端口；404可能是不同显式配置；403检查 Host/Origin；503检查凭据/就绪状态。不新增诊断工具。

正常启动不清空数据。需主动替换一次性环境时，停止后端并将整个私有状态移到私有备份位置，不用于静默切换已部署数据库权威。Go/Web 检查沿用根 README；macOS 的既有 Unix socket 测试可能需 `TMPDIR=/private/tmp go test ./...` 避开过长/带符号链接的临时路径。
