# HTTP authorization example

See the [deployment/trust contract](../../docs/HTTP_AUTHORIZATION.md) ([中文](README.zh-CN.md)).
`policy.yaml` is a loopback-only example; replace `__HOST__` with the exact ingress authority.
Publish it with facts `{"authorityID":"<your authority>","agents":[],"contexts":[]}` for
explicit HTTP-only enforcement. It declares no network Binding and grants no kernel entry.
For both layers, declare correct network Bindings separately and require both gates.

Use the existing managed API/console CAS publication; never add permission through runtime
configuration. `nginx.conf.template` is an actual auth_request configuration used by the
required browser CI. Substitute prefix/listen/application/authz ports and a fresh independent
Token into a private0600 generated file, never commit the rendered secret. The caller file
must hold its SHA256 and scope `route-v1`/`route-private-v1`. Only the former has source Grants.
Set actual Resource endpoint ports and trusted fixed routing mappings for deployment.

The loopback template deliberately does not implement production TLS or upstream isolation.
Keep upstreams private, do not trust client forwarded headers, protect callback credentials,
and add TLS at the existing ingress. It is not a generic reverse-proxy product.

Required CI command: `npm ci && npm run build && npm run test:e2e` in `web`, with Go,
actual Chromium and Nginx/auth_request installed. Missing capabilities fail, never skip.
