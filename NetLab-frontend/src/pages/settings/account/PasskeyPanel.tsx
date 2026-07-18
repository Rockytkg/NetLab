import { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Empty, Flex, Input, List, Modal, Space, Spin, Typography, App, theme } from 'antd'
import { KeyOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { accountApi } from '@/services/account'
import { usePasskey } from '@/hooks/usePasskey'
import type { PasskeyInfo } from '@/types/settings'
import EmailCodeField from './EmailCodeField'

const { Text } = Typography

interface PasskeyPanelProps {
  /** 系统安全策略状态；undefined 表示配置仍在加载。 */
  enabled?: boolean
  initialPasskeys?: PasskeyInfo[]
}

/**
 * 个人中心 · Passkey 管理面板。
 * 添加与删除均需邮箱验证码二次校验。
 */
export default function PasskeyPanel({ enabled, initialPasskeys }: PasskeyPanelProps) {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const { isSupported, register } = usePasskey()

  const [list, setList] = useState<PasskeyInfo[]>([])
  const [loading, setLoading] = useState(false)

  const [addOpen, setAddOpen] = useState(false)
  const [adding, setAdding] = useState(false)
  const [name, setName] = useState('')
  const [addCode, setAddCode] = useState('')

  const [deleteTarget, setDeleteTarget] = useState<PasskeyInfo | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [deleteCode, setDeleteCode] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await accountApi.listPasskeys()
      setList(res.passkeys ?? [])
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (initialPasskeys) {
      setList(initialPasskeys)
      return
    }
    void load()
  }, [initialPasskeys, load])

  const supported = isSupported()
  const canOperate = enabled === true && supported

  const openAdd = () => {
    setName('')
    setAddCode('')
    setAddOpen(true)
  }

  const handleAdd = async () => {
    if (addCode.length !== 6) {
      message.warning(t('settings:account.codeRequired'))
      return
    }
    setAdding(true)
    try {
      const ok = await register(name.trim(), addCode)
      if (ok) {
        message.success(t('settings:passkey.addSuccess'))
        setAddOpen(false)
        await load()
      }
    } finally {
      setAdding(false)
    }
  }

  const openDelete = (item: PasskeyInfo) => {
    setDeleteTarget(item)
    setDeleteCode('')
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    if (deleteCode.length !== 6) {
      message.warning(t('settings:account.codeRequired'))
      return
    }
    setDeleting(true)
    try {
      await accountApi.deletePasskey(deleteTarget.id, deleteCode)
      message.success(t('settings:passkey.deleteSuccess'))
      setDeleteTarget(null)
      await load()
    } catch {
      // 拦截器已提示错误
    } finally {
      setDeleting(false)
    }
  }

  return (
    <Flex vertical gap="large">
      <Flex align="center" justify="space-between" gap="large" wrap>
        <Text type="secondary">{t('settings:passkey.description')}</Text>
        <Button type="primary" icon={<PlusOutlined />} disabled={!canOperate} onClick={openAdd}>
          {t('settings:passkey.add')}
        </Button>
      </Flex>

      {enabled === false && (
        <Alert
          type="warning"
          showIcon
          title={t('settings:passkey.disabled')}
        />
      )}
      {enabled === true && !supported && (
        <Alert
          type="warning"
          showIcon
          title={t('settings:passkey.unsupported')}
        />
      )}

      <div>
        <Spin spinning={loading}>
          {list.length === 0 ? (
            <Empty description={t('settings:passkey.empty')} />
          ) : (
            <List
              itemLayout="horizontal"
              dataSource={list}
              split
              renderItem={(item) => (
                <List.Item
                  actions={[
                    <Button
                      key="delete"
                      type="text"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={() => openDelete(item)}
                    >
                      {t('settings:passkey.delete')}
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={
                      <KeyOutlined
                        style={{ fontSize: token.fontSizeHeading3, color: token.colorPrimary }}
                      />
                    }
                    title={<Text strong>{item.name}</Text>}
                    description={
                      <Space size={token.margin} wrap>
                        <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>
                          {t('settings:passkey.created')}: {new Date(item.createdAt).toLocaleString()}
                        </Text>
                        <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>
                          {t('settings:passkey.lastUsed')}:{' '}
                          {item.lastUsedAt
                            ? new Date(item.lastUsedAt).toLocaleString()
                            : t('settings:passkey.neverUsed')}
                        </Text>
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          )}
        </Spin>
      </div>

      {/* 添加 Passkey */}
      <Modal
        title={t('settings:passkey.add')}
        open={addOpen}
        onCancel={() => setAddOpen(false)}
        onOk={handleAdd}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={adding}
      >
        <Space orientation="vertical" size={token.margin} style={{ width: '100%' }}>
          <div>
            <div style={{ marginBottom: token.marginXS }}>
              <Text>{t('settings:passkey.nameLabel')}</Text>
            </div>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('settings:passkey.namePlaceholder')}
              maxLength={128}
              allowClear
            />
          </div>
          <div>
            <div style={{ marginBottom: token.marginXS }}>
              <Text>{t('settings:account.codeLabel')}</Text>
            </div>
            <EmailCodeField value={addCode} onChange={setAddCode} purpose="passkey" />
          </div>
        </Space>
      </Modal>

      {/* 删除 Passkey */}
      <Modal
        title={t('settings:passkey.delete')}
        open={!!deleteTarget}
        onCancel={() => setDeleteTarget(null)}
        onOk={handleDelete}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        okButtonProps={{ danger: true }}
        confirmLoading={deleting}
      >
        <Space orientation="vertical" size={token.margin} style={{ width: '100%' }}>
          <Text>{t('settings:passkey.deleteConfirm')}</Text>
          <div>
            <div style={{ marginBottom: token.marginXS }}>
              <Text>{t('settings:account.codeLabel')}</Text>
            </div>
            <EmailCodeField value={deleteCode} onChange={setDeleteCode} purpose="passkey" />
          </div>
        </Space>
      </Modal>
    </Flex>
  )
}
