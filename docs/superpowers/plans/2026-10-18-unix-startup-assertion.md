# Unix authorization readiness assertion repair

Scope: test-only follow-up to PR18 main CI39141222771, Go117715886276.
The real Unix case observed mode0755 at authz_test.go:92. Dependencies.Listen
publishes the address before returning to listenAuthorization, which then sets
mode0600 before Run starts HTTP Serve. Address discovery is not service readiness.
The private parent remains0700; no production permission change is needed.

## Context map and design

| File | Role | Change |
|---|---|---|
| internal/controllerapp/authz_test.go | Real TCP/IPv6/Unix acceptance | Check0600 after a real401 response proves Serve has started |
| internal/controllerapp/authz.go | Private directory, chmod, owned cleanup | Read only; preserve0600 enforcement |
| internal/controllerapp/runtime.go | Listen→configure→Serve | Read only; confirms startup ordering |
| .github/workflows/ci.yml | Native Linux acceptance | Repeat the actual Unix case100 times under race detector |
| docs/AUTONOMOUS_DELIVERY{,.zh-CN}.md | Handoff | Record main failure and repair ownership |

Root reviewed the map against implementation. No public API, database,
configuration, UI, Agent or kernel change. Existing tests retain authenticated
decision, denied unauthenticated request, listener isolation and shutdown cleanup.
Removing chmod or using a non0600 mode must still fail the permission assertion.
Do not weaken the assertion, poll until it passes, or skip Unix.

## Execution and verification

1. Use the existing isolated host worktree; preserve its branch and other work.
2. Move only the permission assertion after the first actual HTTP response.
   Main raw failure above is RED evidence; local Unix socket creation is blocked
   by EPERM, so GREEN must come from actual native Linux CI.
3. Run complete ordinary/race/vet/build and locked Web checks, honestly recording
   the local Unix restriction. Require full native CI and a repeated Unix race
   stress run, plus an independent whole-branch review of this repair.
4. Match exact local/remote/CI source trees, merge using expected head, verify
   complete main CI; append final acceptance to the repair PR and PR18.

Ruling: synchronize on a real HTTP response instead of listener discovery;
cost: the mode assertion follows an unauthenticated request, while the private
directory and production chmod-before-Serve ordering remain unchanged.

Resource/Binding onboarding is implemented, but PR18 main acceptance is not
green until this follow-up lands and passes. No whole-product completion.
