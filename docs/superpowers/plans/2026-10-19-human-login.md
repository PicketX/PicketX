# Local human login implementation

> Execute inline using executing-plans; user's standing authorization covers
> design, implementation, publishing and expected-head merge without approval pauses.

Spec: docs/superpowers/specs/2026-10-19-human-login.md
Base: local9122dee / remotea99d9fc, identical tree52746b5.

## Global constraints

Keep host-owned isolated worktree identity, preserve key login and GPL boundary.
No weakening skipped/native gates. Local baseline full Go has Unix socket EPERM
in TestAuthorizationRuntimeTCPUnixAndShutdown/unix; native CI must pass it.
No password logging, args/environment storage or browser persistence.

### Task 1: Deliver local human login vertical slice

**Files:** internal/managementaccess/{credentials,password}.go and tests;
internal/management/service.go; internal/cli/{config,password}.go and tests;
internal/controller/{browser,browser_sessions,browser_password}.go and tests;
apps/web/src/{App.tsx,lib/console.ts,locales/*.json}; apps/web/e2e/{console,human-login}.mjs;
go.mod; docs/api/swagger.json; docs/{HUMAN_LOGIN,BROWSER_CONSOLE}*.md;
docs/AUTONOMOUS_DELIVERY.md.

**Interfaces:** produces AuthenticatePassword(username,password)(Principal,string,error),
PasswordLoginEnabled()(bool,error), ResolveSession for token/human fingerprints;
consumes these via optional Service interfaces in HTTP. Session JSON adds
password_login bool consumed by method selector. CLI and Access share internal
password hashing/validation helpers. Existing mocks need not implement human login.

1. Write real credential parsing/auth/session and CLI stdin tests. Run focused Go.
   Expected: missing capability/command fails, then strict validation and revocation
   pass after minimal implementation. Preserve existing token tests.
2. Add real HTTP tests for login/CSRF/strict input/method discovery/role refresh,
   credential rotation/logout and direct-peer rate gate. Run controller tests.
   Expected:404 before route, successful renewal/denials after implementation.
3. Add actual browser helper before UI; publish tests-only branch and run CI.
   Expected: password selector absent (RED), existing key workflows still pass.
4. Implement accessible EN/ZH method selector/password fields and real API calls;
   generate OpenAPI, document private provisioning and limitations in EN/ZH.
   Expected: locked npm ci/typecheck/lint/build pass and real browser tests green.
5. Run complete Go normal/race/vet/build, locked Web checks and native CI.
   Expected: only known local Unix EPERM; native all green including Unix100repeat,
   static SQLite/gRPC/restart/SIGTERM and actual Chromium/Nginx/application.
6. Commit logical test/implementation/docs changes using git-commit skill. Fresh
   independent whole-branch review, fix blocking findings RED→GREEN, verify exact
   local/remote/CI trees and all required checks. Merge with expected head, inspect
   main full CI and immutable artifacts; persist exact evidence and release lease.

**Review focus:** independently consider credential type confusion, malformed
bcrypt hashes/CPU exhaustion, timing/username enumeration bounds, role/deletion/
rotation races, mixed authentication, CSRF and forwarded-peer spoofing, Unicode byte
limits, stale method discovery, password-manager semantics and documentation claims.
Review public TLS/distributed limits as explicit deployment boundaries, not provided
features. No new Agent runtime code is in scope.

## Context map

| Boundary | Current owner | Change |
|---|---|---|
| Private trusted identities | managementaccess/credentials.go | Strict users plus existing keys |
| Auth/password CPU | managementaccess/password.go | Bcrypt validation, bounded verification |
| Session freshness | managementaccess.ResolveSession + controller/browser_sessions.go | Domain-separated user credential binding |
| HTTP transport | controller/browser.go + browser_password.go | Exact-origin/CSRF route, local limiter |
| CLI provisioning | cli.New + password command | Stdin-only safe hashing |
| User interaction | App.tsx + locales + console.ts | Method selection and real login |
| End-to-end | e2e/console.mjs + human-login.mjs | Actual binary/SQLite/browser checks |
| Verification | workflows existing Go/Web | Required native full gates unchanged |

中文：逐层先失败再实现，浏览器必须接真实 API；独立评审与准确提交 CI 通过后合并。
