# 人员账户管理

[English](MANAGED_ACCOUNTS.md) | 简体中文

本目录显式启用后管理本地人员，不管理 Agent Token、入口凭据、网络来源或业务应用账户。
既有文件用户模式不变；本段不包含 OIDC、自助注册、邮件找回或多副本会话。

## 显式启用

使用数据库发布权威，在运行配置中设置：

```yaml
source: {mode: database}
management:
  credentials: /etc/picketx/management-keys.json
  users_mode: database
```

0600 私有密钥文档必须包含 principals，不得包含非空 users，且至少保留一个具有
role:admin、没有 role:blocked 的应急管理密钥。全部人员管理员被禁用时通过该密钥恢复，
请妥善保护。principal:account- 前缀保留给数据库人员，密钥不能使用。Agent 凭据保持独立，
不会自动导入文件用户。

初始化会持久记录数据库用户权威。此后切回默认 file 模式或省略 users_mode 均失败关闭。
其他进程初始化该数据库的用户权威后，既有文件模式进程的下一次凭据读取和就绪检查也失败。
加入文件用户或丢失应急管理员密钥同样使认证和就绪失败；应恢复有效私有密钥文档，不合并两种
权威来源。数据库与应急密钥应一起备份，不直接编辑身份表。

## 页面流程与权限

account:read 显示账户页，account:write 允许修改。管理表迁移至版本3，仅为 admin
增加这两项默认许可，blocked 显式拒绝；viewer/requester 默认不能读取目录。
共享 Casbin 自定义规则保留，页面中人员角色只选四种内置角色。

创建时填写唯一、不可改名的小写 ASCII 用户名（3–64字符）、密码（12–72 UTF-8**字节**）、
角色和原因（1–512UTF-8字节）；最多128人。支持角色调整、禁用/启用和可选密码重置，不提供
删除、改名或用户名复用。重置密码留空保留原哈希。每次修改，包括同值和仅角色修改，均使此前
所有人员会话失效。重新启用或重启不会恢复旧 Cookie；已完成鉴权的在途请求不追溯撤销。

账户目录版本独立于配置发布版本。刷新不暗中推进已有草稿的 CAS 基准。冲突或保存结果未知时，
先查看最新记录与审计，再明确清除草稿后重试。密码在保存请求开始时清除，未知结果核对后方可
重新输入；不返回、不写审计、不写浏览器存储。账户修改不授予网络访问，也不代表 Agent 已应用。

## API

| 方法 | Bearer 路径（浏览器别名位于 /console/api/v1） | 权限 |
| --- | --- | --- |
| GET | /api/v1/accounts | account:read |
| POST | /api/v1/accounts | account:write |
| PUT | /api/v1/accounts/{id} | account:write |

GET 返回人员、最近50条账户审计、目录版本和观测时间。写入需要规范强 If-Match（目录版本）、
严格16KiB JSON、原因；更新必须显式传 disabled 布尔值。账户修改、目录 CAS 和审计在同一事务
提交。缺少版本428、过时412、非法400、账户不存在404、不可用503。仅数据库用户模式开放。
浏览器别名还要求匹配配置的 Origin、原生 Echo CSRF 和 SCS 身份；Cookie 不认证 Bearer API。
bcrypt12 与登录共享四个工作槽；密码和哈希不进入 SQL 错误日志。
