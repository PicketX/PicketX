# Operator browser console

This document covers complete desired policy/facts publications, transactional audit and console deployment. [Grant](GRANT_MANAGEMENT.md) and [Resource](RESOURCE_MANAGEMENT.md) pages provide structured admin flows; [HTTP authorization](HTTP_AUTHORIZATION.md) is a separate scoped ingress integration. [Local human login](HUMAN_LOGIN.md) retains file provisioning; [managed accounts](MANAGED_ACCOUNTS.md) adds explicit database-authoritative human lifecycle with permanent old-session invalidation. OIDC remains deferred. Request/independent approval is delivered separately. Saving is **not node application**. Node observations are point-in-time reports, not independent kernel confirmation.

## Run

Use the database-authoritative configuration from [management API](MANAGEMENT_API.md), with separate private Agent and management credential files. Build Web using the committed lockfile:

```sh
cd apps/web
npm ci
npm run typecheck
npm run lint
npm run build
```

Add to the Controller runtime YAML (paths are relative to its working directory):

```yaml
browser:
  enabled: true
  origin: https://control.example.com
  assets: apps/web/dist
```

Expose the Controller behind your existing HTTPS ingress without rewriting the public Host. `origin` must be the browser's exact `location.origin`, without trailing slash, path, user info, query or fragment. Do not include a default port that the browser omits. The Controller itself does not terminate HTTPS or trust forwarded TLS/identity headers. Restrict direct Controller network access; ingress is the TLS boundary. Open `/console/` and enter the independent management access key corresponding to a configured principal. Agent Tokens cannot sign in. No access key is written to localStorage, sessionStorage, cookies or SCS values.

For loopback-only development, use `origin: http://127.0.0.1:8080` (IPv6 `http://[::1]:8080` also accepted). For Vite development, omit `assets`, set `origin: http://127.0.0.1:5173`, run `npm run dev`; the `/console/api` proxy preserves the public Host to the Controller at127.0.0.1:8080. Never use a remote plaintext development origin. Browser settings are rejected unless explicitly enabled and source mode is database; authority never switches back to policy-file polling.

## Security and lifecycle

SCS2.9.0 uses one process-lifetime memory store and its native expiry cleanup. Sessions do not survive restart or work across replicas. Cookie: `picketx_session`, HttpOnly, SameSiteStrict, path `/console/`, Secure for HTTPS origins, browser-session persistence. Server lifetime8h and idle30m; ordinary reads refresh idle time, never the absolute deadline. Login renews the identifier/deadline; logout destroys it. Complete load/handler/save is serialized only for an existing authenticated session, with exact-token reference-counted locks. Unknown cookies take no lock; unrelated sessions cannot collide. Waiting is cancellation-aware, sessions reload under the lock, and entries are removed when no request holds/waits for them. An older idle refresh cannot resurrect logout. Ordinary connection/body limits remain necessary; this is not a general denial-of-service guarantee.

Only principal ID and credential fingerprint are held server-side. Every authenticated request re-reads the trusted private credential file and resolves current roles. Removed/rotated keys invalidate old sessions on next request, even when the ID is reused with a new credential. In-flight authorized business operations are not retroactively undone. Role changes take effect on next request; Casbin policy changes follow the separately configured periodic reload and fail-closed readiness behavior. Logout affects the current session; rotate the credential to permanently invalidate all its sessions. [Human login](HUMAN_LOGIN.md) adds username/password entry with the same freshness boundary, not OIDC or per-resource delegation.

Every console request must have the configured public Host. Unsafe requests must also have its exact Origin; forwarded headers cannot satisfy either check. Echo's native CSRF protects login/logout/writes: modern browser `Sec-Fetch-Site` must be same-origin (or native permitted non-web metadata with the independent Origin check), while legacy clients must retain the HttpOnly CSRF cookie and echo `/console/api/v1/session`'s `csrf` value in `X-CSRF-Token`. The modern bootstrap value is Echo's metadata marker rather than a cookie token. No CORS is enabled. Login shares the ordinary management local peer rate limits, independent from Agent stream limits.

`/api/v1/*` remains Bearer-only. `/console/api/v1/*` is cookie-only and rejects any Authorization header, including an otherwise valid management Token. All console responses are no-store with CSP, no-referrer and nosniff. Built assets are explicit regular files opened through `os.Root`, with an extension allowlist, no directory listing, symlink escape or API SPA fallback. Build output is not committed.

## Flow and freshness

Read the current policy/facts, edit full documents, supply a change reason, and save using the editor's strong revision ETag. Shared configuration validation runs before acceptance; transactional audit records actor/reason/revision/digest. Viewer credentials can read but cannot save. Error messages do not echo credentials or raw documents; validation currently reports a generic policy/facts error rather than field-by-field compiler details.

Refresh status updates the observed state and current permissions but **never advances the editor CAS base** or replaces a draft. A conflicting write returns412 and preserves policy, facts and reason; only explicitly reloading the latest document (with discard confirmation when dirty) changes its base. There is no automatic retry/merge/overwrite. Tabs preserve the draft; leaving a dirty page invokes the browser's unsaved-change protection. Sign-out/revocation clears local documents and identity. An older successful read remains visible after a read error with a stale warning; a view older than one minute is marked stale. Refresh is manual, not a claim of continuous real-time monitoring. Audit is an ascending exclusive cursor (50rows per page), not an enforcement/request-decision log.

## Evidence gate

`npm run test:e2e` requires Go, the built dist and `npx playwright install --with-deps chromium`. It launches the actual Go binary with fresh private credentials and SQLite, then drives actual Chromium through sign-in, CSRF-backed save, persisted readback/audit, external-writer conflict/draft preservation, a deliberately failed read (negative UI test only), viewer backend denial/mobile layout and credential revocation. CI retains screenshots only, never the temporary credential/database directory. Required Go/race/vet/build/static/OpenAPI and locked Web checks remain in place. This flow does not yet validate browser→L7→ingress proxy→application or a changed Agent kernel implementation.
