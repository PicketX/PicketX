# Real Keycloak HTTPS acceptance

## Intent and scope
Close the deployed-provider gap left by PR39/40: an actual pinned Keycloak server,
not a signed protocol fixture, authenticates a browser to an installed Controller
through normal CA/SAN-verified HTTPS and the existing SCS/Casbin/account binding.
This is a disposable integration deployment, NOT a production identity service,
MFA/HA clearance, arbitrary-provider compatibility or Agent/kernel conjunction.
No Go/UI/schema/production dependency or selected product stack change is planned.

## Approach
Use official Keycloak26.8.1 tar.gz released2026-10-13, SHA256
4e2740b181faa2ea623a1c2ecf00855935311e546046e2f857615a47ed24cbe8.
Official archive + native JDK21 is preferable to an additional Docker dependency;
the existing signed fixture alone cannot close this gap. CI downloads and verifies
the exact archive, starts real `kc.sh start --import-realm` with HTTPS and disposable
file database, an exact confidential S256 client and explicitly provisioned users.
Keycloak is a test service, not included in the PicketX distribution.

Use separate loopback sites127.0.0.2 (Keycloak) and127.0.0.1 (Controller Nginx TLS).
Generate a short-lived test CA and IP-SAN leaf; normal Go/Node/Chromium validation
must succeed with that CA, while unknown CA/wrong SAN must fail. Never use
ignoreHTTPSErrors, TLS rejectUnauthorized=false, browser certificate-ignore flags
or trust environment bypasses. Add only the owned test CA to the disposable CI
user NSS database and remove its unique label in finally; no product internal CA
or mTLS is introduced. Controller native HTTP remains private loopback behind Nginx.

Installed actual CLI init and Controller run with empty runtime PATH. Existing
private init/recovery key/database authority remain. Provision an existing human
via real authenticated API, then bind exact imported immutable Keycloak subject.
Real browser password authentication must reject wrong password and an otherwise
valid unbound user; correct bound login returns the exact local principal/roles.
Observe S256/nonce and Secure HttpOnly Lax correlation cookie before callback,
then Secure HttpOnly Strict SCS and cleared correlation/no browser token storage.
Perform a real human publication and verify exact SQLite/readback revision/audit;
change local roles to viewer, old cookie401, fresh IdP SSO sees viewer and write403;
withdraw binding401 and logout destroys the local cookie. Explicitly do not claim
IdP logout/revocation, network access or full installed Agent/application closure.

## Failure/evidence boundary
Generic phase-only public errors; never print raw Keycloak/Controller/Nginx logs,
exception stacks, private paths, password/client secret, state/code/nonce/tokens.
Public JSON is an explicit allowlisted boolean receipt with exact bundle source,
provider version/archive SHA. Only successful secret-free console screenshots.
Stop owned processes with bounded cleanup and delete only owned temp state/NSS CA.
Preserve every old Node/Web/Go/install/service/ingress gate and all4 artifact names.
Add report/image to controller-install-evidence. Full final raw/source/actual
archives/static ELF/notices, independent whole-branch review and expected-head
merge plus fresh main repeat remain mandatory.

## Context map
| File | Responsibility/change |
| --- | --- |
| scripts/real-idp-https.mjs + .test.mjs | Normal TLS request/certificate helper, actual CA/SAN rejection tests |
| apps/web/e2e/real-idp.mjs | Real installed Keycloak/Nginx/Controller/browser lifecycle and allowlisted receipt |
| .github/workflows/ci.yml | Pinned verified upstream/JDK21 + mandatory new native acceptance; retain all old gates |
| docs/OIDC_LOGIN*.md, AUTONOMOUS_DELIVERY*.md | Accurate usage/evidence/lease and remaining conjunction |
| apps/web/e2e/install.mjs, oidc.mjs, scripts/controller-state-copy.mjs | Read-only existing installed/OIDC/privacy patterns |
Dependencies: node built-ins, existing Playwright/npm lock, external CI-only
OpenSSL/Nginx/certutil/JDK/Keycloak. No public API/DB migration/config schema change.
Tests: actual TLS helper regression + complete existing local/native suites.

References checked2026-10-20: keycloak.org/server/{enabletls,importExport,
supported-configurations}; github.com/keycloak/keycloak/releases/tag/26.8.1.
Self-review: exact bounded deployment slice, no placeholders, all scope boundaries
explicit; native inline implementation, one final independent review. User explicitly
waives routine design/plan/push/merge confirmation, not verification or access limits.
