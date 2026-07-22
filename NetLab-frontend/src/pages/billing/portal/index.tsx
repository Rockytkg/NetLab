import { useCallback, useEffect, useState } from 'react'
import { App, Button, Card, Form, Input, InputNumber, Modal, Result, Select, Space, Table, Tabs, Tag } from 'antd'
import { DeleteOutlined, DisconnectOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import type { ColumnsType } from 'antd/es/table'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import { portalApi } from '@/services/portal'
import type { PortalNasItem, PortalNasPayload, PortalSessionItem } from '@/types/portal'
import { renderTime } from '@/pages/billing/shared'

type NasForm = PortalNasPayload

export default function PortalPage() {
  const { t } = useTranslation(['portal', 'common'])
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canRead = can('portal.read')
  const [nas, setNas] = useState<PortalNasItem[]>([])
  const [sessions, setSessions] = useState<PortalSessionItem[]>([])
  const [nasTotal, setNasTotal] = useState(0)
  const [sessionTotal, setSessionTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [sessionUser, setSessionUser] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<PortalNasItem | null>(null)
  const [form] = Form.useForm<NasForm>()

  const load = useCallback(async () => {
    if (!canRead) return
    setLoading(true)
    try {
      const [nasResult, sessionResult] = await Promise.all([
        portalApi.listNas({ page: 1, size: 100, keyword: keyword || undefined }),
        portalApi.listSessions({ page: 1, size: 100, username: sessionUser || undefined }),
      ])
      setNas(nasResult.items ?? []); setNasTotal(nasResult.total ?? 0)
      setSessions(sessionResult.items ?? []); setSessionTotal(sessionResult.total ?? 0)
    } finally { setLoading(false) }
  }, [canRead, keyword, sessionUser])
  useEffect(() => { load() }, [load])
  if (!canRead) return <Result status="403" title="403" subTitle={t('common:noPermission')} />

  const submitNas = async () => {
    const values = await form.validateFields()
    if (editing) await portalApi.updateNas(editing.id, values); else await portalApi.createNas(values)
    message.success(t('portal:messages.saved')); setModalOpen(false); setEditing(null); await load()
  }
  const openNas = (item?: PortalNasItem) => { setEditing(item ?? null); form.setFieldsValue(item ? { ...item, sharedSecret: undefined } : { vendor: 'mobile', protocolProfile: 'cmcc-v2', acPort: 2000, coaEnabled: false, status: 'enabled' }); setModalOpen(true) }
  const deleteNas = (item: PortalNasItem) => modal.confirm({ title: t('portal:confirm.deleteNas'), okButtonProps: { danger: true }, async onOk() { await portalApi.deleteNas(item.id); message.success(t('portal:messages.deleted')); await load() } })
  const terminate = (item: PortalSessionItem) => modal.confirm({ title: t('portal:confirm.terminate'), okButtonProps: { danger: true }, async onOk() { await portalApi.terminateSession(item.id); message.success(t('portal:messages.terminated')); await load() } })

  const nasColumns: ColumnsType<PortalNasItem> = [
    { title: t('portal:nas.name'), dataIndex: 'name' }, { title: t('portal:nas.identifier'), dataIndex: 'identifier' }, { title: t('portal:nas.vendor'), dataIndex: 'vendor' },
    { title: t('portal:nas.profile'), dataIndex: 'protocolProfile' }, { title: t('portal:nas.sourceIp'), dataIndex: 'sourceIp' }, { title: t('portal:nas.acPort'), dataIndex: 'acPort' },
    { title: t('portal:nas.status'), dataIndex: 'status', render: (value) => <Tag color={value === 'enabled' ? 'green' : 'default'}>{t(`portal:status.${value}`)}</Tag> },
    { title: t('portal:common.actions'), key: 'actions', render: (_, item) => <Can permission="portal.manage"><Space><Button type="text" icon={<EditOutlined />} onClick={() => openNas(item)} aria-label={t('portal:common.edit')} /><Button type="text" danger icon={<DeleteOutlined />} onClick={() => deleteNas(item)} aria-label={t('portal:common.delete')} /></Space></Can> },
  ]
  const sessionColumns: ColumnsType<PortalSessionItem> = [
    { title: t('portal:sessions.username'), dataIndex: 'username' }, { title: t('portal:sessions.clientIp'), dataIndex: 'clientIp' }, { title: t('portal:sessions.mac'), dataIndex: 'macAddr' }, { title: t('portal:sessions.authenticatedAt'), dataIndex: 'authenticatedAt', render: renderTime }, { title: t('portal:sessions.lastSeenAt'), dataIndex: 'lastSeenAt', render: renderTime },
    { title: t('portal:common.actions'), key: 'actions', render: (_, item) => item.state === 'active' ? <Can permission="portal.manage"><Button type="text" danger icon={<DisconnectOutlined />} onClick={() => terminate(item)} aria-label={t('portal:sessions.terminate')} /></Can> : <Tag>{t('portal:status.terminated')}</Tag> },
  ]
  return <Card title={t('portal:title')} extra={<Button icon={<ReloadOutlined />} onClick={load}>{t('common:refresh')}</Button>}>
    <Tabs items={[
      { key: 'nas', label: t('portal:nas.tab'), children: <><Space style={{ marginBottom: 16 }}><Input.Search placeholder={t('portal:nas.search')} onSearch={(v) => setKeyword(v.trim())} allowClear /><Can permission="portal.manage"><Button type="primary" icon={<PlusOutlined />} onClick={() => openNas()}>{t('portal:nas.create')}</Button></Can></Space><Table rowKey="id" columns={nasColumns} dataSource={nas} loading={loading} pagination={{ total: nasTotal }} /></> },
      { key: 'sessions', label: t('portal:sessions.tab'), children: <><Input.Search style={{ width: 280, marginBottom: 16 }} placeholder={t('portal:sessions.search')} onSearch={(v) => setSessionUser(v.trim())} allowClear /><Table rowKey="id" columns={sessionColumns} dataSource={sessions} loading={loading} pagination={{ total: sessionTotal }} /></> },
    ]} />
    <Modal title={editing ? t('portal:nas.edit') : t('portal:nas.create')} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={submitNas} destroyOnHidden>
      <Form form={form} layout="vertical"><Form.Item name="name" label={t('portal:nas.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="identifier" label={t('portal:nas.identifier')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="vendor" label={t('portal:nas.vendor')} rules={[{ required: true }]}><Select options={[{ value: 'mobile', label: t('portal:vendor.mobile') }, { value: 'huawei', label: t('portal:vendor.huawei') }]} /></Form.Item><Form.Item name="protocolProfile" label={t('portal:nas.profile')} rules={[{ required: true }]}><Select options={[{ value: 'cmcc-v1', label: t('portal:profile.cmccV1') }, { value: 'cmcc-v2', label: t('portal:profile.cmccV2') }, { value: 'huawei-v1', label: t('portal:profile.huaweiV1') }, { value: 'huawei-v2', label: t('portal:profile.huaweiV2') }]} /></Form.Item><Form.Item name="sourceIp" label={t('portal:nas.sourceIp')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="acPort" label={t('portal:nas.acPort')} rules={[{ required: true }]}><InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} /></Form.Item><Form.Item name="sharedSecret" label={t('portal:nas.secret')} rules={[{ required: !editing }]}><Input.Password /></Form.Item><Form.Item name="status" label={t('portal:nas.status')}><Select options={[{ value: 'enabled', label: t('portal:status.enabled') }, { value: 'disabled', label: t('portal:status.disabled') }]} /></Form.Item></Form>
    </Modal>
  </Card>
}
