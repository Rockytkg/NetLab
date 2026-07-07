import { useMemo } from 'react'
import { Dropdown, Button, type MenuProps } from 'antd'
import { GlobalOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useAppStore } from '@/stores/appStore'
import { useI18n } from '@/hooks/useI18n'
import { LOCALE_OPTIONS } from '@/types/i18n'
import type { SupportedLocale } from '@/types/i18n'

interface LanguageSwitcherProps {
  /**
   * Keeps the trigger legible on dark artwork while the dropdown itself
   * follows the global light/dark/system theme.
   */
  inverted?: boolean
  /** Show the current language label next to the globe icon. */
  showLabel?: boolean
}

/**
 * Standalone language switcher. Split out from the combined display-settings
 * popover so theme and language are independent controls.
 */
export default function LanguageSwitcher({ inverted = false, showLabel = false }: LanguageSwitcherProps) {
  const { t } = useTranslation(['login', 'common'])
  const { switchLanguage } = useI18n()
  const locale = useAppStore((s) => s.locale)

  const current = LOCALE_OPTIONS.find((opt) => opt.value === locale) ?? LOCALE_OPTIONS[0]

  const items: MenuProps['items'] = useMemo(
    () =>
      LOCALE_OPTIONS.map((opt) => ({
        key: opt.value,
        label: opt.label,
      })),
    [],
  )

  return (
    <Dropdown
      trigger={['click']}
      placement="topLeft"
      menu={{
        items,
        selectable: true,
        selectedKeys: [locale],
        onClick: ({ key }) => switchLanguage(key as SupportedLocale),
      }}
    >
      <Button
        className={inverted ? 'netlab-inverted-control' : undefined}
        type="text"
        size="small"
        icon={<GlobalOutlined />}
        aria-label={t('login:languageLabel')}
      >
        {showLabel ? current.label : null}
      </Button>
    </Dropdown>
  )
}
