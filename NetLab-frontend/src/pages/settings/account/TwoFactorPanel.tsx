import { useState } from 'react'
import { Alert, App, Button, Card, Modal, Radio, Space, Tag, Typography, theme } from 'antd'
import { LockOutlined, SafetyOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import TwoFactorBindingSteps from './TwoFactorBindingSteps'
import EmailCodeField from './EmailCodeField'

const { Text } = Typography

interface TwoFactorPanelProps {
  /** 系统是否强制开启两步验证 */
  forceRequired: boolean
}

/** 个人中心 · 两步验证面板：开启（绑定验证器）、关闭（邮箱验证码）与首选方式。 */
export default function TwoFactorPanel({ forceRequired }: TwoFactorPanelProps) {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const userInfo = useAuthStore((s) => s.userInfo)
  const fetchUserInfo = useAuthStore((s) => s.fetchUserInfo)
  const enabled = !!userInfo?.twoFactorEnabled

  const [enableOpen, setEnableOpen] = useState(false)
  const [enableKey, setEnableKey] = useState(0)
  const [disableOpen, setDisableOpen] = useState(false)
  const [disabling, setDisabling] = useState(false)
  const [disableCode, setDisableCode] = useState('')

  const [preferredMethod, setPreferredMethod] = useState<string>(userInfo?.preferredAuthMethod || 'totp')
  const [savingPreferred, setSavingPreferred] = useState(false)

  const openEnable = () => {
    setEnableKey((k) => k + 1)
    setEnableOpen(true)
  }

  const handleEnabled = async () => {
    setEnableOpen(false)
    await fetchUserInfo()
  }

  const handleDisable = async () => {
    if (disableCode.length !== 6) {
      message.warning(t('twoFactor.disableEmailRequired'))
      return
    }
    setDisabling(true)
    try {
      await authApi.disableTwoFactor(disableCode)
      message.success(t('twoFactor.disableSuccess'))
      setDisableOpen(false)
      setDisableCode('')
      await fetchUserInfo()
    } catch {
      // interceptor
    } finally {
      setDisabling(false)
    }
  }

  const handlePreferredChange = async (method: string) => {
    setPreferredMethod(method)
    setSavingPreferred(true)
    try {
      await authApi.setPreferredAuthMethod(method as 'totp' | 'passkey')
      message.success(t('twoFactor.preferredMethodUpdated'))
      await fetchUserInfo()
    } catch {
      // 回滚到原值
      setPreferredMethod(userInfo?.preferredAuthMethod || 'totp')
    } finally {
      setSavingPreferred(false)
    }
  }

  const passkeyAvailable = !!userInfo?.hasPasskey

  return (
    <Card
      title={t('twoFactor.title')}
      variant='outlined'
      extra={
        <Tag color={enabled ? 'success' : 'default'}>
          {enabled ? t('twoFactor.statusEnabled') : t('twoFactor.statusDisabled')}
        </Tag>
      }
      styles={{ body: { paddingBlock: token.paddingLG } }}
    >
      <Text type='secondary'>{t('twoFactor.description')}</Text>

      {forceRequired && (
        <Alert
          type='warning'
          showIcon
          title={enabled ? t('twoFactor.forcedDisableHint') : t('twoFactor.forceNotice')}
          style={{ marginTop: token.margin }}
        />
      )}

      <div style={{ marginTop: token.marginLG }}>
        {enabled ? (
          <Button danger icon={<LockOutlined />} disabled={forceRequired} onClick={() => setDisableOpen(true)}>
            {t('twoFactor.disable')}
          </Button>
        ) : (
          <Button type='primary' icon={<SafetyOutlined />} onClick={openEnable}>
            {t('twoFactor.enable')}
          </Button>
        )}
      </div>

      {enabled && (
        <div style={{ marginTop: token.marginLG }}>
          <Text strong>{t('twoFactor.preferredMethod')}</Text>
          <div style={{ marginTop: token.marginXS }}>
            <Radio.Group
              value={preferredMethod}
              onChange={(e) => handlePreferredChange(e.target.value)}
              disabled={savingPreferred}
            >
              <Space orientation='vertical'>
                <Radio value='totp'>{t('twoFactor.preferredMethodTotp')}</Radio>
                <Radio value='passkey' disabled={!passkeyAvailable}>
                  {t('twoFactor.preferredMethodPasskey')}
                  {!passkeyAvailable && (
                    <Text type='secondary' style={{ fontSize: 12, marginLeft: token.marginXS }}>
                      {t('twoFactor.preferredMethodPasskeyDisabled')}
                    </Text>
                  )}
                </Radio>
              </Space>
            </Radio.Group>
          </div>
          <Text type='secondary' style={{ fontSize: 12 }}>
            {t('twoFactor.preferredMethodHelp')}
          </Text>
        </div>
      )}

      <Modal title={t('twoFactor.setupTitle')} open={enableOpen} onCancel={() => setEnableOpen(false)} footer={null} width={440}>
        <TwoFactorBindingSteps key={enableKey} onComplete={handleEnabled} />
      </Modal>

      <Modal
        title={t('twoFactor.disableTitle')}
        open={disableOpen}
        onCancel={() => {
          setDisableOpen(false)
          setDisableCode('')
        }}
        onOk={handleDisable}
        okText={t('twoFactor.disableConfirm')}
        okButtonProps={{ danger: true, loading: disabling }}
        cancelText={t('common:cancel')}
      >
        <Space orientation='vertical' size={token.margin} style={{ width: '100%' }}>
          <Alert type='warning' showIcon title={t('twoFactor.disableNotice')} />
          <div>
            <div style={{ marginBottom: token.marginXS }}>
              <Text>{t('twoFactor.disableEmailLabel')}</Text>
            </div>
            <EmailCodeField value={disableCode} onChange={setDisableCode} purpose='disable-2fa' />
          </div>
        </Space>
      </Modal>
    </Card>
  )
}
