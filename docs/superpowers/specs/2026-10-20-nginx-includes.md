# Private packaged Nginx authorization includes

Scope: a local `picketx nginx-init` command, not an ingress installer. Existing
ingress retains TLS, listeners, business proxying and upstream isolation. The CLI
creates a NEW0700 directory with `location.conf` and `server.conf`0600. Include the
former inside each protected location and the latter in its server. No reload,
registry mutation, publication or Grant is performed. Approval/publication remains
different from real execution. Existing Go/Cobra and static bundle are retained.

Inputs: absolute new directory with existing non-symlink parent, fixed valid route
UID, explicit literal unmapped/unzoned IPv4/IPv6 loopback IP:nonzero-port callback,
and absolute private regular non-symlink token file. This installation helper
deliberately supports the64lowercase-hex ingress token emitted by controller-init
(optional single LF); other runtime-valid tokens are refused, never normalized.
This bounded subset avoids Nginx variable/directive injection without changing the
decision API's existing32–256ASCII token contract. Token never appears in flags,
stdout or errors. Trusted operator controls parent directories exclusively.

Route-derived SHA256 names isolate method variables/internal callbacks across
multiple routes. Protected include forces `satisfy all`, captures original method
before auth_request. Internal callback disables inherited auth_request, forwards
no body/inherited headers, fixes UID/token and overwrites source/host/method/URI
with trusted original context. No forwarded identity/cache; bounded timeouts.
Do not use bypassing error_page, rewrite/internal redirects or alternate public
paths to a protected upstream. Integrator reviews full nginx -T privately: it
contains secrets. Credential rotation requires private regeneration and reload;
deleting its hash registry entry revokes independently on the real Controller.

Generate everything before exclusive mkdir; never overwrite an existing target
or recursively clean partial output. Retain partial private state on I/O failure.
No adversarial privileged operator/filesystem race or production TLS claim.

Acceptance: CLI real-filesystem RED→GREEN; installed empty-PATH binary produces
includes loaded by real Nginx against actual packaged Controller/API and a real
HTTP app, source Grant allow/revoke, method/host/path/source/header/caller isolation
and callback outage. IPv4/IPv6 callback generation/real native execution. Full Go,
race,vet,build and locked Web/native CI; one whole-branch independent review.
No Agent/kernel behavior changes; installed Gitea+Agent conjunction is next.

Context map: internal/cli/config.go assembles commands; new nginx_init{,_test}.go
owns input validation/rendering/private writes. web/e2e/ingress.mjs owns real
Nginx/app fixture, install.mjs consumes packaged CLI; new installed-ingress.mjs
owns real publication/revocation acceptance. CI install adds actual nginx package.
deploy/controller README translations are shipped by existing bundle script.
docs/HTTP_AUTHORIZATION translations describe the unchanged runtime trust boundary.

中文：新增离线私有片段生成器，仅支持初始化命令生成的独立64位十六进制入口密钥；
不创建授权、不修改/重载入口服务。使用真实安装包、Controller和Nginx验收，保持
现有入口承担TLS与业务转发；后续再把交付包接入真实Agent/Gitea联合验收。

Self-review: bounded token subset explicit; no policy schema/domain change; both
IP families, multiple routes and private partial failure are covered. User's
autonomous mandate waives routine design/plan/push/merge confirmation pauses.
