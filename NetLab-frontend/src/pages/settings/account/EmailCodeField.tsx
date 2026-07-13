import { useEffect, useState } from 'react'
import { Button, Input, Space, App } from 'antd'
import { useTranslation } from 'react-i18next'
import { authApi } from '@/services/auth'
import type { AccountCodePurpose } from '@/types/auth'

interface EmailCodeFieldProps {
  value: string
  onChange: (val: string) => void
  purpose: AccountCodePurpose
  disabled?: boolean
}

/**
 * 账户敏感操作的邮箱验证码输入组件。
 * 「发送验证码」按钮会调用已认证端点向本人邮箱发码，并显示 60s 冷却倒计时。
 */
export default function EmailCodeField({ value, onChange, purpose, disabled }: EmailCodeFieldProps) {
  const { t } = useTranslation(['settings', 'common'])
  const { message } = App.useApp()
  const [cooldown, setCooldown] = useState(0)
  const [sending, setSending] = useState(false)

  useEffect(() => {
    if (cooldown <= 0) return
    const timer = setInterval(() => setCooldown((c) => (c <= 1 ? 0 : c - 1)), 1000)
    return () => clearInterval(timer)
  }, [cooldown])

  const handleSend = async () => {
    setSending(true)
    try {
      const res = await authApi.sendAccountEmailCode(purpose)
      setCooldown(res.cooldown > 0 ? res.cooldown : 60)
      message.success(t('settings:account.codeSent'))
    } catch {
      // 拦截器已提示错误
    } finally {
      setSending(false)
    }
  }

  return (
    <Space.Compact style={{ width: '100%' }}>
      <Input
        value={value}
        onChange={(e) => onChange(e.target.value.trim())}
        placeholder={t('settings:account.codePlaceholder')}
        maxLength={6}
        disabled={disabled}
      />
      <Button onClick={handleSend} loading={sending} disabled={disabled || cooldown > 0}>
        {cooldown > 0 ? t('settings:account.codeResend', { seconds: cooldown }) : t('settings:account.sendCode')}
      </Button>
    </Space.Compact>
  )
}
