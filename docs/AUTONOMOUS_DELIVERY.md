# Autonomous delivery checkpoint

Updated: 2026-10-18T08:36:00Z. User explicitly authorized autonomous development, tests, push, PRs and merges until a usable management application and basic Layer 7 authorization exist. Routine design/plan and integration decisions do not require another approval round. Preserve existing security boundaries and required checks.

## Current lease
Owner: interactive Controller runtime increment. Branch: `codex/controller-runtime`. Lease expires: 2026-10-18T10:00:00Z (Asia/Shanghai 18:00). Do not start competing work while this lease is active; inspect PR12/CI instead. This lease automatically releases when PR12 is merged and its main CI succeeds, even before the timestamp. Renew with each durable checkpoint; stale work may be resumed from actual branch/PR state.

## Accepted baseline
- Core PR11/main793ac15faadd6565c64c713959c59990c2bf3a1b, tree f6f98df86d7e0e69c0ba7ac41cf563c4d865572a. Main CI39109301425 passed94 top-level/311 total cases, real gRPC/TCP/TLS. Embeddable stream service only.
- Agent PR15/main1949d6352da6afb0bf6796d97e45eabe6d197ee8. Static host L4, independent deadlines, scoped strict cleanup, durable restart and systemd readiness accepted; Controller provider pending.

## Work sequence and acceptance
1. Running Controller: strict Koanf configuration, SQLite durable publication, credential reload, optional standard server TLS, bounded HTTP/gRPC lifecycle and static Linux artifact smoke. Implemented in PR12 and locally validated; independent review/PR CI/main CI determine acceptance. After PR12 is merged with green main CI, proceed to step2 without repeating this implementation.
2. PicketXD Controller Provider: exactly one authoritative provider, scoped validation, independent deadlines during outage, provider-bound remote LKG and actual native apply reports. Reuse the typed protocol without importing GPL Agent code into Core.
3. Management APIs/persistence/login/RBAC and basic HTTP request authorization through a trusted existing ingress plus scoped local decisions. Keep approval, desired state and applied assertions distinct.
4. Real React administration pages (resources/bindings, Agent freshness, grant issue/revoke, authorization rules and records) against live APIs; basic portal/approval workflows as documented. Use the selected shadcn/Tailwind/i18next stack, UX/UI skills and explicit loading/error/empty/stale states.
5. Actual browser/API/proxy/application walkthrough and Linux enforcement evidence. At completion pause the hourly task and report precise acceptance evidence.

Next tests also include PR11 deferred minor: newly credentialed scope receives existing nonempty desired inventory without republishing. Every increment needs uncached relevant full tests, fresh whole-branch review, checked remote source tree/head, passing required PR CI, expected-head merge and main CI verification. No forced pushes or bypassed checks; no fixture/UI-only completion claims. The hourly task must attempt real implementation when an execution environment is available, and report an actual blocker when it is not.

Deferred PR12 review minor: the existing CAS test covers initial INSERT collision; add stale UPDATE regression by seeding revision1, opening two stores at revision1, committing revision2 through one, then checking a different valid stale change fails without replacing revision2. Implementation retains the correct revision predicate.

## 2026-10-18 Provider handoff
Core PR12 is accepted: main5e2254efec90f2a5448450a2974c587c37f1dd18,
tree d78219ef42fb515e73c0144e40e7ea85a3896cc8, main CI39113728426 success.
The PR11 newly credentialed nonempty-target regression was completed in PR12.

Active next implementation is Agent PR16 (`codex/controller-provider`), owner automated
Provider increment, lease until 2026-10-18T12:30:00Z or merged/green main CI.
Inspect that PR and Agent docs/AUTONOMOUS_DELIVERY.md before starting new development.
This Core branch only supplements the deferred stale UPDATE CAS regression: seed revision1,
open two stores at revision1, persist a valid revision2 and reject the stale distinct update.
The new test caught deliberate removal of the revision predicate; production code is unchanged.
Local complete race115 top-level/344 total tests, vet/build and Web lockfile checks passed.
