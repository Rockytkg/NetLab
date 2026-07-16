# NetLab Frontend — Development Constitution

> **Product**: 网络管理中心前端  
> **Domain**: 真实网络设备纳管、SNMP 监控、Syslog 日志、RADIUS 认证审计、告警与运维模板  
> **Stack**: React 19 · TypeScript 6 · Ant Design 6.x · Vite 8 · Zustand 5 · React Router 7 · i18next  
> **Design Doc**: `docs/ui-redesign-proposal.md`  
> **Last Updated**: 2026-07-14

---

## 1. Product Architecture

NetLab is no longer modeled as a simulator/lab product. Frontend architecture must use the following operations-domain model:

| Domain | Primary Route | Source Area |
|--------|---------------|-------------|
| Operations overview | `/dashboard` | `src/pages/dashboard/` |
| Device groups | `/device-groups` | `src/pages/device-groups/` |
| Device inventory | `/device-library` | `src/pages/device-library/` |
| Device topology | `/devices/:deviceId/topology` | `src/pages/devices/` |
| Monitoring and logs | `/observability` | `src/pages/observability/` |
| Operations templates | `/operations-templates` | `src/pages/operations-templates/` |
| System/account settings | `/settings/**` | `src/pages/settings/` |

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
| `src/assets/css/device-groups.css` | `pages/device-groups/` |
| `src/assets/css/settings.css` | `pages/settings/**` |
| `src/assets/css/topology.css` | `pages/devices/**` topology and device panel surfaces |

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
| `settings` | System and account settings |

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
│   ├── device-groups/
│   ├── device-library/
│   ├── devices/
│   ├── observability/
│   ├── operations-templates/
│   ├── settings/
│   ├── help/
│   └── error/
├── router/
├── stores/
│   ├── appStore.ts
│   ├── authStore.ts
│   └── operationsStore.ts
├── services/
├── hooks/
├── types/
│   ├── auth.ts
│   ├── api.ts
│   ├── i18n.ts
│   ├── operations.ts
│   └── settings.ts
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

Use these as canonical route contracts:

```
/dashboard
/device-groups
/device-library
/devices/:deviceId/topology
/observability
/operations-templates
/operations-templates/upload
/operations-templates/installed
/settings
/settings/profile
/settings/users
/help
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

## 8. Quality Checklist

Before considering work complete:

```
[ ] pnpm run i18n:check
[ ] pnpm lint
[ ] pnpm build
[ ] New text uses i18n in both languages
[ ] New operations UI uses operations-domain naming
[ ] New routes are registered in router, SideMenu, and HeaderBar title mapping
[ ] Legacy lab/simulator wording is not introduced
```

---

## 9. Roadmap

| Phase | Status | Scope |
|-------|--------|-------|
| Phase 1 | ✅ Complete | Layout shell, auth, theme, i18n, operations information architecture |
| Phase 2 | 🔲 Planned | Device inventory, site/group management, device onboarding |
| Phase 3 | 🔲 Planned | SNMP polling, interface metrics, Syslog ingestion/search |
| Phase 4 | 🔲 Planned | RADIUS audit, alert policies, notification workflows, responsive operations workspace |
