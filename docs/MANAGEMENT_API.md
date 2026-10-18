# Managed configuration API

> English (default) | [简体中文](MANAGEMENT_API.zh-CN.md)

Implemented: one SQLite Controller, independent API principals, Casbin RBAC, complete
desired-state read/replace, revision preconditions and transactional change audit.
Human/SCS login, OIDC, self-service/approval, HTTP L7 decisions, live management pages,
PostgreSQL and multi-Controller operation remain unimplemented.

## One authority

Default `source.mode: file` is compatible with previous deployments and exposes no
management routes. For the API, use a NEW private database and explicit database mode:

```yaml
http: {listen: "127.0.0.1:8080"}
agent:
  enabled: true
  listen: "127.0.0.1:9090"
  authority: "controller:example"
source: {mode: database}
database: "var/managed/controller.sqlite"
credentials: "/etc/picketx/agent-credentials.json"
management:
  credentials: "/etc/picketx/management-principals.json"
  reload: "30s"
  warn_rules: 10000
  warn_load_percent: 50
  requests_per_second: 128
  burst: 256
poll: "1s"
shutdown: "5s"
```

Database mode rejects policy/facts file paths and never reads/falls back to file policy;
Agent credential polling continues independently. A fresh database persists an explicit
empty configuration at revision1 with a `system:bootstrap` audit event. Schema version2
binds the source mode. Deployed version1 publications migrate as file-authoritative;
database mode refuses to silently adopt them. Preserve the old database/source and
deliberately initialize a separate managed database. No in-place authority-switch command
exists yet. Never delete a live database to switch modes; existing explicit withdrawal
and Agent recovery rules still apply.

HTTP itself has no TLS: bind loopback behind trusted HTTPS ingress or use a controlled
encrypted hop. Never expose Tokens over untrusted plaintext. Management and Agent
credentials are independent; do not reuse Tokens. The API does not trust forwarding
identity headers or provide CORS/cookie/session authentication.

## Principals and RBAC

Use a private regular credential file (mode0600, maximum1MiB), held by the trusted host
configuration owner. Shape (not a usable credential):

```json
{"principals":[{"id":"principal:operator","sha256":"<64 hex SHA256 digits>","roles":["role:admin"]}]}
```

Generate a cryptographically random Token; keep its raw value in a private caller secret
store, not Git, URLs, process arguments or shell history. Store only SHA256 in the file.
Send exactly one `Authorization: Bearer <Token>` header. Tokens are32–256 printable ASCII
bytes without whitespace. Maximum128 principals,16 roles each,128-byte IDs. Prefix IDs
with `principal:` and role/group IDs with `role:`/`group:`; suffixes use ASCII letters,
digits, dash, underscore or dot. Duplicate IDs/digests/roles and ambiguous/unknown/null
JSON are rejected. Empty principals deliberately disable authenticated access.

The file is checked at startup and per request. Removal/role edits affect the next
request; already-authorized requests are not retroactively cancelled. Replace atomically
and retain permissions. Malformed/unreadable files yield503/not-ready, not cached success.

One Casbin3.11.0 SyncedEnforcer uses official GORM adapter3.41.0 and the same SQLite
connection. Principal membership is resolved from trusted credentials, never mirrored
into Casbin. Shared `g` rules support group-to-role/role inheritance. All subjects enter
one deny-precedence evaluation. Initial rules are seeded only once if policy is empty:

| Role | publication:read | publication:write | audit:read |
| --- | --- | --- | --- |
| role:admin | allow | allow | allow |
| role:viewer | allow | default deny | allow |
| role:blocked | deny | deny | deny |

These are GLOBAL complete-publication operations, not per-resource delegation/tenancy.
Admin plus blocked is denied. The adapter initializes only its own Casbin tables;
business migrations remain versioned. Shared policy reload is periodic only, using the
configured positive interval (maximum24h). Trusted direct DB changes are not a public
API and are visible only after successful reload. Failure disables authorization and
readiness until recovery. Startup logs report p/g counts and load cost. Configurable
rule/cost thresholds suggest longer intervals/scoping but never change configuration.
Longer intervals increase revocation delay, not reduce resident policy size.

Ordinary HTTP limits key on actual socket peer IP, ignoring forwarding headers. Behind
ingress, callers share its bucket. Agent stream/message limits remain separate. SCS
browser sessions and their CSRF/membership-freshness boundary are a later slice.

## HTTP contract

All responses use `Cache-Control: no-store`; errors use `{"code":"…"}` without secret,
database or compiler details. Every operation has backend authorization:

| Route | Permission | Result |
| --- | --- | --- |
| GET /api/v1/me | authenticated | Trusted ID/roles and current allowed operations |
| GET /api/v1/publication | publication:read | authority/revision/exact policy and facts strings, strong ETag |
| PUT /api/v1/publication | publication:write | Validate/commit complete desired state with audit, distribute |
| GET /api/v1/audit?after=0&limit=50 | audit:read | Ascending change events and next_after cursor |

PUT body: `{"policy":"<complete YAML>","facts":"<complete JSON>","reason":"<audit reason>"}`.
Use documented facts JSON (including numeric family arrays); never serialize arbitrary
internal Go structs. Policy/facts each have a4MiB byte limit; reason is nonblank and at
most512 UTF-8 bytes; complete JSON body is at most12MiB. Shared compiler validation and
replacement/identity constraints remain authoritative. DTOs reject duplicate/null/
unknown/aliased keys and multiple documents. Only application/json, no form/query binding.

GET returns e.g. `ETag: "2"`; PUT requires exactly `If-Match: "2"`. Weak/wildcard/multiple
tags and leading zeros are invalid. Missing precondition428; stale412 EVEN for no-op.
Current semantic no-op preserves exact accepted inputs/revision and adds no audit.
Concurrent writers cannot overwrite each other. Change and actor/reason/revision/digest/
UTC audit are one transaction; audit failure rolls back both. Audit keyset cursor is
exclusive, limit1–100/default50; unknown/duplicate/malformed queries fail. Empty pages
retain next_after; no total or immutable history snapshot is implied.

401 invalid principal;403 denied;400 invalid fields/configuration;413 oversized;415
non-JSON;429 local rate limit;503 unavailable credentials/RBAC, durability, publication
owner or delivery. Bad Agent credentials block writes but not management inspection.
Durability or post-commit delivery failure stops admission/process; restart replays exact
durable state. A503 after commit is deliberately ambiguous: GET after recovery before
retrying with a new ETag. HTTP200 means accepted desired state, NOT Agent/kernel applied.
No Agent heartbeat/status API is added; existing RECEIVED/APPLIED limits remain.

## Documentation and evidence

Pinned Swaggo2.0.0-rc6 produces [OpenAPI3.1](api/swagger.json); embedded module metadata
checks the tool (its --version only says2.0.0). Deterministic normalization adds same-origin
server/strict DTO constraints omitted by rc6. Go contract tests check routes/status/auth/
DTO shape. Locked Swagger Parser13.1.0/Ajv2020 validates the LOCAL document with all
remote/file resolution disabled; no CLI telemetry. No documentation HTTP route or
renderer is shipped yet; selected future UI remains self-hosted Swagger UI5.33.2.

Tests cover actual HTTP/TCP→SQLite→gRPC→restart, independent revocation, stale/no-op CAS,
audit rollback and post-commit failure recovery. Static smoke inspects actual ELF and
runs file-mode gRPC plus managed Token/Casbin/API/audit/restart/SIGTERM. These are Controller
evidence, not native Linux/L7/browser product completion. Exact CI/head evidence and
remaining work are in AUTONOMOUS_DELIVERY.md.
