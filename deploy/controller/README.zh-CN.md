# 独立 Controller：私有首次安装

[English](README.md) | 简体中文

本 Linux-amd64 包包含全静态 `bin/picketx-controller`、`bin/picketx` 和构建后的
React `web/`。Go/npm/Python、额外共享库只用于构建，运行不需要。现有可信入口
继续负责公开 TLS 与业务转发。这是单进程 SQLite Controller，不是高可用、PG 或
自动安装器。systemd 文件仍需人工核对；CI另行验证真实非特权服务生命周期和停服恢复，
不等于整机重启验收。

## 构建与检查

在干净、已提交的源码目录，准备 go.mod 指定的 Go、.node-version 指定的 Node、
npm、Ubuntu/Debian 的 musl-tools、file/readelf/coreutils：

```sh
bash scripts/build-controller-bundle.sh /absolute/new/controller-bundle
cd /absolute/new/controller-bundle
sha256sum --check SHA256SUMS
```

输出目录必须不存在；SOURCE.json 记录源码提交/树。SHA256 证明内容一致，不证明
发布者身份：通过可信渠道获取文件并核对 CI 源码证据，不以 root 运行任意 PR 包。
不会自动发布 Release。先移动到最终部署路径，再初始化。
Controller 显式使用 musl-gcc；CLI 使用 CGO_ENABLED=0。保留仓库第三方声明及
实际 Go/npm/musl/编译器许可原文，放在 licenses/，索引版本并纳入 SHA256SUMS；
许可收集不是完整法律审查，也不替代任何上游许可。

## 手工安装非特权服务

仅用于新机器；账户命令按发行版调整。先检查所有目标，不覆盖已有部署或状态。
创建非特权 picketx 用户/组，将已验证包放到 `/opt/picketx`（root 所有，服务可读
不可写）。仅创建父目录 `/var/lib/picketx`，归服务用户所有，权限0700，然后：

```sh
sudo -u picketx /opt/picketx/bin/picketx controller-init \
  --directory /var/lib/picketx/controller \
  --assets /opt/picketx/web --origin https://picketx.example.com
```

父目录与 Web 路径必须已存在且不经符号链接。初始化拒绝一切已有目标；创建随机
权威 ID，不监听、不授予访问。I/O 失败保留私有半成品供检查，不重试覆盖，也不删除
活跃状态。配置为合法 YAML 的 JSON 语法，均用绝对路径；移动须停服后显式改配置，
不能重跑初始化。
origin 必须符合浏览器规范写法：小写 ASCII/punycode 主机、规范 IP/压缩 IPv6，
端口无前导零且省略默认80/443；否则初始化在写文件前拒绝，避免浏览器规范化后无法登录。

检查 controller.yaml：策略来源与人员账户均为 database 权威，无文件策略/facts
轮询；私有 SQLite 在 controller/state/controller.sqlite。默认仅回环 HTTP8080、
Agent9090、决策9081。可用 --listen/--agent-listen/--authz-listen 指定互不冲突的
回环 IPv4/IPv6 和非零端口，不会自动公开监听。远程 Agent 必须另行审阅显式配置，
使用正常服务端 TLS 或现有安全隧道，不强制 mTLS；勿公开明文 Agent/决策接口。

核对程序、用户、配置路径后，再人工复制 deploy/picketx-controller.service 到
系统 unit 目录并启动/启用。服务无 capabilities，除私有状态目录外文件系统只读；
Type=simple 仅表进程启动，须另查 healthz/readyz。SIGTERM 停服。Controller 不需要
nftables/内核权限；整机开机与 Agent 服务联合验收另行完成。

服务验收在一次性 Linux 主机上沿用发行模板的隔离与重启设置，仅替换独立测试用户和
路径。核对真实可执行文件、非 root UID、零 capabilities、NoNewPrivileges、IPv4 HTTP
与 IPv6 Agent 监听，实测 SIGKILL 自动重启及 SIGTERM 正常停服；不会替你启用/重启
整机，不冒充生产反向代理或 Agent 服务部署验收。

## 首次登录与凭据

私下读取 bootstrap-key，粘贴到精确 origin 的 `/console/` 的 Access key 登录。
命令只打印路径，不输出密钥。不要把密钥放进 URL、历史、截图、问题或 CI 日志。
到 Accounts 创建有 Admin 角色的个人账户，再用人员用户名/密码登录。账户、角色、
停用、重置密码、审计均通过真实页面管理。重启使内存浏览器会话失效，但保留账户、
发布和审计。

| 私有明文文件 | 仅 SHA256 注册文件 | 授权边界 |
| --- | --- | --- |
| bootstrap-key | principals.json | principal:bootstrap，role:admin 恢复/API |
| agent-token | agents.json | initial-agent，仅 node-a Agent |
| ingress-token | callers.json | initial-ingress，仅 route-web 路由 UID |

三把随机256位密钥彼此独立；目录0700，文件0600。--agent-ref/--route-id 可指定初始
作用域，支持1–128字母/数字/点/下划线/连字符。凭据不会自动创建 Agent facts、
资源、绑定、路由、Grant；在控制台发布真实主机作用域，确保 Agent/入口 IDs 一致。
审批/发布不等于内核应用确认；来源 Grant 不替代应用账户、密码、SSH key 或仓库权限。

bootstrap-key 是独立于数据库人员账户的管理员恢复凭据，按离线恢复凭据保护。
撤销/轮换需原子替换0600哈希注册文件，保留并实测另一恢复入口，验证旧 key 失效。
Agent/入口不能复用恢复 key；入口 key 只绑定指定路由 UID。使用 examples/http 的
入口约定，由可信代理覆盖外来原始请求头。代理头不会自动证明网络来源；来源与
facts 必须匹配真实执行点。Grant 并集、deny 优先、独立绝对期限与 Agent 既有撤销
语义不变；graceful SSH 只阻止新连接，不承诺断开已允许的现有连接。

## 备份、重启、升级

停服后复制整个私有目录，或使用已验证的 SQLite 在线备份程序。备份也按密钥保护，
将配置、注册文件、密钥、数据库及权威一起保存；WAL 活跃时不能只拷主数据库文件。
恢复所有权与0700/0600，保留原 authority，不对恢复状态运行 init。保留旧包和升级前
停服备份；不承诺迁移回滚，不能直接用旧程序打开已升级库。本段不新增数据库迁移。

停服迁移时，将**整个**私有实例复制到新的0700备份并逐文件校验；不得覆盖原目录/
备份或只恢复 SQLite 文件。恢复到可信私有父目录下的新0700实例，保留各0600文件，
人工核对复制的 JSON 语法 controller.yaml 中四处绝对私有路径：database、credentials、
management.credentials、authz.credentials。原 authority、浏览器 origin、assets 不变。
恢复服务用户所有权，再更新 unit 的 WorkingDirectory/config/ReadWritePaths，执行
daemon-reload/start；检查 readyz、新登录、发布/账户/审计精确一致及旧会话失效。
验收通过前保留停服原目录和备份并保护凭据，不对恢复状态运行 controller-init。
CI 辅助程序不是生产备份命令，也不是在线/多写者备份过程。

原生 CI 在源码目录外复制安装包，运行时 PATH 为空，用真实 Chromium 完成开户、
人员登录、发布、重启后数据库/审计读取，拒绝错误用途 key 与旧会话，并检查归档。
这不等于已验证打包 Agent→入口→Gitea 的真实部署或主机重启。
