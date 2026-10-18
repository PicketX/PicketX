# HTTP授权示例

参见[部署/信任契约](../../docs/HTTP_AUTHORIZATION.zh-CN.md)（[English](README.md)）。`policy.yaml`仅回环示例，须把`__HOST__`替换为精确入口authority。使用facts `{"authorityID":"<实际权威>","agents":[],"contexts":[]}`发布，明确仅HTTP执行；没有网络Binding，不会放行内核入口。需要两层时另行配置正确网络Binding，两者均须通过。

通过已有管理API/工作台CAS发布，不从运行配置添加权限。`nginx.conf.template`是真实浏览器CI使用的auth_request配置。替换prefix/监听/应用/authz端口及全新独立Token，生成私有0600文件，绝不提交含密钥文件。调用方凭据保存其SHA256并限权`route-v1`/`route-private-v1`，仅前者有来源Grant。部署须修改实际Resource入口端口和可信固定路由映射。

回环模板没有生产TLS或上游隔离。应保护后端，不能信任客户端转发头，保护回调密钥，在现有入口添加TLS；这不是通用反向代理产品。

必需CI命令：在`apps/web`运行`npm ci && npm run build && npm run test:e2e`，需Go、真实Chromium及启用auth_request的Nginx；缺少能力即失败，不跳过。
