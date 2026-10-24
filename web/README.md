# PicketX Web

React + TypeScript + Vite, Tailwind CSS, shadcn/ui and i18next. Use Node.js 24 LTS and npm.

```bash
npm ci
npm run dev
npm run typecheck
npm run lint
npm run build
```

Run the independent backend with `go run ./cmd/picketx-controller` from the repository root. It prepares SQLite/admin automatically. Open `http://127.0.0.1:5173/console/`, choose username/password and use `admin` plus private `var/controller/initial-admin-password`. See [local development](../docs/LOCAL_DEVELOPMENT.md). Vite retains its ordinary API proxy and strict port5173.

This is a bilingual operator console backed by real SCS sessions and management publication/audit APIs, not a complete access portal. English is default. Use [browser deployment](../docs/BROWSER_CONSOLE.md): opt in to database authority; an explicit public origin is optional except for OIDC. Normal unconfigured startup creates the first database administrator; explicitly configured deployments retain their existing provisioning contract. Built assets use `/console/`; Vite proxies `/console/api` to127.0.0.1:8080. No API mocks/sample state or browser-stored access keys are used.

`npm run test:e2e` requires Go, a built dist and `npx playwright install --with-deps chromium`. It drives actual Chromium against a temporary real Controller/SQLite. The CI screenshot artifact contains no credential/database files.

## UI conventions

- `src/components/ui`: official shadcn components, kept close to upstream.
- `src/lib`: shared utilities and i18next configuration.
- `src/locales`: matching English and Simplified Chinese translation keys.
- Add `src/components/shared` and `src/features` when their first real components are implemented.
- The first console uses native fetch/React state and manual refresh. Add router/query/form abstractions only when a feature needs them and consistent with the architecture. Draft ETags must never advance on refresh; see the explicit conflict/reload flow.

The initial components use the official Radix/new-york-v4 implementation with neutral theme tokens. They were manually installed from upstream because the CLI initialization service was unavailable in the build environment. `components.json` supports normal CLI component additions. This is an initial implementation choice, not a requirement that future components use a second UI library.

Before upgrades, commit current work and inspect changes:

```bash
npx shadcn@latest add button --dry-run
npx shadcn@latest add button --diff
# Only after reviewing the changes; overwrites local component edits:
npx shadcn@latest add button --overwrite
```

npm dependency updates and copied component-source updates are separate. Preserve `package-lock.json`. See [third-party notices](../THIRD_PARTY_NOTICES.md) for upstream licensing.

## 中文说明

后端在根目录执行 `go run ./cmd/picketx-controller`，自动准备 SQLite/管理员。Web 独立 `npm run dev`，用户名 `admin`，密码在私有 `var/controller/initial-admin-password`。详见[本地开发](../docs/LOCAL_DEVELOPMENT.zh-CN.md)。Vite 保留普通 API 代理及严格5173端口。

这是接入真实SCS会话、配置发布与审计API的双语管理工作台，不是完整权限门户。按[部署说明](../docs/BROWSER_CONSOLE.zh-CN.md)启用数据库权威；除 OIDC 外，公开 Origin 可选；正常无配置启动自动创建首个数据库管理员，显式配置部署保留原供应契约。使用npm，Go/Web分别构建，路径/console/。草稿ETag不随刷新推进，412须显式重新载入。test:e2e使用真实Chromium→Go→SQLite，截图不含凭据/数据库文件。官方组件内部保持贴近上游，后续组合及申请/审批独立实现，保留锁文件。
