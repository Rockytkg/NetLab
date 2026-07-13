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

## Testing Guidelines

Backend tests use Go’s built-in test framework and live beside package code, such as `pkg/captcha/captcha_test.go` and `pkg/crypto/*_test.go`. Add table-driven tests for new backend logic where practical. The frontend currently relies on `pnpm check`; add focused tests only when introducing a frontend test framework.

## Commit & Pull Request Guidelines

Recent history uses concise Conventional Commit-style subjects, for example `chore: 初始化 NetLab 项目并修复后台布局问题`. Prefer `feat:`, `fix:`, `chore:`, `docs:`, or `refactor:` plus a short summary.

Pull requests should include a clear description, linked issue when available, screenshots for UI changes, migration or environment notes, and the commands run for verification.

## Security & Configuration Tips

Do not commit `.env` or `.env.local`. Keep frontend and backend pre-shared auth/signature keys synchronized between `NetLab-frontend/.env.local` and `NetLab-backend/.env`. Review `CLAUDE.md` and `NetLab-frontend/CLAUDE.md` before larger architecture or UI changes.
