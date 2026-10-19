# Database-authoritative human account lifecycle

## Intent
An operator can manage real local human accounts through the existing console,
without editing a credential file for each account change. Preserve existing
file users, key login, SCS/Casbin and separate network subjects. This is a focused
local-account increment, not OIDC, registration, mail reset, MFA or HA.

## Authority and recovery
`management.users_mode` defaults to `file`; opt-in `database` requires database
publication mode and a private key credential document with no users and at least
one unblocked role:admin key. Keys remain independently file-authoritative bootstrap
administration; database user IDs reserve `principal:account-` plus random128 bits,
and key IDs in that range are rejected in database mode. No automatic user import.
Persist a versioned database-user authority marker. Once initialized, file mode
fails startup/readiness/authentication rather than falling back; any file user in
database mode fails closed. File polling cannot overwrite database users.

Store at most128 accounts, unique immutable lowercase usernames using existing
3–64 ASCII format. IDs/username are never renamed/deleted/recycled by this API;
disable is the initial removal workflow. Roles1–4 are a nonempty unique subset of
role:admin/viewer/requester/blocked; custom file/Casbin policy remains supported
unchanged, but directory editing of arbitrary groups is outside this increment.
Only account:read/account:write operations authorize listing/mutation; admin gets
both, blocked denies both, no default viewer/requester directory access.

## Persistence and session safety
Versioned account tables reuse injected official GORM/SQLite connection. One global
directory revision provides strong If-Match for create/update, independent of
publication revisions. One transaction performs revision CAS, account change and
append-only account audit (actor, account ID, operation, reason, revision, time;
never passwords/hashes). SQL secret-bearing operations use silent scoped logging.
Validate current rows fail-closed; no successful read after a failed transaction.
Passwords use existing bcrypt12/12–72 UTF8-byte rules and share four expensive-work
slots with login. API/UI never return hashes. Every update increments persistent
account version, including disable/re-enable and role-only updates. Database-human
session fingerprints bind version as well as UID/username/hash; every request
resolves current rows, so old cookies cannot resurrect after re-enable or restart.
An in-flight operation already authorized is not retroactively undone.

## API and UI
GET /accounts returns mode, revision, accounts and latest50 account audit events;
POST /accounts creates username/password/roles with a reason; PUT /accounts/:id
replaces roles/disabled and optionally password with a reason. Register on existing
Bearer and Origin/CSRF/SCS routes. Strict bounded JSON, canonical strong If-Match,
ETag response, no-store and detailed nonsecret errors. No hidden field binding.
API responses contain only public account metadata, never credential material.

Real bilingual Accounts console uses existing React/shadcn/Tailwind compositions:
explicit create vs edit, username fixed when editing, roles, disable, optional reset,
reason and confirmation explaining all prior sessions become invalid. Show empty,
loading, failure and stale/unknown outcome; retain explicit revision ownership,
never silently rebase drafts. Refresh/rebase discards secrets safely; clear password
as request starts, never browser storage. Account-only capabilities work without
publication/request privileges. Show real account audit distinct from Grant audit.
Use frontend-design/ui-ux-pro-max guidance, preserve established visual language.

## Validation and limits
Observed RED→GREEN for database authority, CAS/audit/rollback, credentials/session
generation, role deny, secret-free strict HTTP, and actual browser lifecycle.
Real browser creates account, logs in, changes roles, disables/re-enables (old cookie
still401), rotates password, verifies real persisted audit and375/1440 layouts.
Restart/reopen tests prove persistence and sticky source. Full Go/race/vet/build,
pinned OpenAPI generation, locked Web and all existing native CI remain mandatory.
No Agent/kernel change; Agent PR19 already proves the human approval conjunction.

## Alternatives and costs
Writing API changes back into the polled file introduces competing authority and
fragile durable revocation. OIDC alone needs an external provider and does not solve
the selected local operating mode. Use explicit database users, retaining a private
bootstrap key; cost: operators must retain/protect that key and opt in deliberately.
Disabling all database admins is recoverable via that key, not a hidden bypass.
