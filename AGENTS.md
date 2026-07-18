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
- Frontend `src/pages/settings/roles/index.tsx` (`ADMIN_RESOURCES` / `ACCOUNT_RESOURCES`): place the resource in the permission tree so it mirrors the sidebar menu hierarchy exactly — 系统管理 (administration) > 系统设置 (setting) / 用户管理 (user) / 角色管理 (rbac) / 登录日志 (log). Every future permission must follow this sidebar-aligned grouping.
- Frontend `src/components/layout/SideMenu.tsx`: gate each menu item individually with `can('<resource>.<action>')` and show a menu group only when at least one of its child permissions is present.
- New protected pages render a 403 `Result` when the user lacks the read permission (see `settings/users`, `settings/login-logs`).

## Testing Guidelines

Backend tests use Go’s built-in test framework and live beside package code, such as `pkg/captcha/captcha_test.go` and `pkg/crypto/*_test.go`. Add table-driven tests for new backend logic where practical. The frontend currently relies on `pnpm check`; add focused tests only when introducing a frontend test framework.

## Commit & Pull Request Guidelines

Recent history uses concise Conventional Commit-style subjects, for example `chore: 初始化 NetLab 项目并修复后台布局问题`. Prefer `feat:`, `fix:`, `chore:`, `docs:`, or `refactor:` plus a short summary.

Pull requests should include a clear description, linked issue when available, screenshots for UI changes, migration or environment notes, and the commands run for verification.

## Security & Configuration Tips

Do not commit `.env` or `.env.local`. Keep frontend and backend pre-shared auth/signature keys synchronized between `NetLab-frontend/.env.local` and `NetLab-backend/.env`. Review `CLAUDE.md` and `NetLab-frontend/CLAUDE.md` before larger architecture or UI changes.
