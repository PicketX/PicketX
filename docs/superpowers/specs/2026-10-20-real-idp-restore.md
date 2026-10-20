# Real-provider stopped Controller recovery

Extend the existing real Keycloak26.8.1/normal HTTPS installed Controller flow.
Keep all existing password/unbound/TLS/session/RBAC/publication/revocation proofs,
all three CI jobs and four artifact names. No product Go/API/UI/kernel or dependency
change. Existing autonomous execution and publication/verified-merge authorization
applies; preserve original worktrees/branches and one final independent review.

After actual admin OIDC publication, cleanly stop the installed Controller. Copy
all private instance files to a fresh backup and fresh different restore path,
verify byte manifests, rewrite only controller.yaml and identity/provider.json
path references using the established helpers, and retire the original instance.
Keep the actual external provider running, with issuer/client/subject unchanged.
Restart the shipped Controller with empty PATH and normal CA verification against
the restored instance. Require old-cookie401, identical authoritative views (only
accounts.observedAt excluded after fresh-time validation), same local ID/roles on
fresh observed PKCE/nonce OIDC login, new human publication/audit, binding withdrawal
and no revival of the observed revoked cookie. Preserve authority/keys/binding
bytes and original provider fields. This is stopped single-node Controller state
recovery, not IdP DB recovery, online backup, host reboot, HA or Agent recovery.

A focused helper owns copy/relocation/inventory verification, with fixed sanitized
errors. Its public receipt gate requires exact fields, pinned provider, actual
source descriptor, bounded principal/revision and every proof true. Unit fixtures
are explicitly synthetic. New receipt and1440 screenshot join existing install
artifact only after checked original process/state/trust cleanup. Provider input
must remain untouched. No raw private documents/config/paths/codes/tokens/logs.

## Context map
| File | Purpose/change |
| --- | --- |
| scripts/real-idp-restore.mjs (new) | Private stopped copy/relocate and strict public evidence validation |
| scripts/real-idp-restore.test.mjs (new) | Actual byte/mode/nested-path preservation and unsafe targets, sanitized errors, strict receipt mutations |
| apps/web/e2e/real-idp.mjs | Existing real issuer stays alive while installed Controller is restored; actual browser/API observations |
| .github/workflows/ci.yml | Add mandatory helper regressions, preserve jobs/artifacts/provider pin |
| docs/validation/REAL_IDP_RESTORE*.md | Scope, evidence, runbook/receipt boundaries |
| docs/AUTONOMOUS_DELIVERY*.md | Current owner/lease and final PR authority |

Dependencies: controller-state-copy.mjs (copy/snapshot/relocate/private equality),
controller-service-evidence.mjs (durable views), real-idp-https.mjs (normal TLS,
private provider lifecycle); existing install.mjs signed-fixture restore pattern.
No API break, schema migration or production configuration change.

Self-review: path relocation, stopped-process ownership, exact views, current
identity/session revocation and confidentiality each map to a test/native gate.
