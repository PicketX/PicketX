# Controller first-start implementation plan

**Goal:** normal startup initializes SQLite and an admin password, with independent Web/Vite development.
**Spec:** [approved revised design](../specs/2026-10-24-controller-local-development-design.md).
**Constraints:** reuse account/hash/audit/migrations; explicit-config compatibility; no reset/overwrite, dev CLI, new dependency, script or test framework.

- [x] Add normal unconfigured startup defaults at the existing loading boundary; preserve explicit-file loading and env/flag precedence. Test defaults and overrides before implementation.
- [x] Add focused runtime bootstrap helper: private first-run inputs/password before DB creation, create admin only at directory revision0 via existing SaveAccount, no regeneration after initialization. Test restart, disable/change, missing files, permissions and log redaction before implementation.
- [x] Remove obsolete development script/config and document IDEs running the normal Go entry point with browser Origin5173. Keep only strict Vite port addition.
- [x] Rewrite concise English/Chinese docs around two processes and automatic first startup; built Web uses ordinary environment overrides.
- [x] Run Go test/vet/build, Web ci/typecheck/lint/build, focused bootstrap race tests and a disposable real Controller/Vite/password-login smoke. Review final diff for unnecessary implementation.

Execution: user approved this startup behavior for normal deployment as well as development. Work in the current shared workspace, preserve unrelated `target/`, do not commit or alter user data. Previous heavyweight implementation has been removed.


Verified on macOS: full `TMPDIR=/private/tmp go test ./...`, focused bootstrap race tests, `go vet ./...`, `go build ./...`, Web npm ci/typecheck/lint/build. A disposable real Controller/Vite/Chromium smoke passed automatic SQLite/admin initialization, private random password, restart/session invalidation, no-reset after deleting plaintext copy, and built-Web password login with stable account audit. Existing user Vite5173 was preserved; the smoke used5174. No smoke framework was retained.

Independent review found a recovery-key/retained-credential mismatch edge; added a failing regression, required actual recovery authentication/account-write permission, then confirmed passing. Reviewer confirmed no further material findings.

Ruling: retain the user's subsequent removal of IDE-specific configuration files; document native IDE settings instead. Preserve the user's unrelated `.gitignore`/target changes. No commits or user-state changes performed.

## 2026-10-25: optional public Origin (user approved)

- [x] Default browser.origin to empty; accept omitted/empty Origin config, retain strict validation when configured. Require an explicit public Origin for OIDC callbacks.
- [x] With no configured Origin, use request Host and actual HTTP/TLS scheme for same-origin write checks; do not trust forwarded headers. Retain CSRF/session/RBAC and configured Host checks. Test arbitrary local ports, cross-origin rejection, explicit-origin pinning and cookie Secure behavior without shared mutation.
- [x] Avoid showing a credential form when session capability bootstrap failed; retain the existing explicit login-method selection. This addresses the access-key confusion observed by the user.
- [x] Remove mandatory Origin environment settings from developer instructions, sync English/Chinese contracts, and verify focused/full Go and Web checks plus actual Vite/default-startup password login.


Optional-Origin validation: full Go test/vet/build and focused concurrent HTTP/TLS cookie race checks passed; Web npm ci/typecheck/lint/build passed. Disposable real Chromium+Vite+Controller smoke used separate temporary ports and no browser Origin setting; verified password login, session expiry on restart, missing initial-password copy without reset, built Web, and failed session bootstrap showing error/retry without an access-key form. Independent targeted review reported no material findings. User live services were not restarted or modified.
