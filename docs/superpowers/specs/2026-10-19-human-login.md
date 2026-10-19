# Local human login

Deliver a real username/password entry to the existing console, retaining access
keys, independent Agent Tokens, SCS and Casbin. Operator-controlled private
`management.credentials` remains the sole identity authority. This increment is
not OIDC, database account CRUD, self-service registration or request/approval.

The credential document retains required `principals` and adds optional `users`:
each user has id, username, password_hash, roles, disabled. IDs are globally
unique; usernames are unique lowercase ASCII, 3–64 characters, starting with a
letter/digit, then letters/digits/dot/underscore/hyphen. Explicit null is invalid.
Combined maximum128 identities; existing private regular-file/1MiB constraints
remain. Roles follow existing trusted role/group validation. Bcrypt cost12 hashes
are accepted; passwords12–72 UTF-8 bytes, no trimming. Hashing uses the already
pinned x/crypto dependency. `picketx password-hash` reads one password from stdin
(one terminal LF/CRLF removed), never accepts password arguments, prints only hash.

POST /console/api/v1/login/password uses strict4096byte JSON username/password,
existing exact Host/Origin and Echo CSRF. Wrong/disabled/unknown identities return
the same401. Unknown identities also perform cost12 comparison. A direct-peer
limiter6/minute with burst5 ignores forwarded headers; four concurrent bcrypt
checks per Access instance cap CPU, saturated checks fail503. This is local,
not distributed brute-force protection; production ingress needs global controls.
`session.password_login` advertises configured enabled local users, no enumeration.

Successful login renews SCS and stores only ID plus domain-separated SHA256 of
ID/username/password hash. Each request re-reads identities and trusted roles;
password/username rotation, disable/removal invalidate old sessions on next request.
In-flight operations are not undone. API Bearer authentication stays key-only.
Absolute8h/idle30m, restart invalidation and current logout serialization remain.

UI extends the existing login panel with a visible method selector only when
local users exist, real username/password fields with password-manager metadata,
generic rejection and rate-limit feedback. No browser storage of secrets. Loading,
failure/retry and session expiry remain explicit, EN/ZH supported. Retain existing
console palette/type/spacing; no new marketing/visual system. Labels and focus
remain visible, touch controls44px, narrow375px and desktop1440px verified.

Acceptance: RED→GREEN parsing/auth/session/CLI/HTTP tests and a real Chromium flow
through actual Controller/SQLite/Casbin (no successful API mocks), password admin
write/audit, viewer deny, role change, password rotation, disabled-user revocation,
logout and unchanged key login. Full native Go/race/vet/build and locked Web checks,
real existing Nginx/application suite, fresh whole-branch review, exact-head CI and
main verification required. No new kernel behavior or production TLS claim.

Design self-review: file authority matches current runtime and does not create a
second account source. Fixed hash cost plus bounded verification limits CPU work;
local limiter is explicitly not cluster-wide. Key-only bootstrap remains viable
when no users configured. The later account UI can replace provisioning deliberately.

中文：新增私有文件配置的人员密码登录，沿用会话与权限体系；密钥与 Agent 凭据独立。
密码轮换、禁用和删除在下一请求失效。暂不包含 OIDC、账户管理与申请审批。
