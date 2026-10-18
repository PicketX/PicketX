# Management mutation completion boundary

The existing console must not treat HTTP 202 (accepted), 204, or another
unexpected 2xx as proof that a management write completed. Preserve draft
intent, display the existing accessible error/reconciliation feedback, and
block retry until explicit refresh/rebase. Approval/publication remains distinct
from actual node application. This is a bounded correction, not a new API.

## Contract

- `request<T>(path, init, expectedStatus?)` checks an explicitly supplied success
  status before decoding or returning the response. Non-2xx retains APIError
  semantics; an unexpected 2xx is an ambiguous result (ordinary Error).
- Resource, Binding, HTTPRoute and Grant creation require 201. Removal,
  revocation and publication require 200. GET and session operations are not
  changed in this slice. Existing UID/revision/ETag removal checks remain.
- No automatic retry, no draft clearance or success notice on unexpected status.
  Publication must also block retry after transport/unknown/server failure.
- Native browser fault injections prove these negative paths; each continues
  through the real API afterward. Mock responses are not execution evidence.
- No dependency, schema, auth, kernel, license or deployment change.

## Context map (reviewed before implementation)

| File | Responsibility / change |
| --- | --- |
| apps/web/src/lib/console.ts | Shared transport, optional exact success status |
| apps/web/src/App.tsx | Publication 200 and ambiguous-write retry lock |
| apps/web/src/locales/{en,zh-CN}.json | Publication reconciliation guidance |
| apps/web/src/features/{resources,grants,http-routes}/*.tsx | Explicit status at each management mutation |
| apps/web/e2e/request.mjs | Real helper status/error/ETag/204 regression |
| apps/web/e2e/{console,resources,resource-removal,http-routes}.mjs | Actual form intent and recovery with response fault injection |
| apps/web/e2e/mutation-receipts.mjs | Shared narrow browser fault assertion |

Dependencies: API status contracts in internal/controller/{management,inventory,
grants,http_routes,http_route_removal}.go and existing removal receipt validation.
Reference patterns: HTTPRoute removal retains intent and blocks after ambiguous
response; Resource/Grant failWrite already distinguish ordinary from API errors.
Risks reviewed: no public API change, DB migration or configuration requirement.

## UX direction

Retain existing typography, semantic palette, layout and loading states; this is
interaction correctness, not a visual redesign. Use existing focused role=alert
and explicit rebase guidance. No new toast that suggests authorization is applied.
UI/UX search “form error recovery” supports a visible recovery path, announced
errors and truthful submission feedback; frontend-design supports consistent
action/result vocabulary. Preserve the operator's inputs until reconciliation.

## Acceptance

Test exact 200/201 versus unexpected 202/204/other 2xx with valid-looking bodies,
non-2xx API errors, unchanged GET behavior, and ordinary errors reaching existing
conflict paths. Real browser checks every affected form, preserved reason/name,
no success, disabled repeat, explicit rebase, actual successful write afterward.
Full Go/native Web CI, independent branch review, exact-head merge and main CI
remain gates. Resource editing, identity/approval and combined Linux/browser
acceptance remain outstanding; this increment does not finish the whole goal.
