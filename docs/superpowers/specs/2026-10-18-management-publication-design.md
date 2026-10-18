# Database-authoritative management publication

Status: implementation scope approved by the user's autonomous-delivery mandate.
This increment ships an actual authenticated API, not human login, L7 rules or UI.

## Boundary and decision

Keep the existing file publication mode unchanged by default. Add explicit
`source.mode: database`; reject policy/facts file paths in that mode. Agent credential
reload continues independently, but never reads policy/facts files. A fresh managed
database starts with an explicit empty configuration. Existing databases are bound to
their source mode; switching a populated file database requires a future explicit
migration, not an implicit import or fallback. Database mode requires the running
Agent service and its configured authority in this increment.

Expose `GET/PUT /api/v1/publication`, `GET /api/v1/audit` and `GET /api/v1/me` on the
existing ordinary HTTP listener only in database mode. An independent private credential
file contains stable API principal IDs, SHA256 Token digests and trusted roles. Tokens
are 32–256 printable ASCII bytes; Agent Tokens are never accepted by this API merely
because they are Agent credentials. No cookies, browser login, CORS or forwarded identity
are introduced. Re-read the credential document on every request: malformed/unreadable
credentials fail closed, and removal revokes the next request. The host configuration
owner is trusted; external ingress must provide HTTPS for non-loopback use.

Use the selected Casbin v3 and official GORM adapter, sharing the publication database.
One injected SyncedEnforcer per process holds shared role rules, not user memberships.
Seed admin/viewer/blocked role policy only when its own versioned bootstrap marker is
absent. A single combined evaluation across all trusted roles preserves deny precedence.
Only periodic reloads observe external policy changes; failures mark authorization
unavailable until a successful reload, never a fresh success. Log p/g counts, duration
and configured scale/cost guidance without changing the selected interval. SQLite only
is implemented; importing the official adapter's other drivers does not establish
PostgreSQL or clustered support.

## Publication and audit contract

GET returns the exact accepted policy YAML and facts JSON as strings, revision and
authority, never re-encoding presence-sensitive configuration structs. PUT accepts
`policy`, `facts`, `reason` and requires a strong `If-Match: "<revision>"` header.
Missing precondition is 428; stale revision is 412 even for an otherwise semantic no-op.
Strict JSON rejects duplicate, unknown, aliased and null fields. Validator checks DTO
sizes; the existing shared compiler enforces all domain invariants. Retargeting existing
Grants remains rejected. The complete candidate is preflighted before commit.

The expected revision check, publication CAS and audit insertion share one transaction
under the store lock. A successful semantic no-op consumes no revision and creates no
change event. Changed writes persist actor, reason, revision, semantic digest and UTC
time, but no credential or policy body. Audit failure rolls back the publication. Audit
reads use bounded keyset pagination. Database durability or post-commit distribution
failure disables writes and stops the runtime; restart republishes the committed state.
HTTP success means accepted desired state, never native application confirmation.

The publication/distribution owner serializes API writes and credential polling; requests
cannot publish revisions out of order. Invalid Agent credential reload blocks managed
writes while still allowing management inspection. Status remains separate from policy.

## Acceptance and next slice

TDD proves mode separation, v1 migration, stale/no-op CAS, concurrent writers, atomic
audit rollback, credential revocation, multi-role deny, periodic reload failure recovery,
strict bounded HTTP errors, and real TCP HTTP → SQLite → gRPC update → restart. File
mode must not expose management routes. Complete Go race/vet/build, Web lockfile checks,
static Controller smoke and independent full-branch review precede exact-head merge.
Generate OpenAPI 3.1 using pinned Swaggo v2.0.0-rc6 and check the endpoint contract in CI.
Human SCS login, self-service/approval, L7 adapters/local decision interface and real
React pages remain the next vertical slices; no final product acceptance is claimed.
