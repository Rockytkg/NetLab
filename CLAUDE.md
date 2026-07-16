# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NetLab is a Network Management Center for real network operations. It uses a React frontend and Go backend to manage real routers, switches, firewalls, load balancers, and other infrastructure devices. The product direction is a comprehensive operations platform with device inventory, SNMP monitoring, Syslog aggregation, RADIUS authentication/auditing, alert policies, and operational dashboards.

| Layer | Stack | Directory |
|-------|-------|-----------|
| Frontend | React 19 · TypeScript 6 · Ant Design 6.x · Vite 8 · Zustand 5 · React Router 7 · i18next | `NetLab-frontend/` |
| Backend | Go 1.25 · Gin · GORM · PostgreSQL · Redis · JWT · Zap | `NetLab-backend/` |

**Detailed frontend conventions** (component patterns, CSS architecture, i18n rules, design tokens) are in `NetLab-frontend/CLAUDE.md`. Read that file before working on any frontend code.

## Common Commands

### Frontend (`NetLab-frontend/`)

```bash
pnpm dev              # Start Vite dev server (default: http://localhost:5173)
pnpm build            # Production build
pnpm lint             # oxlint (zero warnings required)
pnpm preview          # Preview production build
pnpm i18n:check       # Run i18n audit script
pnpm check            # Full check: i18n + lint + build
```

### Backend (`NetLab-backend/`)

```bash
make build            # go build -o bin/netlab-server .
make run              # Build + run
make dev              # Hot-reload via air
make test             # go test ./... -v -cover
make test-race        # go test ./... -v -race
make lint             # golangci-lint run ./...
make swagger          # Generate Swagger docs from annotations
make docker-up        # Start PostgreSQL + Redis via docker-compose
make docker-down      # Stop infrastructure containers
make migrate          # Apply initial SQL migration (manual psql)
```

## Repository Structure

```
NetLab/
├── CLAUDE.md                    # This file (root-level guidance)
├── NetLab-frontend/             # React SPA
│   ├── CLAUDE.md                # Frontend development constitution (READ FIRST)
│   ├── src/
│   │   ├── components/          # layout/, auth/, common/
│   │   ├── pages/               # login/, dashboard/, device-groups/, devices/, observability/, settings/, etc.
│   │   ├── router/              # Single-file route config
│   │   ├── stores/              # Zustand: appStore, authStore, operationsStore
│   │   ├── services/            # Axios instance + API service objects
│   │   ├── hooks/               # useAuth, usePasskey, useI18n, useResolvedTheme
│   │   ├── i18n/                # i18next init + zh-CN/en-US locale JSONs
│   │   ├── types/               # TypeScript DTOs
│   │   └── utils/               # crypto.ts, token.ts, constants.ts, i18n-bridge.ts
│   └── docs/                    # api-for-ai-agents.md, ui-redesign-proposal.md
└── NetLab-backend/              # Go API server
    ├── main.go                  # Entry point: config → DB/Redis → repos → services → handlers → router
    ├── config/config.go         # Viper-based env config (all structs + Load())
    ├── internal/
    │   ├── router/router.go     # Gin route setup with rate-limited endpoint groups
    │   ├── middleware/          # auth (JWT), cors, crypto (AES decrypt), i18n, ratelimit, recovery, requestid, signature
    │   ├── handler/auth/        # HTTP handlers (auth_handler.go)
    │   ├── service/auth/        # Business logic: auth, crypto, oauth, passkey, token services
    │   ├── repository/          # Data access: user, token, passkey, config repos (GORM + Redis)
    │   ├── model/               # GORM models: user, token, passkey, config
    │   ├── dto/                 # request/response DTOs
    │   └── database/            # PostgreSQL + Redis connection setup + auto-migration
    ├── pkg/
    │   ├── jwt/jwt.go           # JWT manager (access + refresh tokens, blacklist interface)
    │   ├── captcha/             # Math captcha generation with Redis store
    │   ├── crypto/              # AES-GCM, HMAC-SHA256, hash utilities
    │   ├── email/smtp.go        # SMTP email sender
    │   ├── response/response.go # Standardized API response envelope
    │   └── apperrors/errors.go  # Typed application errors with i18n codes
    ├── migrations/              # SQL migration files
    └── docker-compose.yml       # PostgreSQL 16 + Redis 7
```

## Backend Architecture

### Layered Architecture (top → bottom)

```
Handler (HTTP) → Service (business logic) → Repository (data access) → DB/Redis
```

- **Handlers** parse requests, call services, return responses. No business logic.
- **Services** contain all business logic, orchestrate repos and external services.
- **Repositories** encapsulate GORM and Redis operations. One repo per aggregate root.
- **Models** are GORM entities with struct tags for both DB columns and JSON serialization.

### API Response Envelope

All endpoints return `{ code: number, data: T, message: string }`. Success codes: `0` or `200`. The frontend Axios interceptor auto-unwraps this — service functions receive `data` directly.

### Authentication Flow

1. **Pre-shared key endpoints** (`/auth/login`, `/auth/register`, `/auth/reset-password`): Client encrypts sensitive fields (password) with AES-256-CBC derived from `AUTH_PRESHARED_KEY`, signs with HMAC-SHA256 using `AUTH_SIGNATURE_KEY` + `AUTH_SIGNATURE_SALT`. Backend middleware (`Crypto` + `Signature`) decrypts and verifies before the handler runs.

2. **Public endpoints** (`/auth/refresh`, `/auth/captcha`, `/auth/send-code`, passkey/OAuth flows): Optional JWT auth — attaches user info if a valid token is present but doesn't reject unauthenticated requests.

3. **Authenticated endpoints** (`/auth/userinfo`, `/auth/logout`, passkey registration): `RequireAuth` middleware enforces valid JWT + blacklist check.

4. **Token refresh**: Access tokens expire in 15 min, refresh tokens in 7 days. The frontend proactively refreshes 5 min before expiry and retries on 401 with a queue to prevent concurrent refresh storms.

### Rate Limiting Tiers

| Tier | Limit | Endpoints |
|------|-------|-----------|
| Very strict | 3 req/min per IP | `/auth/send-code` |
| Strict | 5 req/min per IP | `/auth/login`, `/auth/reset-password`, `/auth/refresh`, passkey verify |
| Moderate | 15 req/min per IP | `/auth/register`, `/auth/captcha`, passkey/OAuth, `/auth/config` |
| Standard | 60 req/min per IP | `/auth/userinfo`, `/auth/logout`, passkey registration |
| Global | 100 req/min per IP | All routes (applied before endpoint-specific limits) |

### Middleware Chain (execution order)

```
RequestID → CORS → Recovery → I18N → GlobalRateLimit → [OptionalAuth | RequireAuth] → [Crypto] → [Signature] → [EndpointRateLimit] → Handler
```

## Frontend-Backend Contract

### Pre-Shared Keys (must match)

Three environment variables must be identical between frontend and backend:

| Frontend (.env.local) | Backend (.env) | Purpose |
|------------------------|----------------|---------|
| `VITE_AUTH_PRESHARED_KEY` | `AUTH_PRESHARED_KEY` | AES-256-CBC encryption of password fields |
| `VITE_AUTH_SIGNATURE_KEY` | `AUTH_SIGNATURE_KEY` | HMAC-SHA256 request signing |
| `VITE_AUTH_SIGNATURE_SALT` | `AUTH_SIGNATURE_SALT` | Signature payload salt |

**Key derivation (frontend):** `SHA256(presharedKey)` → AES key; `SHA256(presharedKey + ":iv")` → first 16 bytes as IV. The backend `CryptoService` must use the same derivation.

### API Base URL

The frontend proxies `/api` to the backend. Set `VITE_API_BASE_URL` in `.env.local` to the backend address (default: `http://localhost:8080/api`).

## Infrastructure

```bash
# Start PostgreSQL 16 + Redis 7
cd NetLab-backend && make docker-up

# Start backend (reads .env)
cd NetLab-backend && make dev

# Start frontend (reads .env.local)
cd NetLab-frontend && pnpm dev
```

PostgreSQL runs on port 5432 (user: `netlab`, db: `netlab`). Redis runs on port 6379. In `debug` mode, the backend auto-migrates GORM models and seeds default OAuth configs.

## Phase Roadmap

| Phase | Status | Scope |
|-------|--------|-------|
| Phase 1 | ✅ Complete | Layout shell, theme, routing, i18n, auth, dashboard device-group list, placeholder pages |
| Phase 2 | 🔲 Planned | Device inventory, site/group management, device details, real network topology view |
| Phase 3 | 🔲 Planned | SNMP polling, metric trends, interface monitoring, Syslog ingestion and search |
| Phase 4 | 🔲 Planned | RADIUS authentication/auditing, alert policies, notification workflows, responsive operations workspace |
