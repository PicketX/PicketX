# Real-provider stopped Controller recovery

The installed real-Keycloak HTTPS acceptance now retains the external IdP while
cleanly stopping, backing up and relocating the complete Controller instance.
This extends, and retains, the original real-provider and separate signed-protocol
systemd restore gates. It does not restore the IdP database or claim online backup,
host reboot, Agent recovery, arbitrary providers, HA or production topology support.

Every private file is copied with0700directories/0600files. Backup and restored
inventories/hashes match before changing only controller.yaml and the nested
identity/provider.json reference. The original path is retired and inaccessible;
no init, new authority, key rotation or partial DB-only restore is used. Private
helper errors are fixed text, never a diff of private documents or raw exceptions.

The shipped static Controller restarts with emptyPATH and explicit normal CA trust
at the same browser origin. Actual HTTPS/API/browser observations must show:
old-cookie401, exact preserved publication/accounts/audit/authority/keys/binding
bytes (only a validated fresh accounts.observedAt is excluded), fresh observed
PKCE/nonce/same-local-principal/admin session, new publication3 and its human audit,
current-role downgrade, binding withdrawal, and no revival of the observed revoked
cookie after rebind. The original signed provider test is not called this proof.

Existing four artifact names stay. controller-install-evidence additionally carries
real-idp-restore-report.json and real-idp-restored-1440.png. The strict public receipt
contains exact source/provider descriptors, public principal/revision, a bounded
private-state inventory count/hash and observed booleans; never private paths,
configuration, subject, code, token or credentials. Only after checked process,
provider-state and owned NSS cleanup may success reports be emitted.

Local30new helper regressions were RED→GREEN; all62 Node tests pass. Baseline Go
normal fails only TestAuthorizationRuntimeTCPUnixAndShutdown because this executor
cannot create Unix sockets (EPERM), never a native success claim. Final raw CI,
actual four archives/static/source/notices/screenshots, one independent review,
expected-head merge and fresh main remain mandatory. Final immutable PR receipt
supersedes this pending source note; whole-product completion is not claimed.
