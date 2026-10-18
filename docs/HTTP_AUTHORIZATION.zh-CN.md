# 可信 HTTP 授权（alpha）

[English（默认）](HTTP_AUTHORIZATION.md) | 简体中文

本增量提供基于来源的逐请求授权，不是用户认证、应用代理或完整管理产品。精确head/tree/CI与限制见[交付检查点](AUTONOMOUS_DELIVERY.md)和Core PR16。现有入口负责TLS、路由、转发；应用保留自身认证。Agent源码/协议及原生证据未改变。

## 策略与信任

`HTTPRoute`与Resource、来源Grant、NetworkDeny属于同一严格`picketx.io/v1alpha1`完整发布，沿用文件或数据库单一权威。它通过`resourceRef:{name,uid}`和`endpointName`固定引用已有TCP入口；必须明确非空`host`、`methods`、`paths`。Host为规范小写DNS/IP authority，可含明确数字端口，应与客户端实际发送值一致。方法仅GET/HEAD/POST/PUT/PATCH/DELETE/OPTIONS，不默认HEAD、不支持通配。路径仅精确匹配；拒绝非ASCII、百分号转义、反斜杠、重复斜杠、点段。查询参数不参与授权，也不审计；不能用它保护仅靠查询参数区分的应用操作。保留同一route UID时不能更换Resource/Endpoint/Host；更换目标应新建UID并显式更新入口映射。

路由匹配仍须来源有有效Grant且没有有效NetworkDeny；Grant并集、独立绝对期限、未来生效、撤销与deny优先不变。时间由服务端提供。申请人、Cookie、X-Forwarded-For、声明用户名都不是已验证请求身份。本alpha不支持要求验证用户身份的策略，也不降级为IP允许。HTTP允许不代表内核应用或绕过已有网络门禁；未配置网络Binding的资源可明确仅用HTTP集成。本段未提供Agent HTTP执行或Controller断连HTTPProvider。

## 独立本地监听器

默认关闭，通过已有严格Koanf文件/环境配置启用：

```yaml
authz:
  enabled: true
  network: tcp
  address: 127.0.0.1:9091
  credentials: /etc/picketx/ingress-callers.json
```

支持IPv6回环`[::1]:9091`，拒绝DNS、通配、非回环监听。或用`network: unix`及`address: /run/picketx-authz/auth.sock`：父目录预先创建为私有0700，目标不得存在，socket0600，仅清理自己创建的原对象。不可公开转发该监听器；其中没有管理/工作台API。读/头/写超时3s/2s/3s，idle30s，头32KiB；独立本地限流128/s、burst256。未测容量、不自动调参。

私有0600凭据文件（不能混用Agent/管理凭据）：

```json
{"callers":[{"id":"nginx","sha256":"<独立随机Token的SHA256十六进制>","routes":["route-v1"]}]}
```

独立随机Token32–256可打印ASCII，禁止复用Agent、管理、应用密钥。最多128调用方、每方128 route UID、文件1MiB。重复、未知、null及非法字段拒绝整份文件。每次请求重读，删除/轮换使下一回调失败；损坏/不可读返回503并使readiness失败。调用方作用域是信任上限，不产生来源权限。保护凭据、socket、生成的Nginx配置和后端访问不受不可信本地用户控制。

`POST /authz/v1/decisions`仅JSON、最大16KiB、独立Bearer：

```json
{"route":"route-v1","source":"192.0.2.1","host":"app.example","method":"GET","uri":"/allowed"}
```

source必须规范IPv4/IPv6主机地址，不能CIDR、mapped或zone。未知/越权route403；合法决策200返回`allowed,reason,revision,grant_ids,deny_ids,next_change,cache_ttl:0`。无效凭据401、非法上下文400、服务/审计不可用503。全部no-store，不缓存允许或拒绝。已有连接不是权限；逐请求检查不能终止已升级WebSocket/流或已开始的应用操作，需要该保证时禁用升级。

## Nginx集成

参见[模板](../examples/http/nginx.conf.template)和[策略示例](../examples/http/policy.yaml)。示例故意仅授权回环及固定路径；实际部署须显式设置Host、来源、入口端口和真实路由映射。两个测试资源共享一个应用，仅证明HTTP路由隔离，不声称L4可区分。

`GET /authz/v1/nginx`要求每个`X-PicketX-Route/Source/Host/Method/URI`恰好一个。Nginx必须使用`internal`回调位置、每个真实上游映射固定route UID、独立密钥、`$remote_addr`、原始`$http_host/$request_uri`，并在保护location中先保存原始方法。清除请求体、Content-Length和继承头。绝不使用客户端头/查询参数选择route或密钥。模板覆盖所有回调上下文，不启用real_ip、不信任X-Forwarded-For。若前置可信负载均衡改变来源观测，另行设计受限real_ip信任，不能盲目启用。

204允许、403拒绝、401表示集成身份无效，其余状态/超时必须fail-closed。Nginx负责转发。应用后端不得有绕过入口的访问路径；回调审计不证明所有应用访问经过它。部署TLS仍由现有入口负责，不能经不可信网络明文发送回调密钥。设置有界超时、关闭缓存，不能用`satisfy any`等旁路授权。回环HTTP验收不代表已验证生产HTTPS证书/入口拓扑。

## 持久与审计

版本化SQLite schema3新增`controller_http_audit`，支持从1/2事务升级。升级前停止服务并备份完整私有数据库；旧Core不能读schema3。文件/数据库权威绑定不变，文件轮询不能回写覆盖管理状态。合法且作用域内的决策与发布串行，在当前已接受revision上求值，**先**持久记录调用方、route、source、revision、结果/原因、贡献ID、时间，才返回。审计失败503，绝不返回允许。不记录原始URI/查询/Host/方法、请求体、头或凭据。回调逐次追加记录；未实现保留期/容量控制，需规划磁盘并监测增长。单进程SQLite串行未测吞吐，不声称HA。

`GET /api/v1/http-decisions`使用管理Bearer，`GET /console/api/v1/http-decisions`使用SCS；均须当前Casbin `audit:read`。`after`为排他游标默认0，`limit`1..100默认50，升序返回events/next_after。本段未新增结构化决策表格页面。

## 验证与后续

测试覆盖严格模型/Grant/deny/期限、审计故障/重开/升级、凭据撤销/RBAC、真实TCP/Unix生命周期。实际CI必须运行有auth_request的真实Nginx及Chromium/Go/SQLite/HTTP应用，验证keepalive逐请求目标/方法/路径/伪造来源头隔离、浏览器发布/撤销/期限和最小审计。本地Unix EPERM及缺少Nginx/Chromium不计通过，必需能力交由Linux CI。结构化资源/Grant/审批/状态页面、人员/OIDC、HTTP已验证用户策略、生产HTTPS及最终Linux联合工作流仍待完成。
