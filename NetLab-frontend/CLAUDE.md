# NetLab — Development Constitution

> **Product**: 类 EVE-NG 网络模拟器前端  
> **Stack**: React 19 · TypeScript 6 · Ant Design 6.x · Vite 8 · Zustand 5 · React Router 7 · i18next  
> **Design Doc**: `docs/ui-redesign-proposal.md`  
> **Last Updated**: 2026-07-06

---

## 1. Design Token — Immutable Globals

Every visual decision must reference Ant Design tokens. **Never hardcode hex values, font sizes, or spacing numbers** outside the ConfigProvider theme definition in `src/App.tsx`.

| Category | Token Source |
|----------|-------------|
| Colors | `var(--ant-color-*)` CSS variables |
| Typography | `token.fontSize*`, `token.fontFamily` |
| Spacing | `token.padding*`, `token.margin*` — 4px grid, multiples of 8 |
| Radius | `token.borderRadius` (6), `token.borderRadiusLG` (8), `token.borderRadiusSM` (4) |
| Motion | `token.motionDurationFast` (0.1s), `Mid` (0.2s), `Slow` (0.3s) |
| Inline styles | `const { token } = theme.useToken()` — NEVER raw values |

**Component token overrides** go in `src/App.tsx` → `ConfigProvider.theme.components`.

---

## 2. Development Workflow (Mandatory Sequence)

Every feature or refactoring task must follow this sequence. **No step may be skipped.**

### Phase A — Foundation
```
A1. Types (src/types/)
A2. API services (src/services/)  
A3. State stores (src/stores/)
A4. i18n keys — ALL namespaces, BOTH languages (zh-CN + en-US)
```

### Phase B — Implementation
```
B1. Components / Pages
B2. Router registration (if new route)
B3. Wire components together
```

### Phase C — Quality Gates (MANDATORY — run in order)
```
C1. i18n Audit    → pnpm run i18n:check  (see §3 below)
C2. Lint          → pnpm lint             (zero warnings required)
C3. Build         → pnpm build            (zero errors required)
C4. Manual Check  → pnpm dev, spot-check 3+ pages
```

**If any gate fails, fix it before proceeding. Do NOT accumulate warnings.**

---

## 3. i18n Hygiene — The Most Common Failure Point

This project uses `i18next` with 5 namespaces: `common`, `login`, `menu`, `lab`, `topology`.

### Rules

1. **ZERO hardcoded user-facing strings** — No Chinese, no English inline. Every string visible to the user goes through `t('namespace:key')`.

2. **Both languages must be updated simultaneously** — When adding a key, write `zh-CN` AND `en-US` in the same commit. Never leave one language with a stale/empty value.

3. **Error messages use i18n keys** — `request.ts` and `authStore.ts` must use `t('common:networkError')` etc., never raw strings.

4. **i18n Audit — run after every task:**
   ```bash
   # Check for hardcoded Chinese characters in TSX files
   rg '[一-鿿]+' src/ --glob '*.tsx' --glob '*.ts' \
     | grep -v 'i18n' | grep -v 'locales' | grep -v '\.test\.' \
     | grep -v '// i18n' | grep -v 'console\.'

   # Check that every t() key exists in both locale files
   # Manual: scan src/i18n/locales/zh-CN/*.json keys vs en-US/*.json keys
   ```

5. **Namespace map:**
   | Namespace | Used In | Files |
   |-----------|---------|-------|
   | `common` | Shared UI: save, cancel, delete, error messages, etc. | All pages |
   | `login` | Login, register, forgot-password, passkey, OAuth | `src/pages/login/**` |
   | `menu` | Sidebar navigation, page titles | `SideMenu.tsx`, `HeaderBar.tsx` |
   | `lab` | Lab list, lab operations | `src/pages/dashboard/**`, `src/pages/labs/**` |
   | `topology` | Canvas editor, device panel | `src/pages/lab/**` (Phase 2+) |

### i18n Cleanup Checklist
- [ ] `src/i18n/locales/zh-CN/*.json` — all keys present, no empty values
- [ ] `src/i18n/locales/en-US/*.json` — all keys present, no empty values
- [ ] `src/types/i18n.ts` — `I18nNamespace` includes all active namespaces
- [ ] `src/i18n/index.ts` — all locale JSON files imported
- [ ] Zero `rg '[一-鿿]' src/ --glob '*.tsx' | grep -v i18n | grep -v locales` results
- [ ] All `t()` calls use correct namespace prefix

---

## 4. Component Architecture

### Layout (immutable shell)
```
MainLayout
├── Sider (232px, collapsible → 80px, breakpoint="lg")
│   ├── BrandingBlock (logo + "NetLab" + "Network Simulator")
│   └── SideMenu (sectioned: Workspace / Labs / Market)
├── HeaderBar (64px, sticky)
│   ├── MobileMenuButton (.netlab-mobile-only)
│   ├── SidebarToggle (.netlab-desktop-only)
│   ├── PageTitle (dynamic, i18n-driven from route)
│   └── Search + Notifications + UserDropdown
├── Content (responsive padding: xxl=32, xl=24, <md=16)
│   └── <Outlet />
└── Footer (48px, hidden on xs)
```

### Page Pattern (every page must follow)
```tsx
export default function SomePage() {
  const { t } = useTranslation(['namespace1', 'namespace2'])
  const { token } = theme.useToken()

  return (
    <div style={{ width: '100%' }}>
      <div className="netlab-page-header">
        <div>
          <Title level={3}>{t('menu:pageTitle')}</Title>
          <Text type="secondary">{t('pageSubtitle')}</Text>
        </div>
        {/* Action buttons */}
      </div>
      {/* Page content */}
    </div>
  )
}
```

---

## 5. File Naming & Organization

```
src/
├── components/
│   ├── layout/        # MainLayout, SideMenu, HeaderBar
│   ├── auth/          # AuthGuard
│   └── common/        # Loading, shared utilities
├── pages/
│   ├── login/         # index.tsx, LoginForm.tsx, RegisterForm.tsx,
│   │                  # ForgotPasswordModal.tsx, PasskeyButton.tsx,
│   │                  # OAuthSection.tsx, OAuthCallbackPage.tsx
│   ├── dashboard/     # index.tsx (lab list)
│   ├── lab/           # editor.tsx, monitor.tsx (Phase 2)
│   ├── labs/          # index.tsx
│   ├── device-library/# index.tsx
│   ├── templates/     # index.tsx, upload.tsx, installed.tsx
│   ├── settings/      # index.tsx, profile.tsx
│   ├── help/          # index.tsx
│   └── error/         # 403.tsx, 404.tsx
├── router/            # index.tsx (single file, all routes)
├── stores/            # appStore, authStore, labStore
├── services/          # request.ts (axios), auth.ts (authApi)
├── hooks/             # useAuth, useI18n, usePasskey
├── types/             # auth.ts, api.ts, i18n.ts, lab.ts
├── i18n/
│   ├── index.ts       # i18next init
│   └── locales/{zh-CN,en-US}/
│       ├── common.json
│       ├── login.json
│       ├── menu.json
│       ├── lab.json
│       └── topology.json
├── utils/             # constants.ts, token.ts
├── App.tsx            # ConfigProvider theme root
├── index.css          # Global styles (5 sections)
└── main.tsx           # ReactDOM entry
```

**Rules:**
- One component per file
- Index file for directory re-exports
- Placeholder pages use `<Result>` with i18n `comingSoon` / `underDevelopment` keys
- New feature directories get `index.tsx` as entry

---

## 6. CSS Architecture (`src/index.css`)

Organized in 5 numbered sections:

| Section | Scope | Class Prefix |
|---------|-------|-------------|
| 1. Reset & Base | `*`, `body`, `#root` | — |
| 2. Layout Shell | Sider, Content, Footer | `.netlab-layout-*` |
| 3. Page Header | Shared page title bar | `.netlab-page-header` |
| 4. Dashboard | Lab list filter/table | `.netlab-dashboard-*` |
| 5. Topology Canvas | Editor canvas, toolbar, minimap | `.netlab-canvas-*` |
| 6. Device Panel | Bottom device drawer | `.netlab-device-*` |
| 7. Config Drawer | Right-side device config | `.netlab-config-*` |
| 8. Utilities | Visibility toggles, icon buttons | `.netlab-mobile-only`, etc. |
| 9. Responsive | 5 breakpoint tiers (xxl→xs) | Media queries |
| 10. Login | Login shell, intro, feature grid | `.netlab-login-*` |

**Responsive breakpoints (Ant Design Grid):**

| Breakpoint | Width | Behavior |
|-----------|-------|----------|
| xxl | ≥1600px | Full layout, 32px content padding |
| xl | ≥1200px | Standard, 24px padding |
| lg | ≥992px | Sider visible (default collapsed) |
| md | ≥768px | Sider→Drawer, compact footer |
| sm | ≥576px | Stacked filters, simplified header |
| xs | <576px | Hidden footer, 12px padding, read-only hint |

---

## 7. API Layer Convention

All API calls go through `src/services/request.ts` (Axios instance).

**Interceptors:**
- Request: attaches Bearer token, `Accept-Language`, `X-Request-Id`
- Response: unwraps `{ code, data, message }` envelope, auto-refreshes 401 with queue

**Service files** export named API objects:
```ts
export const authApi = {
  login(params: LoginParams): Promise<LoginResult> { ... },
  // ...
}
```

**Backend contract:** All endpoints return `{ code: 0 | 200, data: T, message: string }`.

---

## 8. State Management (Zustand)

| Store | Persist Key | Scope |
|-------|-----------|-------|
| `appStore` | `netlab-app` | locale, sidebarCollapsed |
| `authStore` | `netlab-auth` | accessToken, refreshToken, userInfo |
| `labStore` | — (memory) | labs[], filter, selectedLabIds, activeLabId |

**Rules:**
- `persist` middleware only for cross-session state (auth tokens, preferences)
- Lab/topology data stays in-memory (saved to API)
- Use `useStore.getState()` for non-React contexts (axios interceptors)
- New stores: create in `src/stores/`, import type from `src/types/`

---

## 9. Quality Checklist (Pre-Commit)

Before considering any task complete, verify:

```
[ ] pnpm lint           → 0 warnings
[ ] pnpm build          → 0 errors
[ ] i18n audit          → 0 hardcoded strings, 0 missing keys
[ ] All new text uses t('namespace:key') — both zh-CN and en-US populated
[ ] No raw colors/px values outside App.tsx theme or index.css
[ ] New routes registered in src/router/index.tsx
[ ] New pages follow netlab-page-header pattern
[ ] Responsive: check at 1920px, 1200px, 768px, 375px
[ ] No console.log left in production code
[ ] Placeholder pages use <Result> + i18n comingSoon, not hardcoded text
```

---

## 10. Phase Roadmap

| Phase | Status | Scope |
|-------|--------|-------|
| **Phase 1** | ✅ Complete | Layout skeleton, theme, routing, i18n, login (passkey/OAuth/register/forgot-password/captcha), dashboard lab list, placeholder pages |
| **Phase 2** | 🔲 Planned | AntV X6 topology editor, device panel, device config drawer, lab editor |
| **Phase 3** | 🔲 Planned | Lab monitoring (G6), device console (xterm.js + WebSocket), WebSocket real-time |
| **Phase 4** | 🔲 Planned | Dark mode, responsive polish, keyboard shortcuts, collaborative editing (Yjs), WASM |
