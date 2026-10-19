# Trusted HTTP authorization (alpha)

English (default) | [简体中文](HTTP_AUTHORIZATION.zh-CN.md)

This increment adds source-backed, per-request authorization, not user authentication,
an application proxy or a production-complete management workflow. Exact acceptance
head/tree/CI and limitations are in [the delivery checkpoint](AUTONOMOUS_DELIVERY.md)
and Core PR16. Existing ingress owns TLS, routing and forwarding; applications keep
their authentication. Agent source/wire and its native evidence are unchanged.

## Policy and trust

`HTTPRoute` is part of the same strict `picketx.io/v1alpha1` publication as Resources,
source Grants and NetworkDenies, in either file or database authority mode. It pins
`resourceRef:{name,uid}` and `endpointName` to an existing TCP Endpoint. It requires
exact canonical `host`, explicit nonempty `methods` and at least one path selector. Methods are
GET/HEAD/POST/PUT/PATCH/DELETE/OPTIONS, with no implicit HEAD or wildcard. Paths are
exact. Optional `pathPrefixes` adds path-segment prefixes: `/repo` matches itself,
`/repo/` and `/repo/child`, not `/repository`. Explicit `/` covers all canonical
paths; other trailing-slash prefixes are invalid. Exact and prefix selectors form
a union, at most128 combined. Omitted lists contribute nothing; explicit empty or
null lists are invalid, even with another nonempty list. No implicit wildcard,
glob or regex is added. Non-ASCII, escapes (`%`), backslashes, duplicate slashes and dot
segments are rejected. Query strings do not affect authorization and are never audited.
Do not use this adapter for application actions distinguished only by query parameters.
Host is a lowercase DNS/IP authority, optionally with explicit numeric port; include
the actual port sent by clients. A retained route UID cannot change Resource/Endpoint/Host;
create a new UID and update the ingress mapping explicitly for a new target.

Every matching request still requires an active source Grant and no matching active
NetworkDeny. Existing union, independent absolute expiry/future activation and revocation
semantics apply. The server supplies evaluation time. No workflow actor, Cookie,
X-Forwarded-For or claimed username becomes a verified request identity. Policies that
require verified user identity are not supported by this alpha contract, never downgraded.
HTTP allows do not confirm or bypass any configured network gate. Resources without
network Bindings may explicitly use this HTTP-only integration. No Agent HTTP enforcement
or Controller-disconnected HTTP provider is supplied here.

## Structured management

In database-authority mode, use **HTTP routes** in the operator console to create
and inspect routes. Select a pinned Resource and TCP Endpoint, enter the
actual request Host (including port), explicitly select methods and list exact
paths and/or segment prefixes in separate fields. The API uses `path_prefixes`,
the manifest uses `pathPrefixes`; unused lists must be omitted, not empty/null.
A reason and current revision are required. Viewer roles cannot write.
Failed/conflicting operations retain drafts; inspect the catalog and explicitly
use the latest route revision before retrying an ambiguous result.

The corresponding Bearer API is `POST /api/v1/http-routes` with strong `If-Match`
and strict JSON (maximum16KiB), for example:

```json
{"name":"app-read","resource_name":"app","resource_uid":"app-v1","endpoint_name":"web","host":"app.example","methods":["GET"],"paths":["/allowed"],"reason":"Expose exact read route"}
```

The server returns201 `{uid,revision}` and ETag; `GET /api/v1/inventory` includes
`http_routes`. Cookie equivalents use `/console/api/v1` plus Origin/CSRF. The
publication compiler validates the complete candidate and audit commits atomically.
Creation does **not** add source Grants, configure a caller, change ingress or
prove live application. Independently add the generated UID to the private caller
allowlist and fixed ingress mapping, then grant the intended source. Editing and
removal still use the advanced desired-configuration editor.

## Separate local listener

Disabled by default. Add runtime settings (file/env through existing strict Koanf):

```yaml
authz:
  enabled: true
  network: tcp
  address: 127.0.0.1:9091
  credentials: /etc/picketx/ingress-callers.json
```

IPv6 loopback `[::1]:9091` is also supported. DNS/wildcard/nonloopback listeners are rejected.
Alternatively use `network: unix`, `address: /run/picketx-authz/auth.sock`; the parent must
already be a private0700 directory. The socket must not exist; creation uses0600 and
cleanup only removes the exact owned object. Do not expose/forward this listener publicly.
It has no console/admin endpoints. Read/header/write limits3s/2s/3s, idle30s, max headers32KiB;
independent local rate budget128/s burst256. No automatic limit tuning/capacity claim.

Private0600 credential JSON (not an Agent/management credential document):

```json
{"callers":[{"id":"nginx","sha256":"<SHA256 hex of dedicated random Token>","routes":["route-v1"]}]}
```

Use32–256 printable ASCII characters of independent random secret; never reuse Agent,
management or application keys. At most128 callers/128 route UIDs each/1MiB. Duplicate,
unknown/null or invalid fields reject the complete file. Reload is per request. Removal
or rotation rejects the next callback; malformed/unreadable files return503 and readiness
fails. Route scope is a trust boundary/upper bound, not a source Grant. Protect the file,
socket, Nginx generated configuration and upstream network access from untrusted local users.

`POST /authz/v1/decisions` accepts JSON only, max16KiB, dedicated Bearer:

```json
{"route":"route-v1","source":"192.0.2.1","host":"app.example","method":"GET","uri":"/allowed"}
```

Source must be a canonical IPv4/IPv6 host, not CIDR/mapped/zone. Unknown/out-of-scope routes
return403. Valid decisions return200 `{allowed,reason,revision,grant_ids,deny_ids,next_change,
cache_ttl:0}`. Invalid credentials401, invalid context400, unavailable/persistence failure503.
All responses no-store; cache neither allows nor denies. Do not rely on existing connections
as permission. Per-request checks cannot terminate an already upgraded WebSocket/stream or
cancel an in-flight application operation; disable upgrades where that contract is required.

## Nginx integration

See [template](../examples/http/nginx.conf.template) and [example policy](../examples/http/policy.yaml).
The example intentionally authorizes only loopback sources and fixed paths; configure real
host/sources/endpoint ports and actual routing mappings deliberately. Its two resources share
one acceptance application to prove the request gate distinguishes routes, not L4 identities.

`GET /authz/v1/nginx` requires exactly one each of `X-PicketX-Route`, `-Source`, `-Host`,
`-Method`, `-URI`. Nginx must use an `internal` location, a fixed route UID for each actual
upstream mapping, dedicated secret, `$remote_addr`, original `$http_host`/`$request_uri`,
and the original method captured in the protected location before auth_request. Clear body,
Content-Length and inherited request headers. Never use client headers/URI parameters to select
the route or caller credential. The provided template overwrites all callback context and does
not enable real_ip or trust X-Forwarded-For. If an upstream trusted load balancer changes source
observation, design/configure its bounded real_ip trust separately; do not blindly enable it.

204 allows;403 denies;401 rejects an unauthenticated integration; other callback statuses/errors
must fail closed. Nginx auth_request owns forwarding behavior. Keep application upstreams
unreachable by bypass paths; auditing a callback does not prove all application access used it.
Add TLS at the existing ingress for deployment, never send the dedicated callback key over an
untrusted connection. Configure bounded proxy timeouts/cache off and do not combine an alternate
`satisfy any` allow that bypasses auth_request. HTTPS deployment/certificate topology is not
validated by the loopback HTTP acceptance scenario.

## Durability and records

Versioned SQLite schema3 adds `controller_http_audit`, with transactional upgrade from1/2.
Back up the complete private DB while stopped before deploying an upgrade; older Core versions
cannot read schema3. File/database ownership remains bound; policy file polling cannot overwrite
managed intent. Each valid scoped decision is serialized with publication, evaluated against the
accepted revision, then durably records caller/route/source/revision/result/reason/contributing
IDs/time **before** returning. Audit failure returns503, never an allow.
Evaluation time is refreshed after the publication lock. If a relevant Grant/deny boundary
passes during durable audit, recompute and append a fresh decision before returning; at most
two evaluations, then503 if boundaries keep advancing. Intermediate records are decisions,
not proof of forwarding. This is not an atomic reservation of an application's operation.
Raw URI/query/Host/method, body, headers and credentials are not stored. Repeated callbacks create records; retention and
capacity controls are not implemented, so provision disk and monitor growth. Single-process
SQLite serialization is intentional; no HA/throughput guarantee.

`GET /api/v1/http-decisions` (independent management Bearer) and
`GET /console/api/v1/http-decisions` (SCS) require current `audit:read` Casbin permission.
They use exclusive `after` cursor/default0 and `limit`1..100/default50; events ascending and
`next_after`. This increment does not add a structured decision-table page.

## Verification and remaining scope

Tests retain full strict model/Grant/deny/deadline semantics, audit fault/reopen/schema upgrade,
credential revocation/RBAC and actual TCP/Unix listener lifecycle. Actual CI requires a real
Nginx binary with auth_request and real Chromium/Go/SQLite/application, not a mock decision
server. It checks per-request keepalive target/method/path/source spoof isolation, browser
publication/revocation/deadline and minimal audit. Local Unix socket EPERM and absent Nginx/
Chromium are not counted as passing; Linux CI supplies those required execution capabilities.
Structured Resource/Grant/approval/status pages, human/OIDC identity, user-identity HTTP policies,
production HTTPS ingress and final combined Linux workflow remain before product completion.
