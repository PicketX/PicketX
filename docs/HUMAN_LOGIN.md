# Local human sign-in

The console supports operator-provisioned usernames/passwords alongside access
keys. Both resolve current trusted roles through the same injected Casbin/SCS
boundary; human accounts are **not** network source identities or Agent Tokens.
This is a single-Controller local identity implementation, not OIDC, self-service
registration, a database account editor, per-resource delegation or approval.

## Provision without plaintext configuration

Build `go build -o picketx ./cmd/picketx`. `picketx password-hash` reads exactly one
password from stdin and prints a bcrypt cost12 hash. It accepts no password
arguments/flags. Passwords require12–72 UTF-8 bytes, including spaces, without
trimming; one terminal LF or CRLF is removed. Embedded newlines are rejected by
the provisioning command. Never put real passwords in shell history, command
arguments or environment variables. For an interactive Bash shell:

```sh
umask 077
IFS= read -r -s -p 'Password: ' picketx_password
printf '\n' >&2
printf '%s' "$picketx_password" | ./picketx password-hash
unset picketx_password
```

Use the printed hash in the existing private `management.credentials` JSON:

```json
{
  "principals": [],
  "users": [{
    "id": "principal:alice",
    "username": "alice",
    "password_hash": "REPLACE_WITH_COMMAND_OUTPUT",
    "roles": ["role:admin"],
    "disabled": false
  }]
}
```

The placeholder is intentionally invalid. Keep existing key principals if needed;
`principals` is required even when empty. `users` is optional; null, unknown/duplicate
fields, duplicate usernames/IDs or invalid roles/hashes fail the entire credential
authority closed, including keys. Files must remain private regular files (0600),
at most1MiB; combined identities at most128, roles1–16. Usernames:3–64 lowercase
ASCII characters, initial letter/digit, subsequent letters/digits/dot/underscore/
hyphen. IDs are globally unique `principal:` identifiers. Hashes must be60-character
`$2a$12$` or `$2b$12$` bcrypt encodings; plaintext/other costs are rejected. Use
`role:viewer` for read-only, `role:admin` for management writes; `role:blocked`
denies even when combined with an allowing role. Trusted group membership and
periodic Casbin policy reload behave as documented in [management API](MANAGEMENT_API.md).
Write a private replacement file and atomically rename it into place; do not leave
partial JSON during rotation. Do not commit credentials or upload them as evidence.

## Browser and API behavior

With an enabled user configured, `/console/` shows Sign-in method → Username and
password. Existing access-key login remains available and is the default. Fields
support password managers, paste and visible-password toggle; secrets are never
written to localStorage/sessionStorage. English and Simplified Chinese are supplied.
Wrong, absent and disabled accounts receive a generic authentication failure, not
account existence details. The session bootstrap reports only `password_login`,
not usernames or hashes. When all users are disabled/removed, the UI returns to keys.

`POST /console/api/v1/login/password` accepts strict JSON `{username,password}`,
limited4096bytes, application/json only. Exact public Host/Origin and Echo CSRF
are mandatory, exactly as for key login. Forwarded identity/address headers are
not trusted. There is an additional local direct-peer bucket6/minute, burst5,
returning429; four concurrent expensive comparisons per Access instance return503
when saturated. Unknown/disabled users with otherwise valid input perform the same
cost12 comparison. Invalid-format usernames/password lengths may reject earlier;
this is not an absolute timing-privacy guarantee.

Successful login renews SCS and stores only principal ID and a domain-separated
SHA256 binding to ID/username/password hash. Roles and credential validity are
re-read on each authenticated request. Password/username rotation, disabled/removal
invalidate sessions on their next request; role changes immediately affect the
next operation. Logout destroys the current session server-side, including
in-flight idle-refresh protection. Absolute8h, idle30m, process-memory storage and
restart invalidation are unchanged. In-flight authorized changes are not undone.
To invalidate *all* prior cookies permanently before re-enabling/recreating an
identity, generate a new salted hash (even for the same password). Disabling and
restoring the exact same record without an intervening request is not a persistent
revocation-generation database. API `/api/v1/*` remains Bearer-key-only; human
passwords and console cookies cannot authenticate that API.

Use HTTPS through the existing trusted ingress for non-loopback use, preserve
Host/Origin and restrict Controller direct access. Limits are process-local, not
cluster-wide or per-person account lockout; a reverse proxy shares the direct-peer
bucket. Production ingress must provide global abuse controls, connection/body
limits and appropriate monitoring. No MFA, password-reset delivery, password change
screen, distributed sessions, login-event audit or production deployment claim is
included. Publication audit does record the real human principal for accepted writes.

## Validation scope

Go tests exercise strict credentials, key/password separation, trusted role deny,
freshness/rotation/disable, CSRF/renewal and bounded work. CLI tests exercise stdin,
newline/length limits and bcrypt verification. The required real browser suite
uses the built Controller, SQLite, SCS/Casbin and the actual CLI hash command:
human sign-in, publication readback and audit actor, viewer backend denial, logout
cookie replay, rotated-password sign-in, disabled-account revocation and375/1440
layout. Existing key and Nginx/application suites remain required. These do not
establish OIDC, approval, changed Agent kernel behavior or production TLS safety.
