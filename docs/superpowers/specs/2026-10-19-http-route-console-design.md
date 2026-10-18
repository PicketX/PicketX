# Structured HTTPRoute management

Create and inspect HTTPRoutes from the operator console without editing a manifest.
Reuse the current SQLite-authoritative publication, CAS, current RBAC, CSRF,
transactional audit and complete configuration compiler. No second authority,
schema migration, forwarding engine or new dependency.

POST /api/v1/http-routes (and the cookie namespace) accepts a name, pinned
resource name/UID, TCP endpoint name, exact host, explicit methods and exact paths,
plus an audit reason and strong If-Match. The server generates the route UID.
GET inventory returns detached route entries. Invalid selectors, duplicate names,
stale revisions and unauthorized calls cannot modify policy, facts or audit.

The separate HTTP routes page retains the existing console visual language:
catalog, exact-target form, explicit revision, error summary, loading/empty/error/
stale states and read-only mode. Refresh never silently rebases a dirty draft;
412 or ambiguous mutation requires explicit rebase. Hidden drafts block logout.
English and Chinese labels remain equivalent. Forms remain usable at375px.

Creation is desired configuration, not a grant or confirmed ingress setup. A
trusted caller must separately allow the generated route UID and the existing
ingress must pin that UID/Host. Source Grants remain independent; no wildcard,
prefix, identity, TLS termination, proxy configuration or automatic credentials.

Acceptance: real SQLite API persistence/CAS/audit/RBAC/invalid-scope tests;
real Chromium creation and conflict/read-only/error paths; generated UID actually
used by trusted Nginx auth_request and the application with matching source Grant,
denial and revocation. Full Go/race/vet/build, locked Web and native Linux gates.
One fresh whole-branch review before expected-head merge and fresh main CI.

Self-review: existing selectors and pinned TCP targets cover the scope; no new
runtime authority or caller scope is accepted in this DTO. Routine design/plan
approval is already explicitly waived by the user. Existing lifecycle editing,
enrollment, identity/approval and live desired/applied status are separate work.

## Console visual/interaction rules

Retain repository tokens: background#f5f7fa, foreground#18283b, primary#145da0,
muted#526174, surface#ffffff, error#a52222; system sans16px/1.5 with code only
for opaque IDs/paths. Left-aligned section heading and revision observation,
single create form grouped by target then selectors, followed by inspectable
desired routes. Two-column target selectors collapse to one column at800px.
No marketing hero, new font, animation or decorative metrics. The distinctive
element is the exact UID/target/request contract and its explicit setup boundary.
frontend-design brief review favors console continuity over a new visual identity.
ui-ux-pro-max returned a marketing landing pattern twice, so no verified matching
page pattern is adopted. Verified web error-summary guidance and React effect
cleanup are applied; layout/style use repository rules, not unverified output.
