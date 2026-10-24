# Controller first startup and local development

English | [简体中文](2026-10-24-controller-local-development-design.zh-CN.md)

Status: user-approved revised direction, 2026-10-24. Supersedes the earlier developer CLI/initializer design.

## Decision

Normal **unconfigured** Controller startup uses a complete single-instance SQLite setup, creates an initial database-managed `admin` with a random password, and writes that password to a private local file. This is ordinary first-run behavior for development and deployment, not a special development mode. Explicit runtime configuration keeps its existing defaults and trust contracts for compatibility with file-authoritative and provisioned installations.

Go and Web remain independently runnable. Vite keeps its ordinary API proxy to8080 and strict port5173. IDEs run the normal Go entry point without an Origin setting. No developer CLI, preparation script, doctor, generated profiles, new dependency, process supervisor or dedicated development E2E suite is retained.

## Startup contract

- No config file: full backend, HTTP8080, Agent gRPC9090, local decisions9091, database authority/accounts, browser origin unpinned by default, no static assets until explicitly configured.
- Default private state: `var/controller/`, database `controller.sqlite`. Existing versioned migrations and managed-publication bootstrap remain the persistence path.
- Before opening a new default database, prepare private management recovery credentials, empty Agent/caller files, and `initial-admin-password`. Preserve the existing recovery-key requirement; retain its independent raw key in `initial-recovery-key`.
- Files0600 and parent0700. Print only username and file paths, never password/key/hash. Reject unsafe existing files. No overwrite or automatic rotation.
- Create `admin` via existing account save/hash/audit logic only while the database identity-directory revision is zero, after authenticating the retained initial recovery key and checking account-write permission. Nonzero revision is permanent evidence of initialization: disabling accounts or deleting the password file never re-creates an admin.
- Persist the initial password before creating the account. Interrupted first startup can resume using the pending private password. An existing database never triggers regeneration of missing credential files or passwords.
- Existing explicit configurations retain startup behavior; deployed databases never change source/account authority automatically. Environment/flag precedence and unknown-key rejection remain unchanged.
- Empty Agent/caller files authorize nobody. PicketXD/Nginx are not startup prerequisites; readiness is not enforcement proof.

## Development and built Web

Backend: `go run ./cmd/picketx-controller`; Web: existing `npm ci` / `npm run dev`. Select password login and use `admin` plus the private password file. IDEs directly run the backend without an Origin setting.

For built Web use ordinary `PICKETX_BROWSER_ASSETS=web/dist` with the ordinary backend listener8080; reuse the same database and credentials. Restart invalidates sessions, not users/passwords. Keep current HTTP/client timeout bounds.

## Verification and scope

Test first-run login/account audit, restart stability, changed/disabled admin preservation, missing password/credential behavior, private modes and non-disclosure. Verify actual SQLite/HTTP/gRPC startup and Vite proxy/password login using a disposable local smoke run. Reuse existing Go/Web checks; no new platform publication mechanism or developer-test framework.

Implementation is limited to a small normal-startup defaults loader, a focused runtime bootstrap helper using existing account APIs, tests at those boundaries, existing Vite/IDE settings and concise synchronized docs.

## 2026-10-25 amendment: optional public Origin

User approved balancing ease of use with basic request protection. `browser.origin` defaults empty: no fixed Host/port binding, writes must match the request Host and actual HTTP/TLS scheme. An explicit origin retains exact Host/Origin checks. Forwarded headers are not inferred; HTTPS termination and OIDC use explicit public configuration. Cookie Secure follows configured HTTPS or actual TLS without mutating shared session/middleware state. CSRF/session/RBAC remain enabled. Default local Go/Vite startup needs no Origin environment variable.

The existing explicit login-method selector is retained. If capability bootstrap fails, show the error/retry state instead of an apparent key-only authentication screen.
