import { useCallback, useEffect, useState } from 'react'
import {
  App,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Result,
  Space,
  Table,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { DisconnectOutlined, ReloadOutlined, SearchOutlined, SendOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import Toolbar from '@/pages/billing/components/Toolbar'
import { formatBytes, formatDuration } from '../format'
import { renderTime } from '@/pages/billing/shared'
import type { RadiusCoAPayload, RadiusSessionItem } from '@/types/radius'

/** CoA 表单值：两个字段均可空，但提交时至少一项有值。 */
interface CoaFormValues {
  sessionTimeout?: number | null
  filterId?: string
}

/** RADIUS 在线会话页：分页列表 + 用户名/NAS/MAC 筛选 + 踢下线。 */
export default function RadiusSessionsPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadSessions = can('radius.read')

  const [data, setData] = useState<RadiusSessionItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [usernameInput, setUsernameInput] = useState('')
  const [nasAddrInput, setNasAddrInput] = useState('')
  const [macAddrInput, setMacAddrInput] = useState('')
  const [username, setUsername] = useState('')
  const [nasAddr, setNasAddr] = useState('')
  const [macAddr, setMacAddr] = useState('')
  const [loading, setLoading] = useState(false)

  // CoA 下发弹窗
  const [coaOpen, setCoaOpen] = useState(false)
  const [coaTarget, setCoaTarget] = useState<RadiusSessionItem | null>(null)
  const [coaForm] = Form.useForm<CoaFormValues>()
  const [coaSaving, setCoaSaving] = useState(false)

  const load = useCallback(async () => {
    if (!canReadSessions) return
    setLoading(true)
    try {
      const res = await radiusApi.listSessions({
        page,
        size,
        username: username || undefined,
        nasAddr: nasAddr || undefined,
        macAddr: macAddr || undefined,
      })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadSessions, page, size, username, nasAddr, macAddr])

  useEffect(() => {
    load()
  }, [load])

  const handleSearch = () => {
    setPage(1)
    setUsername(usernameInput.trim())
    setNasAddr(nasAddrInput.trim())
    setMacAddr(macAddrInput.trim())
  }

  const handleReset = () => {
    setPage(1)
    setUsernameInput('')
    setNasAddrInput('')
    setMacAddrInput('')
    setUsername('')
    setNasAddr('')
    setMacAddr('')
  }

  const handleKick = (record: RadiusSessionItem) => {
    modal.confirm({
      title: t('radius:common.confirmTitle'),
      content: t('radius:sessions.kickConfirm'),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        const res = await radiusApi.kickSession(record.id)
        if (res.success) {
          message.success(t('radius:sessions.kickSuccess'))
        } else {
          message.error(t('radius:sessions.kickFailed'))
        }
        await load()
      },
    })
  }

  const openCoa = (record: RadiusSessionItem) => {
    setCoaTarget(record)
    coaForm.resetFields()
    setCoaOpen(true)
  }

  const handleCoaSubmit = async () => {
    try {
      const values = await coaForm.validateFields()
      const sessionTimeout = values.sessionTimeout ?? undefined
      const filterId = values.filterId?.trim() || undefined
      if (sessionTimeout == null && !filterId) {
        message.warning(t('radius:sessions.coaAtLeastOne'))
        return
      }
      if (!coaTarget) return
      setCoaSaving(true)
      const payload: RadiusCoAPayload = {}
      if (sessionTimeout != null) payload.sessionTimeout = sessionTimeout
      if (filterId) payload.filterId = filterId
      const res = await radiusApi.coaSession(coaTarget.id, payload)
      if (res.success) {
        message.success(t('radius:sessions.coaSuccess', { responseCode: res.responseCode }))
        setCoaOpen(false)
        setCoaTarget(null)
      } else {
        message.error(
          t('radius:sessions.coaFailed', {
            responseCode: res.responseCode || '-',
            errorCauseText: res.errorCauseText || res.message || '-',
          }),
        )
      }
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return
    } finally {
      setCoaSaving(false)
    }
  }

  const columns: ColumnsType<RadiusSessionItem> = [
    {
      title: t('radius:sessions.columns.username'),
      dataIndex: 'username',
      key: 'username',
      width: 110,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:sessions.columns.nasAddr'),
      dataIndex: 'nasAddr',
      key: 'nasAddr',
      width: 120,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:sessions.columns.framedIp'),
      dataIndex: 'framedIpaddr',
      key: 'framedIpaddr',
      width: 120,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:sessions.columns.macAddr'),
      dataIndex: 'macAddr',
      key: 'macAddr',
      width: 130,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:sessions.columns.nasPort'),
      dataIndex: 'nasPort',
      key: 'nasPort',
      width: 90,
      responsive: ['xxl'],
      render: (val: number) => val || '-',
    },
    {
      title: t('radius:sessions.columns.startTime'),
      dataIndex: 'acctStartTime',
      key: 'acctStartTime',
      width: 150,
      render: renderTime,
    },
    {
      title: t('radius:sessions.columns.sessionTime'),
      dataIndex: 'acctSessionTime',
      key: 'acctSessionTime',
      width: 100,
      render: (val: number) => formatDuration(val),
    },
    {
      title: t('radius:sessions.columns.sessionTimeout'),
      dataIndex: 'sessionTimeout',
      key: 'sessionTimeout',
      width: 120,
      responsive: ['xxl'],
      render: (val?: number) => (val ? formatDuration(val) : '-'),
    },
    {
      title: t('radius:sessions.columns.upload'),
      dataIndex: 'acctInputTotal',
      key: 'acctInputTotal',
      width: 100,
      render: (val: number) => formatBytes(val),
    },
    {
      title: t('radius:sessions.columns.download'),
      dataIndex: 'acctOutputTotal',
      key: 'acctOutputTotal',
      width: 100,
      render: (val: number) => formatBytes(val),
    },
    {
      title: t('radius:sessions.columns.lastUpdate'),
      dataIndex: 'lastUpdate',
      key: 'lastUpdate',
      width: 150,
      responsive: ['xxl'],
      render: renderTime,
    },
    {
      title: t('radius:common.actions'),
      key: 'actions',
      width: 150,
      align: 'center',
      fixed: 'right',
      render: (_, record) => (
        <Can permission="radius.manage">
          <Space size={4}>
            <Button
              type="text"
              size="small"
              icon={<SendOutlined />}
              onClick={() => openCoa(record)}
            >
              {t('radius:sessions.coa')}
            </Button>
            <Button
              type="text"
              size="small"
              danger
              icon={<DisconnectOutlined />}
              onClick={() => handleKick(record)}
            >
              {t('radius:sessions.kick')}
            </Button>
          </Space>
        </Can>
      ),
    },
  ]

  if (!canReadSessions) {
    return <Result status="403" title="403" subTitle={t('settings:permissionDenied')} />
  }

  return (
    <div>
      <Card variant="outlined">
        <Toolbar
          left={
            <>
              <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                {t('common:search')}
              </Button>
              <Button onClick={handleReset}>{t('common:reset')}</Button>
            </>
          }
          right={
            <>
              <Input
                value={usernameInput}
                onChange={(e) => setUsernameInput(e.target.value)}
                onPressEnter={handleSearch}
                placeholder={t('radius:sessions.searchUsername')}
                allowClear
                className="netlab-billing-toolbar-search"
              />
              <Input
                value={nasAddrInput}
                onChange={(e) => setNasAddrInput(e.target.value)}
                onPressEnter={handleSearch}
                placeholder={t('radius:sessions.searchNas')}
                allowClear
                className="netlab-billing-toolbar-search"
              />
              <Input
                value={macAddrInput}
                onChange={(e) => setMacAddrInput(e.target.value)}
                onPressEnter={handleSearch}
                placeholder={t('radius:sessions.searchMac')}
                allowClear
                className="netlab-billing-toolbar-search"
              />
              <Button icon={<ReloadOutlined />} onClick={load} />
            </>
          }
        />

        <Table
          className="netlab-billing-table"
          rowKey="id"
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={{
            current: page,
            pageSize: size,
            total,
            showSizeChanger: true,
            onChange: (p, s) => {
              setPage(p)
              setSize(s)
            },
            showTotal: (tt) => t('settings:loginLogs.total', { total: tt }),
          }}
          scroll={{ x: 1250 }}
          tableLayout="fixed"
        />
      </Card>

      {/* 下发 CoA：修改会话的 Session-Timeout / Filter-Id */}
      <Modal
        title={t('radius:sessions.coaTitle', { username: coaTarget?.username ?? '' })}
        open={coaOpen}
        onCancel={() => {
          setCoaOpen(false)
          setCoaTarget(null)
          coaForm.resetFields()
        }}
        onOk={handleCoaSubmit}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={coaSaving}
        forceRender
        width={{ xs: 'calc(100vw - 32px)', sm: 520 }}
      >
        <Form form={coaForm} layout="vertical" requiredMark={false}>
          <Form.Item
            name="sessionTimeout"
            label={t('radius:sessions.coaSessionTimeout')}
            tooltip={t('radius:sessions.coaSessionTimeoutTip')}
          >
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            name="filterId"
            label={t('radius:sessions.coaFilterId')}
            tooltip={t('radius:sessions.coaFilterIdTip')}
          >
            <Input maxLength={253} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
