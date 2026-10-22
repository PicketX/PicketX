# OIDC browser sign-in

## Intent and boundary
Complete the agreed M1 OIDC workflow, not a new identity product. One opt-in OIDC
provider authenticates an existing human account; trusted local/file or database
roles still come from managementaccess on every request. No email linking, JIT
accounts, imported claim roles, network Grants, protocol identities, refresh tokens,
IdP logout, MFA implementation, or Agent changes. Local key/password recovery stays.
Routine spec/plan/execution approvals are explicitly waived by the autonomous
mandate; implement inline and retain one independent whole-branch review.

## Approach and alternatives
Use pinned coreos/go-oidc v3.21.0 and golang.org/x/oauth2 v0.37.0, checked through
the module proxy on 2026-10-20. They perform discovery/code exchange/JWS verification;
PicketX supplies transaction/session binding and trusted local account resolution.
Hand-written JWT verification is rejected; a fixed external auth gateway would
add an unselected deployment dependency and not implement the portal workflow.
Official sources: https://github.com/coreos/go-oidc/tree/v3.21.0 and
https://pkg.go.dev/golang.org/x/oauth2@v0.37.0 (S256ChallengeOption/VerifierOption),
https://openid.net/specs/openid-connect-core-1_0.html#IDTokenValidation.

## Configuration and identity authority
`browser.oidc_config` optionally names a private regular JSON file containing
`issuer`, `client_id`, `client_secret`, `bindings`. All are required nonempty
strings; bindings is an absolute path to a separate private strict JSON document
`{"bindings":[{"subject":"...","principal_id":"principal:...","generation":"32 lowercase hex"}]}`.
Max1MiB/128 unique subjects; empty bindings is valid deny-all. Bindings are reread
for each OIDC resolution. Provider/client/secret/path require process restart.
Issuer and discovered auth/token/JWKS endpoints require HTTPS, except literal
loopback HTTP for local development; no userinfo/fragment. No insecure TLS option.
Redirect URI is exactly configured browser origin + `/console/api/v1/oidc/callback`.
Only human users can be linked, never API-key principals. Generation rotation,
binding removal/retarget, disabled account, account generation/password changes
invalidate old sessions. File-backed remove/readd requires a new binding generation
to prevent old cookies reviving; managed accounts already have permanent generations.
No writes to file authorities and no implicit authority-mode switch.

## Browser transaction
POST `/console/api/v1/oidc/start` uses existing exact Host/Origin and Echo CSRF and
ordinary limits, plus bounded login-start rate. Generate independent 256-bit state,
nonce, PKCE verifier and browser correlation secret. Keep pending transactions only
in process memory: max128, absolute5min, remove expired on use/start, single consume
under mutex. Return only authorization URL; never tokens or verifiers.
Separate HttpOnly SameSite=Lax callback-path correlation cookie, Secure on HTTPS,
max-age300 permits external GET callback without relaxing Strict SCS cookies.
GET callback accepts exactly one state/code, rejects errors/ambiguous input, consumes
matching transaction once before any exchange, requires correlation cookie, PKCE,
nonce, verified issuer/audience/signature/expiry, nonempty printable ASCII subject
up to255 bytes, valid issued-at and authorized-party for multiple audiences.
Network calls timeout5s with bounded response bodies, no redirects. Failures expose
only a stable generic redirect to `/console/?login_error=oidc`, never provider text.
Success destroys/renews SCS, stores only local ID/account fingerprint plus external
subject/binding fingerprint and absolute ID-token expiry; redirects to `/console/`.
Every later request re-resolves binding and current human roles, with session deadline
no later than ID-token expiry or existing8h/30min SCS limits. Logout is local only.
Cookies, codes, tokens, claims, secrets and provider diagnostics are not logged/audited.

## Real console and verification
Existing EN/ZH sign-in panel gets a separate organization sign-in button only when
session bootstrap reports `oidc_login`; clear secret fields before navigation,
show loading/failed callback/retry, no mocked identity or sample authorization data.
Real signed HTTP issuer fixtures exercise discovery/JWKS/code/PKCE and hostile tokens;
actual Chromium → built Controller → signed protocol fixture → SCS/Casbin → persisted
management API proves wiring. The fixture is not a deployed production IdP and must
not be described as one. Existing native Nginx/Agent/app acceptance remains intact.
Full Go normal/race/vet/build, locked Web, native CI/raw/archive/source verification,
one independent whole-branch review, expected-head merge and main recheck remain gates.

## Context map (self-reviewed)
| Files | Role/change |
| --- | --- |
| internal/oidclogin/{client,bindings}.go + tests | New focused framework-free protocol and private binding adapter |
| internal/managementaccess/oidc.go + tests | Resolve only current human users and account fingerprints |
| internal/management/service.go | Narrow optional human-resolution forwarding |
| internal/controller/browser{,_sessions,_oidc}.go + tests | Transaction cookies, protected start, safe callback and live session resolution |
| internal/runtimeconfig/config.go + tests | Typed optional restart-only private provider config path |
| internal/controllerapp/runtime.go + tests | Explicit bounded client construction/injection |
| web/src/{App.tsx,lib/console.ts,locales/*} | Real existing sign-in composition, EN/ZH states |
| web/e2e/*, .github/workflows/ci.yml | Actual protocol/browser acceptance, all prior gates retained |
| go.mod/go.sum, scripts/collect-bundle-licenses.mjs | Pinned dependencies and actual release notices |
| docs/OIDC_LOGIN*.md, AUTONOMOUS_DELIVERY*.md | Operator contract/evidence/lease |

Dependencies: existing management.Service, SCS, Casbin, private credential loader,
account version fingerprint, browser Origin/CSRF wrappers, release manifest builder.
Reference patterns: browser_sessions.go, browser_password.go, human-login.mjs.
Risks: additive API/config; no business DB migration; external identity trust and
callback races are high risk. Self-review: exact endpoints/limits/identity provenance
defined, no placeholders or authority ambiguity; source and network roles stay distinct.
