# Repository Guidelines

## Project Structure & Module Organization

NetLab is split into a React frontend and Go backend. Frontend code lives in `NetLab-frontend/src/`: `components/` for shared UI, `pages/` for route screens, `router/` for route registration, `stores/` for Zustand state, `services/` for API calls, `hooks/`, `types/`, `utils/`, and `i18n/locales/{zh-CN,en-US}/`. Static assets are in `NetLab-frontend/public/`.

Backend code lives in `NetLab-backend/`. `main.go` wires configuration, database clients, repositories, services, handlers, and routes. Domain code is under `internal/` (`handler/`, `service/`, `repository/`, `model/`, `middleware/`, `dto/`, `database/`). Reusable packages are under `pkg/`, SQL migrations under `migrations/`.

## Build, Test, and Development Commands

Frontend commands run from `NetLab-frontend/`:

- `pnpm dev`: start the Vite development server.
- `pnpm build`: create a production build.
- `pnpm lint`: run `oxlint`.
- `pnpm i18n:check`: audit translation keys.
- `pnpm check`: run i18n audit, lint, and build.

Backend commands run from `NetLab-backend/`:

- `make docker-up`: start PostgreSQL and Redis.
- `make dev`: run the API with `air` hot reload.
- `make build`: compile `bin/netlab-server`.
- `make test`: run `go test ./... -v -cover`.
- `make test-race`: run race-enabled tests.
- `make lint`: run `golangci-lint`.

## Coding Style & Naming Conventions

Use TypeScript with React function components. Keep one component per file, use PascalCase for component files, and keep feature entry points as `index.tsx`. User-facing frontend strings must use `t('namespace:key')` with matching `zh-CN` and `en-US` entries. Prefer Ant Design tokens and existing CSS class prefixes.

Go code must be `gofmt`/`go fmt` formatted. Keep HTTP parsing in handlers, business logic in services, and GORM/Redis access in repositories. Test files use Go’s standard `*_test.go` naming.

## Permissions & Menu Conventions

RBAC permissions use `resource.action` keys (e.g. `log.read`). When adding a new permission, update all of these together:

- Backend `internal/permission/permission.go`: add the constant and a `Catalog` entry (synced into the database at startup; builtin superadmin/admin get all permissions automatically).
- Frontend `src/i18n/locales/{zh-CN,en-US}/settings.json`: add a `roles.permissionNames.<resource>.<action>` entry in both locales.
- Frontend `src/pages/settings/roles/index.tsx` (`ADMIN_RESOURCES` / `BILLING_RESOURCES` / `ACCOUNT_RESOURCES`): place the resource in the permission tree so it mirrors the sidebar menu hierarchy exactly — 系统管理 (administration) > 系统设置 (setting) / 用户管理 (user) / 角色管理 (rbac) / 登录日志 (log), and 认证计费 (billing) > Radius (radius). Every future permission must follow this sidebar-aligned grouping.
- Frontend `src/components/layout/SideMenu.tsx`: gate each menu item individually with `can('<resource>.<action>')` and show a menu group only when at least one of its child permissions is present; new top-level groups must also be added to `rootSubmenuKeys`.
- New protected pages render a 403 `Result` when the user lacks the read permission (see `settings/users`, `settings/login-logs`, `billing/*`).

## RADIUS Module (认证计费)

The RADIUS server (ported from ToughRADIUS, MIT) lives in `NetLab-backend/internal/radiusd/` (protocol runtime: auth/acct pipelines, EAP, RadSec, CoA) with vendor dictionaries under `radiusd/vendors/`, plugins under `radiusd/plugins/`, and store interfaces under `radiusd/repository/`. GORM models (`nb_radius_*` tables, incl. `nb_radius_certs` for managed TLS certificates) are in `internal/model/radius*.go`; management API is `/api/radius/*` gated by `radius.read` / `radius.manage`. Frontend pages live in `NetLab-frontend/src/pages/billing/` (nas / users / profiles / sessions / accounting / auth-logs / settings（服务设置，合并原系统配置与 RADIUS 配置）/ dot1x（802.1X 认证）/ bypass（免认证，IP/MAC 放行）/ certs) with the `radius` i18n namespace. The sidebar billing group is split into three collapsible submenus — 业务管理 (business pages), 认证方式 (802.1X / 免认证; reserved for future Portal auth), 服务配置 (certs / settings).

Runtime configuration is two-layered, mirroring ToughRADIUS: `RADIUS_*` env vars (`.env.example`) provide defaults, and the admin UI sections (服务设置页 基础设置/认证与会话 两个 Tab、802.1X 认证页) persist overrides as JSON blobs in `system_configs` (keys `radius.system` / `radius.server` / `radius.eap`) via `internal/service/config` (sysconfig). The effective config is recomputed on every save and applied through `internal/radiusd/manager.go` (`Manager.Apply`): listener-level changes (bind host, ports, RadSec toggle/certificates) rebuild the UDP/TLS listeners in-process, everything else (EAP method, managed-cert references, Message-Authenticator mode, reject-delay, interim interval, session-timeout cap, ignore-password) is hot-swapped via `RadiusService.UpdateConfig`. EAP/RadSec TLS material resolves per-handshake/start from `nb_radius_certs` (DB certs take precedence over the `*_CERT_FILE` paths). Managed cert private keys, NAS secrets and RADIUS user passwords are all AES-256-GCM encrypted with the same `CONFIG_ENCRYPTION_KEY` cipher.

## Testing Guidelines

Backend tests use Go’s built-in test framework and live beside package code, such as `pkg/captcha/captcha_test.go` and `pkg/crypto/*_test.go`. Add table-driven tests for new backend logic where practical. The frontend currently relies on `pnpm check`; add focused tests only when introducing a frontend test framework.

## Commit & Pull Request Guidelines

Recent history uses concise Conventional Commit-style subjects, for example `chore: 初始化 NetLab 项目并修复后台布局问题`. Prefer `feat:`, `fix:`, `chore:`, `docs:`, or `refactor:` plus a short summary.

Pull requests should include a clear description, linked issue when available, screenshots for UI changes, migration or environment notes, and the commands run for verification.

## Security & Configuration Tips

Do not commit `.env` or `.env.local`. Keep frontend and backend pre-shared auth/signature keys synchronized between `NetLab-frontend/.env.local` and `NetLab-backend/.env`. Local AI-assistant guidance such as `CLAUDE.md` is ignored by Git; consult it when it is available locally, but do not rely on it as a versioned project artifact.
