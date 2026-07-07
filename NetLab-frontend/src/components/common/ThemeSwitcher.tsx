import { useMemo } from 'react'
import { Dropdown, Button, type MenuProps } from 'antd'
import { SunOutlined, MoonOutlined, DesktopOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useAppStore, type ThemeMode } from '@/stores/appStore'

/**
 * Theme option descriptors.
 *
 * Icons follow Ant Design semantics:
 * - SunOutlined     → light (day / high contrast)
 * - MoonOutlined    → dark (night / low light)
 * - DesktopOutlined → system (follows OS-level preference)
 */
const THEME_OPTIONS: { value: ThemeMode; labelKey: string; icon: React.ReactNode }[] = [
  { value: 'light',  labelKey: 'login:themeLight',  icon: <SunOutlined /> },
  { value: 'dark',   labelKey: 'login:themeDark',   icon: <MoonOutlined /> },
  { value: 'system', labelKey: 'login:themeSystem', icon: <DesktopOutlined /> },
]

interface ThemeSwitcherProps {
  /**
   * Keeps the trigger legible on dark artwork while the dropdown itself
   * follows the global light/dark/system theme.
   */
  inverted?: boolean
  /** Show the current mode label next to the icon. */
  showLabel?: boolean
}

/**
 * Standalone theme (light/dark/system) switcher. Split out from the combined
 * display-settings popover so theme and language are independent controls.
 */
export default function ThemeSwitcher({ inverted = false, showLabel = false }: ThemeSwitcherProps) {
  const { t } = useTranslation(['login', 'common'])
  const themeMode = useAppStore((s) => s.themeMode)
  const setThemeMode = useAppStore((s) => s.setThemeMode)

  const current = THEME_OPTIONS.find((opt) => opt.value === themeMode) ?? THEME_OPTIONS[2]

  const items: MenuProps['items'] = useMemo(
    () =>
      THEME_OPTIONS.map((opt) => ({
        key: opt.value,
        icon: opt.icon,
        label: t(opt.labelKey as never),
      })),
    [t],
  )

  return (
    <Dropdown
      trigger={['click']}
      placement="topLeft"
      menu={{
        items,
        selectable: true,
        selectedKeys: [themeMode],
        onClick: ({ key }) => setThemeMode(key as ThemeMode),
      }}
    >
      <Button
        className={inverted ? 'netlab-inverted-control' : undefined}
        type="text"
        size="small"
        icon={current.icon}
        aria-label={t('login:themeMode')}
      >
        {showLabel ? t(current.labelKey as never) : null}
      </Button>
    </Dropdown>
  )
}
