# Controller CLI diagnostics

The shipped `picketx status` and `picketx doctor` read the Controller's existing
management API. Use a management key authorized for publication:read (a viewer
key is sufficient). Agent/ingress Tokens and browser cookies are not management keys.

```sh
picketx status --controller https://controller.example.test --key-file /private/management-key
picketx doctor --controller https://controller.example.test --key-file /private/management-key --agent node-a --format json
```

Examples contain paths only, never put the key value in command arguments. The key
file must be absolute, regular, non-symlink and private0600 (or stricter), with32–256
printable nonspace ASCII bytes and optional one final LF. Use an independent random
key; controller-init's bootstrap-key is accepted. Parent ownership must be trusted.
No credential env lookup, DB access or config-file authority is introduced.

Origin is HTTPS with normal certificate/hostname verification; optional --ca-file
adds a trusted PEM CA. HTTP accepts only literal127.0.0.1/other loopback IPv4 or[::1].
No URL userinfo/path/query/fragment, redirects, ambient proxies or TLS bypass.
Total --timeout defaults5s, positive<=30s; no automatic retries. Read failures,
authentication denial, inconsistent/malformed/oversized data produce fixed errors
and no success report. policy/facts/raw errors, key values and private paths are
not emitted. Only fixed API GETs carry Bearer; readiness probes do not.
Malformed flags and positional arguments also use fixed non-echoing errors.
Publication reads allow worst-case JSON escaping of both supported4 MiB input
fields, then enforce each decoded field limit; response reads remain bounded.

Status returns0 for a valid current read, including empty/disconnected inventory.
Doctor additionally checks readiness and returns0 only for at least one selected
credentialed scope, all currently reporting matching APPLIED. Otherwise it prints
the findings and returns1. Operational/read errors return1 for either command.
--agent selects an exact currently credentialed scope; missing scope fails.
--format text is the default; json supports automation.

| Node state | Meaning |
| --- | --- |
| reported-applied | Current connected stream, fresh matching APPLIED assertion |
| applying | Desired version/hash is not yet reported APPLIED |
| failed | A fresh FAILED assertion |
| stale | Report expired or belongs to an earlier stream |
| disconnected | No current Controller stream |
| unknown | Connected scope without a report |
| unbound | No published target snapshot for this credentialed scope |

The CLI reads publication/observations/publication and refuses changed authority/
revision or inconsistent desired versions. It subtracts local read elapsed time
from remaining freshness; Agent-supplied timestamps cannot extend it. Successful
reads do not refresh report age. These separate API samples are not an atomic lease.
JSON always says scope=controller-observations and enforcement_confirmed=false.
Agent APPLIED is a point-in-time assertion, not independent kernel verification,
a packet probe, continuous health or confirmation of a particular revoked flow.
Empty inventory is not successful deployment. Doctor does not repair/reload state.

Existing static `config check` and deployment commands retain their semantics.
Actual native integration exercises the shipped CLI with empty PATH against the
installed Controller: initial/restarted disconnected states, correct revisions,
wrong-audience denial and stopped-server failure. Actual runtime/gRPC tests also
cover fresh APPLIED, pending revision, disconnect and stale reconnect. Native
archives include cli-diagnostics-report.json; exact source/CI/main acceptance and
review limitations are in the delivery PR, not inferred from this document.

