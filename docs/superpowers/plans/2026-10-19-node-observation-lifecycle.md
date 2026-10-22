# Node observation lifecycle implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans.

**Goal:** Close PR21's two deferred browser lifecycle items.
**Architecture:** Keep the existing Nodes effect and at most one active request.
Coalesce visibility resumes during a read into one follow-up after settlement,
only while visible and mounted. Polls never queue; unauthorized reads stop the
queued refresh. Cleanup aborts and discards pending work.
**Tech Stack:** Existing React/TypeScript/i18next/npm, Playwright and real Controller.
**Spec:** docs/superpowers/specs/2026-10-19-node-observations-design.md

## Global constraints and UI flow
Retain 15-second polling, 60-second server freshness and conservative request age.
No new API, domain, schema, protocol, Agent or kernel behavior. Preserve accepted
layout, text and all loading/error/history states. UI/UX guidance applies to effect
cleanup and stable feedback; frontend-design preserves the existing visual system.
Autonomous authorization covers the routine plan, commits, PR and gated merge.

## Context map
| File | Responsibility |
| --- | --- |
| web/src/features/nodes/Nodes.tsx | Effect-owned active request and queued resume |
| web/e2e/node-lifecycle.mjs | Real browser delayed reads, visibility and disposal |
| web/e2e/nodes.mjs | Invoke lifecycle acceptance with existing signed-in session |
| docs/AUTONOMOUS_DELIVERY.md | Lease and exact evidence/handoff |

Dependencies: request() timeout/AbortSignal, App conditional Nodes mount, existing
Controller credential scope API. No change to freshness.ts or component contracts.

## Review focus
1. Repeated resumes during a request coalesce; no overlapping reads.
2. Hidden again before settlement: no queued network read while hidden.
3. Unmount with queued work: abort and no stale UI/error or follow-up after return.
4. Failed read with queued resume: retry once, recover honest error/history state.
5. Unauthorized response: logout without queued authenticated read.

### Task 1: Browser regression and minimal fix
**Interfaces:** existing Nodes({onUnauthorized}) and
nodeLifecycle({operator}); no new production interface.
- [ ] Add real browser tests using gated actual API responses; deterministic
  visibility injection and browser clock are explicit test controls, not real OS
  backgrounding or fabricated Agent execution. Assert immediate second request
  before advancing the poll clock, coalescing, hidden pause, disposal, failure/401.
- [ ] Run native `npm run test:e2e` on unchanged production code. Expected RED:
  queued-resume request does not arrive within bounded wait.
- [ ] Add effect-local pending flag; consume on new read and drain on settlement
  only if visible/mounted. Clear on 401/disposal. Expected full browser PASS.
- [ ] Run locked npm ci/typecheck/lint/build; full Go test/race/vet/build. Local
  Unix EPERM remains an explicit environment limitation; native CI must not skip.
- [ ] Commit focused fix and evidence; independent whole-branch review, correct
  blockers RED→GREEN; exact-source CI/expected-head merge/main CI and release.

## Ledger
Pre-flight: one task, no shared interfaces. Spec coverage/self-review complete.
Use existing verified isolated host worktree; do not switch or reset shared state.
Deferred product lifecycle/identity/approval and combined Linux acceptance remain.

Outcome: test-only RED CI39155267309 at immediate nextRead; GREEN browser
CI39155687013 covers all five focus cases on tree5a285a28b124ec7780c73fbca859882edcecc515.
Independent b21585a..e33bb9e whole-branch review: no findings. Final native Go/static,
exact final head and merged-main evidence/lease release are recorded in PR22.
No new deferred minor. See delivery checkpoint for environment and scope limits.
