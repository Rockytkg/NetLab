import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  App,
  Avatar,
  Button,
  Card,
  Col,
  Descriptions,
  Flex,
  Form,
  Input,
  Result,
  Skeleton,
  Row,
  Tag,
  Tabs,
  Typography,
} from 'antd'
import {
  PhoneOutlined,
  ReloadOutlined,
  SaveOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/authStore'
import type { UpdateProfileParams } from '@/types/auth'
import { accountApi, type AccountCenterSnapshot } from '@/services/account'
import { getAvatarColor } from '@/utils/avatar'
import ChangeEmailPanel from './account/ChangeEmailPanel'
import ChangePasswordPanel from './account/ChangePasswordPanel'
import OAuthBindingsPanel from './account/OAuthBindingsPanel'
import PasskeyPanel from './account/PasskeyPanel'
import TwoFactorPanel from './account/TwoFactorPanel'

const { Text } = Typography

export default function SettingsProfilePage() {
  const { t } = useTranslation('settings')
  const { message } = App.useApp()
  const setUserInfo = useAuthStore((state) => state.setUserInfo)
  const [form] = Form.useForm<UpdateProfileParams>()
  const [snapshot, setSnapshot] = useState<AccountCenterSnapshot | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [saving, setSaving] = useState(false)

  const loadSnapshot = useCallback(async () => {
    setLoading(true)
    setError(false)
    try {
      const next = await accountApi.getSnapshot()
      setSnapshot(next)
      setUserInfo(next.user)
      form.setFieldsValue({ nickname: next.user.nickname, phone: next.user.phone })
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [form, setUserInfo])

  useEffect(() => {
    void loadSnapshot()
  }, [loadSnapshot])

  const submitProfile = useCallback(async (values: UpdateProfileParams) => {
    setSaving(true)
    try {
      const user = await accountApi.updateProfile({
        nickname: values.nickname.trim(),
        phone: values.phone.trim(),
      })
      setSnapshot((current) => (current ? { ...current, user } : current))
      setUserInfo(user)
      message.success(t('settings:profile.updateSuccess'))
    } finally {
      setSaving(false)
    }
  }, [message, setUserInfo, t])

  const securityItems = useMemo(() => {
    if (!snapshot) return []
    return [
      {
        key: 'profile',
        label: t('settings:profile.editTitle'),
        children: (
          <Form form={form} layout="vertical" onFinish={submitProfile} requiredMark={false}>
            <Flex vertical gap="large">
              <Form.Item
                name="nickname"
                label={t('settings:profile.nickname')}
                rules={[{ required: true, message: t('settings:profile.nicknameRequired') }]}
              >
                <Input prefix={<UserOutlined />} maxLength={64} />
              </Form.Item>
              <Form.Item
                name="phone"
                label={t('settings:profile.phone')}
                rules={[{ required: true, message: t('settings:profile.phoneRequired') }]}
              >
                <Input prefix={<PhoneOutlined />} maxLength={11} autoComplete="tel" />
              </Form.Item>
              <Button type="primary" htmlType="submit" loading={saving} icon={<SaveOutlined />}>
                {saving ? t('settings:saving') : t('settings:profile.save')}
              </Button>
            </Flex>
          </Form>
        ),
      },
      {
        key: 'email',
        label: t('settings:changeEmail.title'),
        children: <ChangeEmailPanel />,
      },
      {
        key: 'password',
        label: t('settings:changePassword.title'),
        children: <ChangePasswordPanel />,
      },
      {
        key: 'passkey',
        label: t('settings:passkey.title'),
        children: <PasskeyPanel enabled={snapshot.system.passkeyEnabled} initialPasskeys={snapshot.passkeys} />,
      },
      {
        key: 'twofa',
        label: t('settings:twoFactor.title'),
        children: <TwoFactorPanel forceRequired={Boolean(snapshot.system.twoFactorRequired)} />,
      },
      ...(snapshot.system.oauthProviders.length > 0
        ? [{
            key: 'oauth',
            label: t('settings:oauthBindings.title'),
            children: <OAuthBindingsPanel providers={snapshot.system.oauthProviders} initialBindings={snapshot.bindings} />,
          }]
        : []),
    ]
  }, [form, saving, snapshot, submitProfile, t])

  if (loading) {
    return <Skeleton active paragraph={{ rows: 10 }} />
  }

  if (error || !snapshot) {
    return (
      <Result
        status="warning"
        title={t('settings:profile.loadFailedTitle')}
        subTitle={t('settings:profile.loadFailedDescription')}
        extra={
          <Button type="primary" icon={<ReloadOutlined />} onClick={() => void loadSnapshot()}>
            {t('settings:profile.retry')}
          </Button>
        }
      />
    )
  }

  const { user } = snapshot
  const displayName = user.nickname || user.username || '-'

  return (
      <Row gutter={[24, 24]} align="top">
        <Col xs={24} lg={8}>
          <Card title={t('settings:profile.currentInfo')} variant="outlined">
            <Flex vertical gap="large">
              <Flex align="center" gap="large">
            <Avatar
              size={80}
              src={user.avatar}
              icon={!user.nickname ? <UserOutlined /> : undefined}
              style={{ backgroundColor: getAvatarColor(user.nickname) }}
            >
              {user.nickname?.charAt(0)}
            </Avatar>
                <Flex vertical gap="small">
                  <Text strong>{displayName}</Text>
                  <Flex gap="small" wrap>
                <Text type="secondary">@{user.username}</Text>
                {(user.roleName || user.role) && <Tag color="blue">{user.roleName || user.role}</Tag>}
                  </Flex>
                </Flex>
              </Flex>
              <Descriptions column={1} size="middle">
                <Descriptions.Item label={t('settings:profile.username')}>{user.username}</Descriptions.Item>
                <Descriptions.Item label={t('settings:profile.nickname')}>{user.nickname || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('settings:profile.phone')}>{user.phone || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('settings:profile.email')}>{user.email || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('settings:profile.twoFactorStatus')}>
                  <Tag
                    icon={<SafetyCertificateOutlined />}
                    color={user.twoFactorEnabled ? 'success' : 'default'}
                  >
                    {user.twoFactorEnabled ? t('settings:profile.enabled') : t('settings:profile.disabled')}
                  </Tag>
                </Descriptions.Item>
              </Descriptions>
            </Flex>
          </Card>
        </Col>

        <Col xs={24} lg={16}>
          <Card variant="outlined">
            <Tabs items={securityItems} type="line" size="large" destroyOnHidden={false} />
          </Card>
        </Col>
      </Row>
  )
}
