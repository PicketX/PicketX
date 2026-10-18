# Running the Controller publisher

English (default) | [简体中文](CONTROLLER_RUNTIME.zh-CN.md)

M2D2 connects the typed Agent stream to the real `picketx-controller` process, with strict runtime configuration, durable SQLite publication and credential reload. This is a runnable publisher, not yet a PicketXD Controller provider, management application or Layer7 authorization service.

## Start a controlled local example

Run from the repository root. The existing static example contains historical absolute deadlines; the publisher preserves them and does not renew expired access. Its trusted-context facts are diagnostic examples, not live Agent attestation.

```sh
mkdir -p var/controller
chmod 700 var var/controller
python3 - <<'PYSETUP'
import hashlib,json,pathlib,secrets
root=pathlib.Path('var');token=secrets.token_hex(32)
(root/'agent.token').write_text(token);(root/'agent.token').chmod(0o600)
(root/'credentials.json').write_text(json.dumps({'credentials':[{'id':'agent-a-1','agent_ref':'agent-a','sha256':hashlib.sha256(token.encode()).hexdigest()}]}))
(root/'credentials.json').chmod(0o600)
PYSETUP
go run ./cmd/picketx-controller -config examples/controller/controller.yaml
```

The raw Token remains only in the private `var/agent.token` provisioning file; distribute it securely to the intended Agent and do not log it. The Controller credential file contains a complete set of SHA-256 records. Update files by writing a sibling temporary file then atomically renaming it; retain0600 credentials/private0700 DB parent. Deleting all records requires explicit `{"credentials":[]}`; missing/null/duplicate/unknown keys are rejected. Changing/deleting/rebinding a record closes corresponding live streams. A credential file error retains the prior accepted set and makes readiness fail; never describe an invalid attempted edit as successful revocation.

Default no-config startup still exposes only HTTP health/readiness on127.0.0.1:8080, with Agent service disabled. With the example, the Agent listener is127.0.0.1:9090. `healthz` is process liveness, `readyz` is generic configured-module readiness and contains no policy/credential details. Neither proves remote enforcement. There are no administrative/business endpoints in this slice. The generated client protocol and Agent assertions are described in [Agent stream](AGENT_STREAM.md).

## Configuration contract

Koanf2.3.8 loads **explicit changed flags > present PICKETX_* environment > one strict YAML file > defaults**. `-config` explicitly selects the file; otherwise PICKETX_CONFIG does. `-listen` overrides http.listen; `-agent-listen` overrides agent.listen. Unspecified flags do not replace lower layers; an explicitly empty value is validated rather than ignored. Unknown/null/duplicate/aliased/multi-document settings and malformed types fail before listening. Relative paths are relative to the process working directory.

| YAML key | Default |
| --- | --- |
| http.listen | 127.0.0.1:8080 |
| agent.enabled / agent.listen | false / 127.0.0.1:9090 |
| agent.authority | Required when enabled; must match source facts and DB |
| source.policy / source.facts | Required versioned full policy YAML / exact facts JSON |
| database / credentials | Required private SQLite path / complete credential JSON path |
| poll / shutdown | 1s / 5s, positive durations |
| agent.max_credentials / agent.max_streams | 128 /128, at most4096 |
| agent.max_frame | 4194304, positive and at most4MiB including frame |
| agent.establish_per_second / agent.messages_per_second | 128 /32, positive process/stream-local windows |
| agent.hello_timeout / agent.send_timeout | 5s /10s, positive durations |
| agent.tls_cert / agent.tls_key | Both absent (plaintext) or both configured |

Environment names uppercase the key and replace dots with underscores, e.g. PICKETX_AGENT_ENABLED=true or PICKETX_DATABASE=/private/state/controller.sqlite. Unknown PICKETX_* names fail. TLS loads a standard certificate/key at startup, minimum TLS1.2; clients use normal root/name verification. No mTLS/internal CA is required. Plaintext has no confidentiality or server authentication: expose it only deliberately on a controlled or tunneled hop. TLS material and runtime config changes require restart; policy/facts/credentials are polled.

## Durability, reload and failure

The selected GORM1.31.2/official SQLite1.6.0/mattn1.14.52 adapter runs an explicit version1 SQL migration. No AutoMigrate, PostgreSQL integration or HA coherence is implied. SQLite uses one connection, DELETE journaling, synchronous FULL and a5s busy bound. The database is a private regular file in a private directory. It stores one complete accepted policy/facts envelope, fixed authority, SHA-256 integrity digest and positive signed64-bit revision. Unknown newer schema, wrong authority and corrupt accepted state fail startup rather than overwrite it.

Compile/Replace and stream preflight precede commit; the committed record precedes distribution. Semantically unchanged input does not consume a revision. Retained identities cannot bypass retargeting restrictions. A concurrent stale database writer is rejected by revision CAS; process memory is not a distributed cache. Credentials and publication are serialized by the runtime owner. Unexpected persistence or postcommit distribution failure stops the process; restart recovers the accepted durable revision.

Without an accepted record, initial source input must validate. With a record, bad current policy/facts leaves prior complete state available over gRPC and readiness503 until corrected. Independent absolute deadlines remain in the snapshot; the Agent must enforce them locally during outages. Credential reload runs independently first, so valid revocation still works during a rejected policy reload. Invalid source/credential edits never partially merge. Readiness200 does not mean any Agent is connected or has applied a revision; APPLIED reports remain authenticated remote assertions.

The process reserves/rolls back listeners centrally, cancels polling before closing the stream service/database, then bounds HTTP/gRPC graceful shutdown and forces remaining connections closed. Input compilation and the SQLite busy bound are separate from the network shutdown timeout. SIGTERM keeps durable accepted state for restart.

## Static Linux artifact evidence

Run `scripts/check-static-controller.sh`. It builds with CGO_ENABLED=1 plus netgo/osusergo/sqlite_omit_load_extension and full external static linking, rejects an ELF interpreter, then actually starts that binary with SQLite, HTTP and Agent endpoints. A real generated gRPC client verifies its scoped revision, sends RECEIVED (not an invented APPLIED), the script checks DB migration/revision, health/readiness and SIGTERM exit. Generation tools and extra shared libraries are not runtime requirements of this artifact. `scripts/check-static-cli.sh` retains the existing CLI deadline/withdrawal checks. CI runs both.
