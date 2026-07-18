[根目录](../CLAUDE.md) > **NetLab-frontend**

# NetLab Frontend -- Development Constitution

> **Product**: 网络管理中心前端  
> **Domain**: 真实网络设备纳管、SNMP 监控、Syslog 日志、RADIUS 认证审计、告警与运维模板  
> **Stack**: React 19 · TypeScript 6 · Ant Design 6.x · Vite 8 · Zustand 5 · React Router 7 · i18next  
> **Design Doc**: `docs/ui-redesign-proposal.md`  
> **Last Updated**: 2026-07-19

---

## 1. Product Architecture

NetLab is no longer modeled as a simulator/lab product. Frontend architecture must use the following operations-domain model:

| Domain | Primary Route | Source Area |
|--------|---------------|-------------|
| Operations overview | `/dashboard` | `src/pages/dashboard/` |
| System/account settings | `/settings/**` | `src/pages/settings/` |
| Login audit (Phase 1.5) | `/settings/login-logs` | `src/pages/settings/login-logs/` |

Planned operations domains (device groups, device inventory, topology, observability, operations templates) had placeholder pages that were removed in the 2026-07-18 cleanup; they will be rebuilt with real data sources in Phase 2/3 under `/device-groups`, `/device-library`, `/devices/:deviceId/topology`, `/observability`, and `/operations-templates`.

Legacy `/labs`, `/lab/:id`, `/monitor`, and `/templates` routes may exist only as redirects. Do not add new feature code under those legacy concepts.

---

## 2. Design Tokens

Every visual decision must reference Ant Design tokens. Avoid hardcoded colors, font sizes, and spacing in component code unless representing an external data color or an existing icon asset.

| Category | Token Source |
|----------|--------------|
| Colors | `token.color*` or `var(--ant-color-*)` |
| Typography | `token.fontSize*`, `token.fontFamily` |
| Spacing | `token.padding*`, `token.margin*` |
| Radius | `token.borderRadius`, `token.borderRadiusLG`, `token.borderRadiusSM` |
| Motion | `token.motionDuration*` |

Use Ant Design layout primitives first: `Layout`, `Flex`, `Grid.useBreakpoint`, `Space`, `Card`, `Table`, `Tabs`, `Drawer`, `Form`, `Descriptions`, `Statistic`, `Alert`, and `Result`.

---

## 2.1 CSS Architecture

`src/index.css` is reserved for global reset and base document styles only: box sizing, `html/body/#root`, default font stack, and browser-level normalization.

Component or page-specific styles must live under `src/assets/css/` and be imported by the component or page entry that uses those classes:

| Style File | Owner |
|------------|-------|
| `src/assets/css/layout.css` | `components/layout/**` shell, header, sidebar, footer responsive helpers |
| `src/assets/css/login.css` | `pages/login/**` auth and login surfaces |
| `src/assets/css/settings.css` | `pages/settings/**` |

Rules:

- Do not add page or component selectors to `src/index.css`.
- Add a new CSS file under `src/assets/css/` when a new page or shared component family needs custom classes.
- Import CSS at the nearest stable owner entry, for example a page `index.tsx` or a layout component, instead of relying on unrelated global imports.
- Keep selectors scoped by the existing `netlab-*` class prefix and continue to use Ant Design CSS variables such as `var(--ant-color-*)`.

---

## 3. Development Workflow

Every feature or refactoring task follows this order:

```
A1. Types in src/types/
A2. API services in src/services/
A3. Zustand stores in src/stores/
A4. i18n keys in both zh-CN and en-US
B1. Pages/components
B2. Router and navigation registration
C1. pnpm run i18n:check
C2. pnpm lint
C3. pnpm build
```

Do not leave warnings for a later pass.

---

## 4. i18n Hygiene

Active namespaces:

| Namespace | Purpose |
|-----------|---------|
| `common` | Shared UI, errors, buttons, global settings |
| `login` | Login, register, forgot-password, passkey, OAuth, 2FA |
| `menu` | Navigation labels and route menu text |
| `operations` | Device groups, inventory, topology, monitoring, Syslog, RADIUS, alerts, templates |
| `settings` | System and account settings, login logs, roles |

Rules:

- No hardcoded user-facing strings in TS/TSX.
- Add zh-CN and en-US values in the same change.
- Use `t('operations:key')` for all operations-domain UI.
- Keep `src/types/i18n.ts` and `src/i18n/index.ts` synchronized with locale files.

---

## 5. File Organization

```
src/
├── assets/
│   └── css/              # Page/component-owned CSS imported by the nearest owner
├── components/
│   ├── layout/
│   ├── auth/
│   └── common/
├── pages/
│   ├── login/
│   ├── dashboard/
│   ├── account/
│   ├── settings/
│   │   ├── account/       # Profile, passkey, OAuth, 2FA, email, password panels
│   │   ├── panels/        # Beian, Security, SMTP, OAuth admin panels
│   │   ├── users/         # User management page
│   │   ├── roles/         # RBAC role management page
│   │   ├── login-logs/    # Login audit log page (NEW)
│   │   ├── components/    # Shared settings components
│   │   ├── context.ts
│   │   └── index.tsx
│   └── error/
├── router/
├── stores/
│   ├── appStore.ts
│   ├── authStore.ts
│   └── operationsStore.ts
├── services/
│   ├── request.ts         # Axios instance with interceptors
│   ├── auth.ts            # Auth API (login, register, etc.)
│   ├── authSecurity.ts    # Security-related auth APIs
│   ├── account.ts         # Account management APIs
│   ├── admin.ts           # System settings + user admin APIs
│   ├── rbac.ts            # RBAC APIs
│   └── log.ts             # Login log APIs (NEW)
├── hooks/
│   ├── useAuth.ts
│   ├── usePasskey.ts
│   ├── useI18n.ts
│   ├── useResolvedTheme.ts
│   └── usePermission.ts
├── types/
│   ├── auth.ts
│   ├── api.ts
│   ├── i18n.ts
│   ├── operations.ts
│   ├── settings.ts
│   └── log.ts             # Login log DTOs (NEW)
├── utils/
│   ├── crypto.ts          # HMAC-SHA256 signing
│   ├── auth-flow.ts       # Token refresh, retry queue
│   ├── auth-normalize.ts  # Auth data normalization
│   ├── password-strength.ts
│   ├── i18n-bridge.ts
│   ├── message-bridge.ts
│   ├── avatar.ts
│   ├── constants.ts
│   ├── fingerprint.ts     # Browser fingerprint via FingerprintJS (NEW)
│   ├── clientInfo.ts      # OS/browser detection via UA-Parser (NEW)
│   └── xlsx.ts           # Browser-side Excel export (NEW)
└── i18n/locales/{zh-CN,en-US}/
    ├── common.json
    ├── login.json
    ├── menu.json
    ├── operations.json
    └── settings.json
```

---

## 6. State Management

| Store | Persist Key | Scope |
|-------|-------------|-------|
| `appStore` | `netlab-app` | locale, sidebarCollapsed, themeMode |
| `authStore` | `netlab-auth` | accessToken, refreshToken, userInfo, securityActions |
| `operationsStore` | memory | deviceGroups, devices, filters, active device/group |

Operations data is server-owned. Keep it in memory unless there is an explicit UX reason to persist a draft.

---

## 7. Routing Rules

Use these as canonical route contracts (currently registered):

```
/dashboard
/settings              # 重定向到 /settings/beian
/settings/beian
/settings/security
/settings/smtp
/settings/oauth
/settings/profile
/settings/users
/settings/roles
/settings/login-logs   # (NEW) 登录审计日志
```

Reserved for Phase 2+ (re-add pages together with real data sources; do not repurpose these paths):

```
/device-groups
/device-library
/devices/:deviceId/topology
/observability
/operations-templates
/operations-templates/upload
/operations-templates/installed
```

Legacy routes are redirects only:

```
/labs → /device-groups
/lab/:deviceId → /devices/:deviceId/topology
/lab/:deviceId/monitor → /observability
/monitor → /observability
/templates/** → /operations-templates/**
```

---

## 8. Client Environment Detection (Login Audit)

The frontend collects client environment info for login audit logging. Two utilities initialize at app startup:

| Utility | Library | Data Collected | Headers Sent |
|---------|---------|---------------|-------------|
| `utils/fingerprint.ts` | FingerprintJS | Browser fingerprint (visitor ID) | `X-Browser-Fingerprint` |
| `utils/clientInfo.ts` | ua-parser-js with Client Hints | OS name + version + architecture, Browser name + major version | `X-Client-OS`, `X-Client-Browser` |

Both utilities are:
- Module-level singletons (init once, cache result)
- Fail-silent (errors never propagate to calling code)
- Used by the `AuthHandler` login flows to record login audit logs (back-end stores in `nb_login_logs`)

---

## 9. Quality Checklist

Before considering work complete:

```
[ ] pnpm run i18n:check
[ ] pnpm lint
[ ] pnpm build
[ ] New text uses i18n in both languages
[ ] New operations UI uses operations-domain naming
[ ] New routes are registered in router, SideMenu, and HeaderBar title mapping
[ ] Legacy lab/simulator wording is not introduced
[ ] New utils have proper error handling (fail-silent pattern for capability-optional features)
```

---

## 10. Roadmap

| Phase | Status | Scope |
|-------|--------|-------|
| Phase 1 | Done | Layout shell, auth, theme, i18n, operations information architecture |
| Phase 1.5 | Done | Login audit log page, browser fingerprint + client info detection, browser-side Excel export |
| Phase 2 | Planned | Device inventory, site/group management, device onboarding |
| Phase 3 | Planned | SNMP polling, interface metrics, Syslog ingestion/search |
| Phase 4 | Planned | RADIUS audit, alert policies, notification workflows, responsive operations workspace |

---

## 11. Changelog

| Date | Change |
|------|--------|
| 2026-07-19 | Added login audit log page (pages/settings/login-logs), services/log.ts, types/log.ts; added client environment detection utils (utils/fingerprint.ts using FingerprintJS, utils/clientInfo.ts using ua-parser-js); added browser-side xlsx export utility (utils/xlsx.ts); added /settings/login-logs route; added loginLogs menu item and i18n keys (menu + settings namespaces); updated file organization chart |
| 2026-07-18 | Removed unrouted placeholder pages (device-groups, device-library, devices/topology, observability, operations-templates, help) plus their dedicated CSS (device-groups.css, topology.css) and dead types (OperationsFilter, DEVICE_STATUS_CONFIG); routing/structure docs now reflect current state |
| 2026-07-18 | Added breadcrumb navigation to root CLAUDE.md; added changelog section |
| 2026-07-14 | Original frontend development constitution created |
