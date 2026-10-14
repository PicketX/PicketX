# ADR 0002: Runtime support baseline

> English (default) | [简体中文](0002-runtime-support.zh-CN.md)

- Date: 2026-10-14
- Status: Accepted selections; runtime integration pending
- Authority: [Architecture](../ARCHITECTURE.md), sections 5.11 and 11

## Context

The Controller and Agent need sessions, request protection, communication authentication, metrics and periodic work. These selections complete the support baseline without implementing those features or freezing configuration schemas.

## Decisions

| Concern | Selected baseline | Boundary |
| --- | --- | --- |
| Login sessions | `github.com/alexedwards/scs/v2` v2.9.0 | Memory, existing-GORM database, or Redis store |
| CSRF | Echo v5 built-in middleware | Browser/cookie-authenticated requests; explicit trusted origins and token transport |
| Rate limiting | Echo v5 built-in memory store | Ordinary HTTP and Agent communication have independent settings and counters |
| Agent RPC | `google.golang.org/grpc` v1.84.0 and official `google.golang.org/protobuf` | Agent-initiated bidirectional stream; protobuf tool versions pinned at integration |
| Agent authentication | Independent revocable Agent Token | Derived Agent identity and resource scope; TLS configurable |
| Metrics | `github.com/prometheus/client_golang` v1.25.0 | Controller and Agent collect separately; configurable metrics exposure |
| Periodic work | Go standard-library Timer/Ticker | Cancellable lifecycle, configurable intervals, no overlapping runs of one job |

### Sessions and CSRF

Use SCS memstore, gormstore reusing the existing GORM connection, or goredisstore. Valkey interoperability must be checked with the actual client/server combination; it is not a separately validated store selection. Echo v5 integration needs a thin adapter validated against the selected versions.

Configure absolute and idle expiration. Renew the session token at login and destroy it at logout. Trusted group/role membership freshness and efficient revoke-all-user-sessions remain application responsibilities; policy reload does not refresh Session membership. Switching store configuration does not migrate existing sessions.

Use Echo's built-in CSRF for relevant browser routes and test the pinned middleware's actual behavior, including login/callback scope. Session-cookie security settings must match the selected HTTP/TLS deployment. Store-table initialization and upgrade behavior are decided when sessions are integrated; business-schema versioned migrations remain required.

### Ordinary requests and Agent communication

Keep independent enable/rate/burst/counter settings for ordinary HTTP and Agent communication. Derive authenticated keys from verified user or Agent identity; use explicitly trusted source extraction before authentication.

Echo HTTP request limits do not count messages inside a long-lived gRPC stream. Stream establishment and message processing need separate integration. Memory counters are per process, not a cluster-wide quota.

### Agent identity and transport

Support plaintext gRPC and server-authenticated TLS. Each Agent has an independent Token; the Controller derives identity from verified credentials rather than accepting a caller-selected Agent ID. Support credential revocation and rotation. Disabling an Agent rejects new calls and closes existing control streams.

Plaintext transport provides no confidentiality or server identity verification; tokens and payloads are exposed on that hop. Document controlled-network/encrypted-tunnel deployment and recommend TLS for public direct connections. When enabled, verify the server certificate normally. External TLS termination is supported with an explicit trusted proxy boundary.

mTLS is a future optional authentication mode. It is not a first-release requirement, and no internal CA, Agent certificate issuance or automatic certificate renewal subsystem is selected.

### Metrics and scheduled work

Expose metrics only through configured listeners and access controls; no new default public Agent port is implied. External Prometheus/Grafana deployment is optional. Keep label cardinality bounded; user/IP/Session identifiers belong in logs or authorization records rather than individual time series.

Prefer Casbin's existing periodic load facility where it preserves the configured policy scope; filtered reload must retain its filter. Other modules own cancellable periodic jobs with failure/duration reporting. A singleton multi-Controller job needs database coordination when introduced.

Ticker delivery is not durable task delivery. Record cleanup does not enforce authorization expiry: the kernel or Agent-local deadline mechanism must preserve the declared expiry contract while the Controller is unavailable.

## Deferred integration details

Exact configuration keys/defaults, migrations, external-TLS topology, Token enrollment/rotation protocol, protobuf versions and Agent metrics listeners are specified with their first implemented workflow. Existing product/domain boundaries are unchanged.

## References

- [SCS](https://github.com/alexedwards/scs)
- [Echo middleware](https://echo.labstack.com/docs/middleware)
- [gRPC authentication](https://grpc.io/docs/guides/auth/)
- [Prometheus Go client](https://github.com/prometheus/client_golang)
- [Go time package](https://pkg.go.dev/time)
