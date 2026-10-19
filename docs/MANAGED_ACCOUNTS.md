# Managed human accounts

English | [简体中文](MANAGED_ACCOUNTS.zh-CN.md)

This opt-in directory manages local humans, not Agent tokens, ingress credentials,
network subjects or application accounts. Existing file-user installations retain
their behavior. OIDC, registration, mail recovery and multi-replica sessions are not
implemented by this increment.

## Enable deliberately

Use database publication authority and set runtime configuration:

```yaml
source: {mode: database}
management:
  credentials: /etc/picketx/management-keys.json
  users_mode: database
```

The private0600 key document must contain `principals`, no nonempty `users`, and
at least one `role:admin` key without `role:blocked`. Protect this recovery key;
it allows recovery if all human admins are disabled. Key IDs beginning
`principal:account-` are reserved and rejected here. Agent credentials remain
separate. There is no automatic import of file users.

Initialization persists a sticky DB-user marker. Returning to `users_mode: file`
(the default), including omitting the option after initialization, fails closed.
Existing file-mode processes fail their next credential read/readiness check if
another process initializes this DB's user authority. Adding file users or losing
the recovery key also fails readiness and authentication; restore the valid private
key document rather than attempting to merge authorities. Back up the entire private
database and recovery keys together. Do not edit identity tables directly.

## Console workflow and permissions

Accounts is visible with `account:read`; writes require `account:write`. The additive
management schema migration3 grants both to admin and denies both to blocked.
Viewer/requester do not receive directory access by default. Shared custom Casbin
rules remain supported; humans edited here select only the four built-in roles.

Create a unique immutable lowercase ASCII username3–64 characters, password12–72
UTF-8 **bytes**, role(s), and audit reason1–512UTF-8bytes. Maximum128 humans.
An account can be disabled/re-enabled or have its roles/password changed. No
delete/rename/username recycling API is provided. Empty optional reset password
retains the existing hash. Every edit, even a same-value or role-only edit, invalidates
all previous human sessions. Re-enabling or restarting cannot revive old cookies.
Already authorized in-flight operations are not retroactively undone.

The directory has its own revision, unrelated to configuration publication.
Refresh never silently changes a dirty draft's CAS base. On conflict or unknown
save outcome, inspect the latest rows/audit then explicitly clear the draft before
retrying. Passwords are cleared when a save request starts; re-enter only after
reconciling an unknown outcome. They are never returned, audited or browser-stored.
Account changes grant no network access and do not establish Agent application.

## API

| Method | Bearer path (browser alias under `/console/api/v1`) | Permission |
| --- | --- | --- |
| GET | `/api/v1/accounts` | account:read |
| POST | `/api/v1/accounts` | account:write |
| PUT | `/api/v1/accounts/{id}` | account:write |

GET returns accounts and latest50 account audit events, directory revision and
observation time. Writes require a canonical strong directory `If-Match`, strict
JSON16KiB, reason, and on update an explicit boolean `disabled`. Account write,
revision CAS and audit append share one transaction. A missing match is428;
stale is412, malformed400, missing account404, unavailable503. Database mode only.
Browser aliases additionally require exact configured Origin, native Echo CSRF and
SCS identity; cookies never authenticate Bearer API paths. Bcrypt12 work shares the
four-slot login budget. Passwords and hashes never enter SQL error logs.
