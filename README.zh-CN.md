# PicketX

[English (default)](README.md) | 简体中文

**简单易用、无需专用客户端的按需访问授权系统。**

PicketX 通过自助申请、策略和审批，为受保护资源提供有期限的访问权限。IP/CIDR 网络权限可由匹配来源的设备和应用共享，服务自身的认证仍然保留。后续身份级授权通过可信 Adapter 集成。

## 项目状态

当前仓库是**开发初始化骨架**，包含 Go Controller 存活检查和中英文 React 起始页。尚未实现认证、审批、Grant、数据库、Agent 同步或执行功能，不能用于保护实际资源。

控制面与 Web 位于本仓库；官方 Linux Agent `picketxd` 属于独立的 **PicketXD** 仓库。本次初始化未创建或修改该仓库。

## 本地开发

需要 Go 1.27+、Node.js 24 LTS 和 npm。

在仓库根目录启动 Controller：

```bash
go run ./cmd/picketx-controller
```

默认监听 `127.0.0.1:8080`，通过 `-listen` 修改地址。`GET /healthz` 仅表示进程存活，不表示授权功能已就绪。

另开终端启动 Web：

```bash
cd apps/web
npm ci
npm run dev
```

访问 Vite 输出的地址。前后端独立构建，Vite 将 `/healthz` 代理到本地 Controller。Controller 暂未托管 Web 静态资源。

## 验证

```bash
go test ./...
go vet ./...
go build ./...

cd apps/web
npm run typecheck
npm run lint
npm run build
```

## 目录与设计

| 路径 | 用途 |
| --- | --- |
| `cmd/picketx-controller` | 本地开发服务入口 |
| `internal/controller` | HTTP Handler 与测试 |
| `apps/web` | React、TypeScript、Vite、shadcn/ui、Tailwind CSS、i18next |
| `docs` | 最新中英文产品、架构与配置模型基线 |

Core 与公开协议包在实现首个契约时引入。参阅[产品设计](docs/PRODUCT_DESIGN.zh-CN.md)、[架构设计](docs/ARCHITECTURE.zh-CN.md)、[配置模型](docs/CONFIGURATION_MODEL.zh-CN.md)和 [Web 开发说明](apps/web/README.md)。

## 许可证

项目代码使用 [Apache-2.0](LICENSE)。复制的第三方组件保留原有声明，见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。PicketXD 采用独立的 GPL-3.0-or-later/OEM 授权设计。
