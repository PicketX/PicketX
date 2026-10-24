# Controller local development

English | [简体中文](LOCAL_DEVELOPMENT.zh-CN.md)

Go backend and Web run independently. Normal Controller startup initializes its own SQLite database and first administrator; there is no preparation script or development CLI. PicketXD/Nginx are not prerequisites.

## Start

Use Go as required by `go.mod`, CGO/a native C compiler, Node24+ and npm. From the repository root, start the backend:

```sh
go run ./cmd/picketx-controller
```

In another terminal:

```sh
cd web
npm ci
npm run dev
```

Open **http://127.0.0.1:5173/console/**, sign in as **admin** using the generated private `var/controller/initial-admin-password` file. Logs identify the file, never the password. Change the initial password using account management; the plaintext file is only the original provisioning copy and can then be removed.

No config file means the complete single-instance Controller: SQLite and database-managed accounts, HTTP8080, Agent gRPC9090 and local decisions9091. Agent/caller credential sets start empty and authorize nobody; configure them explicitly for direct testing. Agent absence does not block readiness and readiness does not prove enforcement. State stays in ignored `var/controller/`; do not commit or share it.

Existing `-config` / `PICKETX_CONFIG` deployments retain their configuration semantics, including file authority. Clear inherited `PICKETX_CONFIG` when intending to use normal no-config defaults. Explicit flags override present environment, then selected configuration/defaults. A wrong/unreadable config is an error, not permission to fall back to a new installation.

`browser.origin` / `PICKETX_BROWSER_ORIGIN` is optional. When set, exact Host/Origin restrictions apply. When omitted or empty, writes must match the request Host and actual HTTP/TLS scheme, with CSRF/session protections retained. Forwarded headers are not trusted; configure the public HTTPS origin behind TLS termination and always configure it for OIDC. Cookie Secure follows explicit HTTPS or actual TLS.

## First startup and restart

Before creating a new default database, Controller writes0600 initial password/recovery files and independent credential files in0700 private parents. Existing files are not overwritten. The database and account audit use existing migrations and account APIs. `initial-recovery-key` retains the independent management key required by the existing recovery contract; store it privately even after changing the admin password.

The initial `admin` is created only if the database identity directory has never committed an account operation (revision0). Restart preserves passwords, account changes and disabled state; missing password files never trigger account reset. Browser sessions expire on restart. Interrupted initial setup can resume using the pending password file. If that file is lost before account creation, or a required credential file is lost for an existing database, restore the private input rather than expecting automatic regeneration. Preparation failure may leave private files for inspection; no automatic directory reset is performed.

## IDE and Vite

In your IDE, directly launch Go package `./cmd/picketx-controller`, use the repository root as working directory, and set `CGO_ENABLED=1` if needed. No startup arguments or initializer are needed. Run Web separately using its existing npm script. No IDE-specific configuration files are required.

Vite keeps its ordinary `/console/api` proxy to8080, preserving public Host. Port5173 is strict. No browser Origin setting is needed by default; if you pin one explicitly, use its exact browser host/port. Update the proxy if the backend port changes. Stop the previous backend before launching another from the IDE.

Vite handles Web hot updates; the developer/IDE restarts Go. HTTP read/write bounds remain10seconds and Web requests15seconds. Long breakpoints can time out. Resume and inspect current state/audit before retrying an interrupted write; an error is not proof of rollback.

## Built Web / deployment

Stop the previous backend, then:

```sh
cd web
npm run build
cd ..
go build -o var/picketx-controller ./cmd/picketx-controller
PICKETX_BROWSER_ASSETS=web/dist var/picketx-controller
```

Open **http://127.0.0.1:8080/console/**. Without an explicit browser Origin, requests use their actual Host and HTTP/TLS scheme. State and credentials are reused; Vite can stop. The same normal bootstrap applies outside development; production may instead use explicitly provisioned runtime configuration and existing HTTPS ingress. A native build does not establish Linux fully-static release acceptance.

## Troubleshooting

Use `curl -i http://127.0.0.1:8080/readyz`, existing Controller/Vite logs and the browser Network panel. Connection failure requires checking the process/port;404 can indicate a different explicit configuration;403 requires matching Host/Origin;503 requires inspecting configured credentials/readiness. No extra diagnostic tool is provided.

Normal starts never reset state. To deliberately replace a disposable environment, stop the backend and move its entire private state to a private backup location. Do not use this to silently switch a deployed database's authority. Use existing Go/Web checks from the README; on macOS existing Unix-socket tests may require `TMPDIR=/private/tmp go test ./...` to avoid long/symlinked temporary paths.
