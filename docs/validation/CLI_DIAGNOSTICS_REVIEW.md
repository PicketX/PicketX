# Controller CLI diagnostics review and rulings

ONE fresh independent read-only whole-branch review of e899b9a..5984e79:
0 Critical, 2 Important, no separate Minor; verdict With fixes. No second review
or GitHub approval is fabricated. Root verified both findings against actual code.

1. Cobra parsing before RunE echoed malformed duration/unknown flag/positional
   arguments. Six command regressions observed RED, then GREEN with fixed flag
   errors and non-echoing argument validation. Installed detached/systemd runners
   additionally exercise the actual binary's stderr, not just a returned error.
2. The 16 MiB publication response cap rejected valid 4 MiB decoded inputs after
   HTML escaping. An actual managed Controller accepted a non-HTML-escaped request
   with a 3 MiB `<` YAML comment and returned >16 MiB JSON; diagnostics observed RED.
   Both decoded oversize-field regressions also observed RED. The client now derives
   its encoded cap from two configuration.MaxBytes fields, sixfold worst-case JSON
   expansion and a 4096-byte envelope, and checks each decoded field on both reads.
   An oversized streamed response still fails without output. No producer limit
   or authentication/readiness/freshness requirement was weakened.

Root ONE regression-first correction pass; final corrected complete native CI/raw/
four actual archives/source/static/notices/images/expected-head/main are mandatory.
Candidate39403452137 passed all3jobs, Go269top/701total, full raw and four actual
archives/source tree00b41005. It is NOT evidence for the corrected source.

## Every declined-to-judge ruling and cost if wrong

| Considered boundary | Root ruling | Cost if wrong |
| --- | --- | --- |
| Atomic/continuous health, changes after samples or blocked stdout | Point-in-time observation only; no repair, lease or ongoing guarantee | Misleading health after state changes |
| Hostile parent replacement/blocking filesystem and CA reads | Trusted parent ownership is required; HTTP deadline starts after bounded regular-file initialization, not a hostile-filesystem deadline | Credential race or initialization delay |
| Discarded Agent observed_at syntax/control characters | Do not broaden to a full producer-schema audit; string is never rendered or used as clock; exact consumed identifiers/state remain validated | An invalid discarded timestamp may be accepted |
| Producer timestamp precision exceeding128bytes | Retain this explicit diagnostic compatibility bound; normal Agent RFC3339Nano output is shorter; no arbitrary fractional precision support claimed | A pathological otherwise parseable report is rejected |
| Independent kernel/packets and deployed HTTPS interoperability | CLI cannot certify either; actual protocol TLS tests and separate retained Linux/application evidence have distinct boundaries | Overstated enforcement/topology support |
| Native exact source/environment/full suites/archives | Root owns raw complete final CI and actual archives; local Unix EPERM and isolated slow-stream success do not waive it | Stale or untested release |
| Exhaustive proxy-env/malformed-PEM/large-fleet stress | Inspected explicit Proxy=nil/normal CA rejection/bounded inventory; broader matrix deferred, no capacity certification | Unexercised edge path or scale limit |

Other explicit decisions: shared read-only northbound client/thin Cobra preserves
auth and Apache/GPL process boundaries (wrong: dependency drift); tests use the real
client through CLI and actual HTTP/TLS rather than duplicate private mocks (wrong:
consumer/client paths missed); publication before/after plus local elapsed TTL is
not atomic (wrong: misleading freshness); APPLIED is reported-applied and always
enforcement_confirmed=false, empty scope cannot pass doctor (wrong: false proof).
Existing autonomous authority waives routine design/plan/push/verified-merge waits,
not evidence gates (wrong: unreviewed release); isolated workspace/expected_sha and
no force preserve user work (wrong: data loss). Only one exact checksum-OK GitHub
annotation on pgx notice_response.go is not a failure: every step and actual full
manifest checksum is checked (wrong: hidden artifact failure).

No UI/Agent/kernel/DB/API/schema/dependency change in this slice. Full-product audit
and final immutable receipts/release belong to PR43, superseding pending notes.
