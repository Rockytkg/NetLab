import { theme as antdTheme, type ThemeConfig } from 'antd'

const fontFamily =
  "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, " +
  "'Helvetica Neue', Arial, 'Noto Sans', sans-serif, " +
  "'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'"

const seedToken: ThemeConfig['token'] = {
  colorPrimary: '#1677FF',
  colorInfo: '#1677FF',
  colorSuccess: '#52C41A',
  colorWarning: '#FAAD14',
  colorError: '#FF4D4F',
  fontFamily,
  fontSize: 14,
  borderRadius: 6,
  borderRadiusLG: 8,
  borderRadiusSM: 4,
  controlHeight: 32,
  wireframe: false,
}

const componentToken: ThemeConfig['components'] = {
  Layout: {
    headerHeight: 64,
    headerPadding: '0 24px',
    footerPadding: '0 24px',
  },
  Menu: {
    itemBorderRadius: 6,
    itemHeight: 40,
    itemMarginBlock: 4,
    itemMarginInline: 4,
    collapsedWidth: 80,
  },
  Card: {
    headerFontSize: 16,
    bodyPadding: 24,
  },
  Drawer: {
    paddingLG: 24,
  },
  Segmented: {
    itemSelectedBg: 'var(--ant-color-bg-container)',
    trackBg: 'var(--ant-color-fill-tertiary)',
  },
  Table: {
    headerBorderRadius: 0,
  },
  Tag: {
    borderRadiusSM: 4,
  },
}

export function createAppTheme(isDark: boolean): ThemeConfig {
  return {
    algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    cssVar: {
      prefix: 'ant',
      key: isDark ? 'netlab-dark' : 'netlab-light',
    },
    hashed: true,
    token: seedToken,
    components: componentToken,
  }
}
