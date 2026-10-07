# PicketX Web

React + TypeScript + Vite, Tailwind CSS, shadcn/ui and i18next. Use Node.js 24 LTS and npm.

```bash
npm ci
npm run dev
npm run typecheck
npm run lint
npm run build
```

This is a bilingual development starting page, not an implemented access portal. English is the default. The language buttons also update the document language. Go and Web builds are independent; start the Controller separately from the repository root.

## UI conventions

- `src/components/ui`: official shadcn components, kept close to upstream.
- `src/lib`: shared utilities and i18next configuration.
- `src/locales`: matching English and Simplified Chinese translation keys.
- Add `src/components/shared` and `src/features` when their first real components are implemented.
- No TanStack Query, router, form library or polling abstraction is required for this static scaffold. Add them only when a feature needs them and consistent with the architecture.

The initial components use the official Radix/new-york-v4 implementation with neutral theme tokens. They were manually installed from upstream because the CLI initialization service was unavailable in the build environment. `components.json` supports normal CLI component additions. This is an initial implementation choice, not a requirement that future components use a second UI library.

Before upgrades, commit current work and inspect changes:

```bash
npx shadcn@latest add button --dry-run
npx shadcn@latest add button --diff
# Only after reviewing the changes; overwrites local component edits:
npx shadcn@latest add button --overwrite
```

npm dependency updates and copied component-source updates are separate. Preserve `package-lock.json`. See [third-party notices](../../THIRD_PARTY_NOTICES.md) for upstream licensing.

## 中文说明

这是中英文开发起始页，尚未实现权限门户。使用 npm，Go 后端与 Web 分别启动和构建。官方组件尽量不改内部实现，后续分页/表格组合放入 `components/shared`，申请/审批业务放入 `features`。更新时区分 npm 依赖和组件源码，覆盖前先审查差异。
