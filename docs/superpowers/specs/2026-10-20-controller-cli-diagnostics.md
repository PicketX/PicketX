# Controller CLI diagnostics design

Purpose: ship read-only status/doctor for existing Controller APIs. User authorizes
autonomous focused design/plan/inline execution and verified integration; routine
handoff waits waived, TDD and one independent review retained. Architectural client
boundary, not new product authorization. Alternatives: shared client recommended;
duplicated command HTTP logic risks drift; direct DB/Agent implementation access
violates process/licensing boundaries. Existing clean linked worktree/new branch.

Entry Coree899b9aaf313a6a364a22f73cf450d8ccffe7628/tree3764b521c72c51f086df5dd5f0b4ad52ddee6fa7,
CI39394733598; Agentb14b54d/tree5b243935/CI39388341832. No openPR; PR42 released08:45:54Z.

## Contract
Cobra status and doctor require --controller origin and --key-file absolute private
regular non-symlink file. Optional --agent exact credentialed scope; --format text|json
(defaulttext); --timeout default5s positive<=30s; --ca-file adds normal HTTPS trust.
No credential value argument, environment discovery or writes. HTTP literal loopback
IPv4/IPv6 only; other origins HTTPS, normal CA/SAN; no userinfo/query/fragment/nonroot
path, redirects, ambient proxy or verification bypass. Deadline covers all reads;
bounded bodies, duplicate/trailing/unknown/malformed JSON rejected, report:null allowed.
Fixed errors never expose response body/URL/key/path/underlying exception.

Client GETs publication, observations, publication; reject changed revision/authority
or node desired revision inconsistent with publication. Doctor additionally reads
/readyz WITHOUT Bearer. Output only scope=controller-observations, authority, decimal
revision, observationtime, readiness(ifdoctor), bounded node state and remainingTTL.
Never output policy/facts/secrets; enforcement_confirmed=false always.
reported-applied requires connected+fresh+remainingTTL>0+matches_desired+APPLIED and
exact desired/report revision+hash agreement. Deduct local elapsed time since the
observations request from serverTTL. Other states unbound/unknown/disconnected/
stale/applying/failed. Agent observed_at is not the freshness clock.
Status returns0 for valid reads including empty/disconnected. Doctor returns0 only
ready200 and nonempty selected scopes all reported-applied; otherwise truthful report
plus nonzero. Missing selected scope is an error. No atomic/continuous/kernel proof.

## Context Map
| File | Responsibility |
| --- | --- |
| internal/controllerclient/client.go (new) | Verified bounded HTTP, origin/private credential/CA |
| internal/controllerclient/status.go (new) | DTO validation and coherent point-in-time state |
| internal/controllerclient/*_test.go | Real HTTP/TLS, response/transport/freshness regressions |
| internal/cli/status.go (new), config.go | Thin Cobra status/doctor adapters and renderers |
| internal/cli/status_test.go (new) | Command contract, error/exit/credential exclusion |
| internal/controllerapp/observations_test.go | Existing actual Controller/gRPC through CLI |
| apps/web/e2e/install.mjs | Shipped empty-PATH CLI status/doctor/auth/stopped-server checks |
| docs/CLI_DIAGNOSTICS*.md, README*.md | Operators, freshness/exit/trust limits |
| deploy/controller/README*.md | Ship the same diagnostic instructions in the bundle |
| docs/AUTONOMOUS_DELIVERY*.md | Short lease, exact final receipt/next work |
Dependencies: publication.Document and agentsync.Observations wire fields,
managementaccess publication:read; nginx_init private-file pattern; runtime
observations test and existing installed runner. No API/schema/dependency change.
No GUI/kernel/Agent local socket/repair/reload/enrollment/new config authority.
Retain original real-provider/recovery/browser/fourartifact gates. Map/spec
self-reviewed: no placeholders/contradictions, truthful assertion boundary.

