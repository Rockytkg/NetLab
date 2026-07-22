import { useState } from 'react'
import { Button, Card, Form, Input, Result, Select, Typography } from 'antd'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { portalApi } from '@/services/portal'

type FormValues = { username: string; password: string; authType: 'chap' | 'pap' }

/** AC redirect landing page. The AC-provided parameters remain immutable. */
export default function PortalLoginPage() {
  const { t } = useTranslation('portal')
  const [params] = useSearchParams()
  const [form] = Form.useForm<FormValues>()
  const [loading, setLoading] = useState(false)
  const [connected, setConnected] = useState(false)
  const acName = params.get('wlanacname') ?? ''
  const clientIP = params.get('wlanuserip') ?? ''
  const available = Boolean(acName && clientIP)
  const submit = async () => {
    const values = await form.validateFields()
    setLoading(true)
    try {
      await portalApi.authenticate({ wlanacname: acName, wlanuserip: clientIP, ...values })
      setConnected(true)
    } finally { setLoading(false) }
  }
  if (connected) return <Result status="success" title={t('login.success')} subTitle={t('login.successDetail')} />
  if (!available) return <Result status="warning" title={t('login.missingContext')} />
  return <main style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: 16 }}><Card style={{ width: '100%', maxWidth: 420 }}><Typography.Title level={3}>{t('login.title')}</Typography.Title><Form form={form} layout="vertical" initialValues={{ authType: 'chap' }} onFinish={submit}><Form.Item name="username" label={t('login.username')} rules={[{ required: true }]}><Input autoComplete="username" /></Form.Item><Form.Item name="password" label={t('login.password')} rules={[{ required: true }]}><Input.Password autoComplete="current-password" /></Form.Item><Form.Item name="authType" label={t('login.authType')}><Select options={[{ value: 'chap', label: t('login.chap') }, { value: 'pap', label: t('login.pap') }]} /></Form.Item><Button type="primary" htmlType="submit" block loading={loading}>{t('login.submit')}</Button></Form></Card></main>
}
