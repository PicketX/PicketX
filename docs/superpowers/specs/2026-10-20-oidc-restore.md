# Installed OIDC stopped-state recovery

Intent: close the optional-OIDC hole in installed Controller service/backup
acceptance. PR39 is already merged/verified. Keep all original password/key,
crash-restart, static-binary, private-copy, durable-view and relocation assertions.
This increment is NOT production IdP/HTTPS/Agent conjunction acceptance.

## Design
Extend the real systemd flow with an additional OIDC browser context. Extract the
existing signed HTTP issuer into a reusable test utility; preserve its one-use
code, PKCE/client/redirect/nonce checks and all original oidc.mjs tests. It remains
explicitly a signed protocol fixture, never a deployed IdP or shipped service.

Private instance: identity/provider.json and identity/bindings.json, directory0700,
regular files0600. Runtime points to provider; provider points to bindings. Offline
relocation changes only exact runtime private paths and nested bindings pathname.
Retain issuer/client ID/secret, subject/account/generation, policy and authority.
Only these exact fixture paths are supported; external/unexpected paths fail.
No external traversal, general backup API, runtime/UI/Agent code or new dependency.

Before stop, real Chromium signs in through issuer to the existing database
operator and reads revision2. Capture cookie. Stop actual unprivileged service;
hash/copy all private files; restore at new path; rewrite two path layers and
make original instance unavailable. Check all other bytes/provider fields equal.
Restart installed static binary as same unprivileged user. Old password AND OIDC
cookies401, exact durable views/authority retained, fresh OIDC login selects same
ID/current roles/revision2. Remove restored binding→401; restore bytes→observed
revoked cookie still401; another new login works. No browser storage/log secrets.

## Public evidence
Add service.restore.oidc with boolean checks and
providerKind:'signed-protocol-fixture', no private path/secret/token/fingerprint.
All existing reports/scenarios remain intact. Native Linux CI mandatory, no skips.
Keep deployed real IdP/normal HTTPS/Secure-cookie and installed Agent conjunction
explicitly pending; a fixture passing cannot close that goal.

## Context Map
| File | Responsibility/change |
| --- | --- |
| scripts/controller-state-copy.mjs | Optional exact provider path, separate nested bindings relocation |
| scripts/controller-state-copy.test.mjs | Literal paths, unchanged fields, external path refusal |
| web/e2e/oidc-provider-fixture.mjs | Reusable signed protocol test server |
| web/e2e/oidc.mjs | Extract fixture only, retain existing assertions |
| web/e2e/install.mjs | Additional OIDC context + actual restored service checks |
| docs/OIDC_LOGIN*.md | New bounded recovery evidence, honest remaining limits |
| docs/AUTONOMOUS_DELIVERY*.md | Lease/progress and final PR receipt routing |

Dependencies: existing Playwright/systemd/static bundle/private copy/durable views.
No public API, schema or DB migration. Risk: merely rewriting JSON can conceal
continued access to old files; original path unavailability +fresh login required.
Self-review: focused integrated deliverable, explicit boundaries/interfaces, no
unresolved requirements. Native inline execution; routine pauses waived by user,
not verification or one independent full-branch review.
