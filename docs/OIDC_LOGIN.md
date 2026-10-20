# OIDC browser sign-in

OIDC is opt-in and authenticates **existing human accounts**, not source IPs,
Agent Tokens or application users. One provider authenticates an exact `issuer`
and `sub`; an explicit private binding selects an existing account. Email, groups
and roles in tokens do not grant PicketX permissions. Current local roles still
pass through Casbin with deny precedence. Signing verification uses pinned
coreos/go-oidc v3.21.0/go-jose4.1.4 and oauth2 v0.37.0, not a custom JWT verifier.

## Configure

Keep the existing database policy authority, management credentials, browser origin
and optional database-user authority. Provision a human account through Accounts
and copy its immutable ID; retain the independent recovery admin key. File-backed
human accounts are also supported; never map an API-key principal.

Set the restart-only runtime setting `browser.oidc_config: /private/oidc.json`
(or `PICKETX_BROWSER_OIDC_CONFIG`). It requires an enabled browser and an absolute
path. Store this strict JSON as a regular private file (0600, not a symlink):

```json
{
  "issuer": "https://identity.example.test",
  "client_id": "picketx",
  "client_secret": "REPLACE_WITH_PRIVATE_CLIENT_SECRET",
  "bindings": "/private/oidc-bindings.json"
}
```

Example notation only; no real provider or credential is included. Register an
authorization-code confidential client at your IdP with the exact redirect URI
`https://YOUR_CONTROLLER_ORIGIN/console/api/v1/oidc/callback`, including any configured
port. No caller-selected redirect URI or return target is supported. Enable S256
PKCE and OpenID `openid` scope. Use the existing trusted HTTPS ingress; PicketX does
not terminate application TLS. Issuer/auth/token/JWKS endpoints must be HTTPS with
normal certificate verification. Literal loopback HTTP is allowed only for local
development, not DNS names/remote plaintext or disabled TLS verification.

Private bindings JSON, max128 unique subjects and1MiB:

```json
{"bindings":[{
  "subject":"EXACT_SUB_FROM_VERIFIED_IDP",
  "principal_id":"principal:EXISTING_HUMAN_ID",
  "generation":"REPLACE_WITH_32_LOWERCASE_HEX"
}]}
```

The generation placeholder is deliberately invalid. Generate16 random bytes and
encode lowercase hex; keep generations unique on removal/recreation. Empty
`{"bindings":[]}` explicitly denies all OIDC identities. Null, unknown/duplicate
fields, duplicate subjects, malformed generation, public/nonregular files fail
closed. Atomically replace the complete private binding file. No Controller write
or file polling backfill changes this authority. Provider/client/secret/binding
path changes require restart; binding contents are reread every OIDC request.

## Login, expiry and recovery

The console adds “Continue with organization” only when configured. Start is POST
with exact public Host/Origin and Echo CSRF; a separate process-local direct-peer
bucket6/minute/burst5 applies. State, nonce, verifier and browser correlation are
independent256bit values. Pending transactions are one-use, max128, expire after
5minutes and are lost on restart. Starting another flow overwrites that browser's
correlation cookie; finish the latest flow or restart sign-in.

Only a top-level GET callback is supported, not form_post/implicit/hybrid. A
separate callback-path HttpOnly SameSite=Lax cookie allows cross-site return.
The ordinary SCS cookie stays HttpOnly SameSite=Strict; both are Secure on HTTPS.
The callback verifies state/correlation/PKCE, exact issuer, audience, signature,
expiry, nonce, nonempty subject, issued-at and authorized party. Network calls
are bounded5s/1MiB and do not follow redirects. Failure returns a fixed local
error destination with generic EN/ZH recovery text, never IdP diagnostics.

Success renews SCS and stores only local ID/account fingerprint, subject/binding
fingerprint and token expiry. No external access/ID/refresh token, client secret,
email or roles are persisted in the browser session. Sessions cannot outlive the
ID token's absolute expiry or existing8h/30min absolute/idle bounds. Provider
refresh tokens, silent refresh and remote logout are not implemented.

Every authenticated request rereads the binding and current human account. Remove
or retarget a binding, rotate its generation, disable/remove a human, or change
its account/password generation to invalidate old sessions. Database account
edits permanently invalidate older versions even after re-enabling. For file
accounts/bindings, removing and restoring identical data without an intervening
request can revive an old cookie: rotate the binding generation when restoring.
Role-only file edits take effect on the next request. Unavailable binding storage
returns503, not cached authorization. Already-authorized in-flight work is not undone.

Local sign-out destroys the PicketX session but **does not sign out at the IdP**.
The IdP may immediately authenticate a new attempt. IdP-side account/session
revocation is not queried on each PicketX request; bound local account revocation
and ID-token expiry define the current maximum freshness boundary. API `/api/v1/*`
remains independent Bearer-key-only. Login/logout never grant/revoke network access.

If discovery/config fails, startup fails with a detail-free OIDC initialization
error. Correct the private files or remove `browser.oidc_config` and restart; the
recovery key/password paths remain independent once the configured process starts.
Counters/sessions/pending state are process-local, not HA/distributed quotas.

Keep the provider and binding files inside the private instance directory when
backing up the complete stopped instance, or separately include every external
referenced file. A restore to a different path also requires explicitly reviewing
`browser.oidc_config` and the nested provider `bindings` path; the existing
test-only four-path relocation helper does not rewrite those. Installed OIDC
service/backup/restore acceptance is still pending; existing non-OIDC restore
evidence is not evidence for this optional configuration.

## Evidence boundary

Go tests use actual HTTP discovery/JWKS/token exchange and signed RSA fixtures,
including PKCE, replay/concurrency, wrong claims, issuer alias, current binding and
human-role checks. The real Chromium suite uses the built Controller/React/SQLite,
cross-site127.0.0.2→127.0.0.1 callback, current account roles, publication/audit,
binding withdrawal, nonce rejection and local logout. Its issuer is explicitly a
signed **protocol fixture**, not a real deployed IdP or authentication service.
Existing native ingress/Agent/application gates remain required. Generic IdP
integration still needs operator registration, HTTPS topology and actual provider
acceptance; no production account, MFA/IdP outage policy or arbitrary provider
compatibility is claimed from fixtures.
