# OIDC 浏览器登录

可选OIDC只认证**已存在的人类账户**，不认证源IP、Agent或应用账户。
精确issuer＋sub通过私有显式绑定关联现有账户；邮箱、claims中的roles/groups
不会赋予PicketX权限。每次请求仍读取本地当前角色并由Casbin执行deny优先。
采用固定go-oidc3.21.0/go-jose4.1.4/oauth2 0.37.0，不自行实现JWT验签。

## 配置

保留现有数据库策略权威、浏览器origin和独立恢复管理密钥。先在账户页创建人类
账户并复制不可变ID（也可用文件人类账户）；不能关联API-key principal。
设置重启生效的`browser.oidc_config: /private/oidc.json`，环境变量为
`PICKETX_BROWSER_OIDC_CONFIG`；要求启用浏览器、绝对路径、0600普通非符号链接文件。
严格JSON字段为`issuer`、`client_id`、`client_secret`、`bindings`（绑定文件绝对路径）。
英文文档给出了完整格式，示例不是已注册的真实提供方或凭据。

在IdP注册保密authorization-code客户端，精确回调为配置的origin加
`/console/api/v1/oidc/callback`，保留端口。仅`openid` scope和S256 PKCE。
issuer/auth/token/JWKS必须HTTPS且正常验服务器证书；仅开发时允许字面量loopback
HTTP，不允许远程明文、DNS开发特例或关闭TLS验证。既有入口继续负责TLS。

单独的私有绑定文件格式：
`{"bindings":[{"subject":"精确sub","principal_id":"principal:现有人类ID","generation":"32位小写hex"}]}`。
generation用16随机字节的小写hex，示例说明不是可用值。最多128个唯一subject、1MiB。
`{"bindings":[]}`明确拒绝全部OIDC；null、未知/重复字段、重复subject、非法generation、
公开权限/非普通文件均fail closed。用原子替换更新完整文件，不回写、不导入或切换
账户/策略权威。提供方、client、secret、绑定路径修改需重启；绑定内容逐请求读取。

## 行为和边界

只在配置后出现“使用组织账户登录”。POST start要求精确Host/Origin和Echo CSRF，
额外直接peer本地限流6次/分钟、突发5。独立256bit state/nonce/verifier/关联随机值；
最多128个待处理、绝对5分钟、一次消费、重启丢失。新尝试覆盖该浏览器关联Cookie，
请完成最新一次。只支持顶层GET回调，不支持form_post、implicit或hybrid。

回调用单独路径限定HttpOnly SameSite=Lax Cookie；普通SCS保持Strict，HTTPS均Secure。
校验state/关联/PKCE、精确issuer、audience、签名、expiry、nonce、非空subject、iat/azp。
外部请求5秒/1MiB且不跟随重定向。失败仅固定本地跳转及通用英中文提示，不泄露IdP详情。
成功重建SCS，只保存本地ID/账户指纹、subject/绑定指纹和令牌绝对期限；不保存外部
access/ID/refresh令牌、client secret、邮箱或claim角色。最长不超过ID令牌expiry和既有
8小时/空闲30分钟。未实现refresh token、静默刷新、远端logout或分布式会话/配额。

每次请求重读绑定及人类账户；移除/更换绑定、generation轮换、禁用/移除或账户密码版本
变更使旧会话失效。数据库账户编辑永久递增版本，重新启用也不会恢复旧Cookie。
文件模式若移除后原样恢复且中间没有请求，旧Cookie可能复活，因此恢复时必须轮换
绑定generation；文件角色修改下次请求生效。绑定存储不可用返回503，不缓存授权；
已授权的进行中操作不会撤销。登录或退出均不改变网络Grant。

本地退出销毁PicketX会话，**不会退出IdP**，IdP可能立即重新认证。IdP端禁用/会话撤销
并不逐请求查询，现有边界是本地账户撤销和ID令牌绝对expiry。`/api/v1/*`仍只接受独立
Bearer管理密钥。配置/discovery失败时启动以不含详情的错误结束；修正文件，或取消
OIDC设置后重启再使用本地密钥/密码恢复。没有配置就绪的进程不承诺可登录。

## 验证声明

Go使用真实HTTP discovery/JWKS/code及RSA签名夹具，覆盖PKCE、重放/并发、恶意claims、
issuer别名、绑定及当前人类权限。实际Chromium→构建Controller/React/SQLite覆盖跨站
127.0.0.2→127.0.0.1回调、发布及真实审计、权限变化、绑定撤销、nonce拒绝和本地退出。
发行者是明确标注的**签名协议夹具**，不是部署的真实IdP。既有Linux/Agent/Nginx/应用
门禁保留；真实提供方仍需操作员注册、HTTPS拓扑及实测，不能用夹具宣称生产MFA、
任意IdP兼容性或全部端到端目标完成。
