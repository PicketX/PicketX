# 本地人员登录

控制台新增管理员配置的用户名/密码登录，与访问密钥共用现有 Casbin/SCS。
人员账户不等于网络来源、Agent Token。此阶段是单 Controller 本地身份，尚不包含
OIDC、自助注册、数据库账户编辑、按资源委派或申请审批。

## 安全配置

执行 `go build -o picketx ./cmd/picketx`，使用 `picketx password-hash` 从标准输入
读取密码，仅输出 bcrypt cost12 哈希；不接受密码参数。密码要求12–72 UTF-8字节，
不自动去空格，命令只移除一个末尾 LF/CRLF，拒绝嵌入换行。不要把真实密码写入
命令历史、参数或环境变量。交互式 Bash 示例（局部变量未导出）：

```sh
umask 077
IFS= read -r -s -p 'Password: ' picketx_password
printf '\n' >&2
printf '%s' "$picketx_password" | ./picketx password-hash
unset picketx_password
```

在原有私有 `management.credentials` JSON 中保留必需的 principals 数组，新增：

```json
{
  "principals": [],
  "users": [{
    "id": "principal:alice",
    "username": "alice",
    "password_hash": "REPLACE_WITH_COMMAND_OUTPUT",
    "roles": ["role:admin"],
    "disabled": false
  }]
}
```

占位符故意无效，必须替换为命令输出。可保留已有密钥条目；users 可省略，不可
为 null。未知/重复字段、重复身份/用户名、无效角色/哈希使整个凭据来源失败关闭，
也不能继续用密钥绕过。文件必须为私有普通文件（0600），不超过1MiB；两类身份
合计最多128，每人1–16角色。用户名为3–64个小写ASCII字符，首字母/数字，后续
可含点/下划线/连字符；ID 为全局唯一 principal: 标识。仅接受60字符 $2a$12$ 或
$2b$12$ bcrypt 编码。role:viewer只读，role:admin可写，role:blocked拒绝优先；
group和Casbin周期重载沿用[管理 API](MANAGEMENT_API.zh-CN.md)。轮换时写私有
新文件再原子重命名，避免半写JSON；不要提交凭据或把它们上传为测试证据。

## 行为与边界

有启用用户时，/console/ 显示“登录方式→用户名与密码”，默认仍为访问密钥。
支持密码管理器、粘贴与显隐切换、中英文，不写浏览器存储。错误/不存在/禁用账户
均给通用认证失败。session仅公开password_login布尔能力，不公开账户列表/哈希；
全部用户禁用或移除后回到密钥方式。

POST /console/api/v1/login/password 接受严格 {username,password} JSON，不超过4096
字节，仅application/json；准确Host/Origin及Echo CSRF不可省略，不信转发身份/地址头。
额外按真实本地对端每分钟6次、突发5次，返回429；每Access实例最多4个并行bcrypt，
满时503。格式合法的未知/禁用账户做同cost12计算，格式错误可提前拒绝，不承诺绝对
时间侧信道隐私。

登录更新SCS，仅存ID和绑定ID/用户名/密码哈希的独立域SHA256。每个认证请求重新
读身份与角色。密码/用户名轮换、禁用/删除在下一请求使会话失效；角色变化下一操作
生效；退出销毁服务端当前会话，保留既有防在途刷新复活机制。绝对8小时/空闲30分钟、
单进程内存/重启失效不变，不追溯撤回已授权在途操作。重新启用/重建身份前，生成新
随机盐哈希（即使用相同密码），才能永久撤销全部旧Cookie；期间没有请求的“禁用后
原样恢复”不是持久撤销代际数据库。/api/v1仍只接受Bearer密钥，密码/Cookie不能登录。

非回环部署必须沿用可信HTTPS入口并保留Host/Origin、限制Controller直连。限流为
本地进程而非分布式/按人锁定；反向代理后用户共享对端桶，生产入口还需全局防滥用、
连接/请求体限制和监控。没有MFA、密码重置投递、改密页面、分布式会话、登录事件
审计或生产部署安全声明；已接受配置写入的审计会记录真实人员ID。

## 验证范围

Go覆盖严格配置、密钥/密码隔离、拒绝优先、角色刷新/轮换/禁用、CSRF/会话更新及
并发上限；CLI验证标准输入、换行/长度和bcrypt结果。必需真实浏览器使用构建后
Controller/SQLite/SCS/Casbin及真实CLI哈希命令，验证登录、发布读回/审计操作者、
viewer后端拒绝、退出Cookie重放、轮换密码重登、禁用撤销及375/1440布局。保留
密钥/Nginx应用全套验证；不据此声称OIDC、审批、新Agent内核或生产TLS完成。
