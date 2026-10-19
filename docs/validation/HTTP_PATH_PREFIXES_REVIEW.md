# HTTP path-prefix delivery review

## Scope and independent review

Core PR34; base31013cd0d97a40dad26af631a5505039413ff19e through
3300de2182627b667d49e3c900da0dbae8f139c9. One fresh read-only whole-branch review
by `http_prefix_review` (gpt-6-astra/high), using requesting-code-review template,
spec/plan, full diff and ledger. No implementer agents, per-task reviewers,
re-review, fabricated GitHub approval or skipped security gate.

Verdict: no Critical or Important defect; ready conditional on exact-final-source
native evidence. Reviewer independently passed focused normal and race HTTP
prefix/route tests plus whole-range diff whitespace checks. It specifically
confirmed segment/canonical boundaries, exact compatibility, strict presence,
cloning, durable publication, conjunctive source/deny/time/caller/target scope and
separate real UI selectors. Current native evidence is independently gated by the
implementer, not inferred from that review.

## Deferred minors

1. Reviewer: `apps/web/src/locales/en.json:43` selectorLimit wording "unique
   selectors across" can imply cross-list uniqueness. Contract and implementation
   permit the same string in exact and prefix lists; Chinese correctly says each
   list. Cost: potentially confusing advice, no authorization widening/rejection.
2. Visual inspection: after changing language, already-rendered failure/success
   messages remain in their former language because existing console state stores
   translated strings. New labels, helpers, prefix details and root warnings do
   translate. This pre-existing behavior was visible in the intentional failed-read
   Chinese screenshots. Cost: mixed-language retained feedback, not stale policy
   acceptance or a loss of drafts/CAS. Deferred, not silently called fully localized.

The independent review needed no blocking code fix; final native CI subsequently
exposed the existing credential test robustness defect below. It enters the one
final TDD fix pass, not another review dispatch. Minors were not opportunistically fixed.

## Final native CI-found test defect

Final run39238841570 Go118014445917 failed the unchanged
TestNewCredentialReceivesPrepublishedNonemptyInventory at runtime_test.go241:
EOF from Hello Send. Web118014445576 passed, but later Go build/Unix/static steps
were skipped by the failed mandatory suite; this run is NOT accepted or merged.
Root cause: its purported admission retry loop called the success-only stream
helper, which fatals on Send before Recv can expose the legitimate Unauthenticated
status while credential polling is pending. Production rejection is correct.

The fix is test-only: each bounded attempt owns/closes its real connection, obtains
status from Recv after Send EOF, retries only Unauthenticated, immediately fails
other errors, explicitly proves rejection before registration, and still requires
the exact agent-b/revision1/two-target nonempty snapshot after polling. Local race
100 repetitions pass after the observed native RED. A pre-fix local100 run happened
to pass; it is not presented as a deterministic reproduction. Add the same100
real native repetitions as a mandatory CI step; no skip, sleep-increase, weaker
server authentication or fabricated snapshot is used. Full final-source/main
Go/Web and raw/archive provenance gates must run again after this fix.

## Decisions and costs (complete ledger rulings)

- Explicit empty lists reject even when another list is populated: preserves
  strict omission semantics. Cost: clients must omit unused lists rather than [].
- Prior autonomous authorization selects inline execution and one final review,
  without routine design/execution confirmation pauses. Cost: design is reviewed
  through persisted spec/plan instead of an additional interactive gate.
- Review declined actual Gitea HTTPS/SSH and release behavior. Ruling: keep them
  separate pending acceptance, as the authoritative spec explicitly requires.
  Cost: this delivery cannot establish Gitea/deployment readiness; next increment
  must supply that actual application/transport evidence.

## Tests-first evidence

- Configuration TestHTTPPathPrefixes RED: unknown pathPrefixes; actual Controller
  prefix-only durable creation RED:400 invalid_json. New engine/config/API then
  pass boundary, root, exact union, IPv4/IPv6 source, Host/method, deny/independent
  expiry, invalid canonical selectors and snapshot clone checks.
- TestHTTPPathPrefixesAddAndClone separately RED on explicit empty exact list
  erased by omitempty; pre-serialization rejection makes it GREEN. Read-only
  inventory null-array regression separately RED then GREEN with paths:[]; policy
  authoring omission remains distinct.
- Native39237338173/Web118009489186 RED at2026-10-19T10:28:15Z: missing prefix input
  in old React at http-prefixes.mjs24, after existing exact-route/removal suites.
  UI implementation followed that observed failure. That run's Go118009489549
  passed244 top-level/596 total tests, zero fail/skip/race, Unix100, vet/build,
  pinned protobuf/OpenAPI and real static SQLite Controller/CLI checks.
- Native code run39237922431 Web118011427553 is GREEN: real prefix-only creation,
  separate persisted selectors, nested paths and query allow, lookalike/canonical/
  Host/method/source denial with no app hit, Grant revoke over reused connection,
  retained prefix draft after failed observation and EN/ZH375/1440 screenshots.
  Existing browser workflows remain mandatory; HTTP allows do not prove Agent apply.
- Same code run Go118011427841 is GREEN:244 top-level/596 total, zero fail/skip/race;
  every mandatory step including actual Unix100, static Controller/SQLite/gRPC/DB/
  SIGTERM and static CLI absolute-deadline examples passed. Raw Go and Web logs
  both identify actual checkout40ec08a4560d37f53e1e8df0d49e9f6c45d0801a.
- Local complete normal/race runs retain only known unchanged
  TestAuthorizationRuntimeTCPUnixAndShutdown/unix socket EPERM. Full relevant
  package normal/race, vet/build, pinned API generation/schema validation and
  locked npm ci/typecheck/lint/build pass. Native full Unix tests are not skipped.

## Code evidence and integration boundary

Code local975a3133f5a846042eaa2352ccedbcaf8107c658, remote26282e141b8091d59b957384d1fba4b2a120cb68,
actual native checkout40ec08a4560d37f53e1e8df0d49e9f6c45d0801a share tree
b8ae773f9f56972926fb940aecfd7c2d9e230217. Screenshot artifact12179963297 independently
downloaded: ZIP SHA256438814e154716c6ccd3d40952048a3a4f0d22f3f62466b4d0fa3acb72d0dfe75
matches metadata/raw upload log. EN root warning and saved prefix inventory plus
Chinese failed-read/root warning inspected at375/1440: no overflow, separate
selector semantics visible, prior-language feedback caveat explicitly above.

Final documentation-only head still requires exact-source full native Go/Web CI,
raw logs and independent archive digest verification, expected-head merge, then
main tree/parents/CI recheck. Those immutable final IDs and immediate lease release
belong in PR34 final metadata, avoiding a self-referential documentation commit.
No Agent/kernel/protocol/SQL migration/dependency changes. Actual Gitea HTTPS/SSH
and deployment/release acceptance remain pending; overall goal is not complete.
