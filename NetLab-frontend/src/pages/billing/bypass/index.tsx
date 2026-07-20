import { useCallback, useEffect, useState } from 'react'
import dayjs, { type Dayjs } from 'dayjs'
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  Modal,
  Result,
  Row,
  Select,
  Space,
  Table,
  Typography,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import Toolbar from '@/pages/billing/components/Toolbar'
import { renderStatusTag, renderTime } from '@/pages/billing/shared'
import type { RadiusBypassItem, RadiusBypassPayload, RadiusBypassType } from '@/types/radius'

const { Text } = Typography

/** 免认证表单值。 */
interface BypassFormValues {
  type: RadiusBypassType
  value: string
  profileId: number
  nasId?: number
  expireTime?: Dayjs
  status: string
  remark?: string
}

const MAC_PATTERN = /^[0-9a-fA-F]{2}([:-][0-9a-fA-F]{2}){5}$/
const IPV4_PATTERN = /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/

/** 哑终端与交换机 Fast MAC Authentication 的准入规则页。 */
export default function BypassPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadRadius = can('radius.read')

  const [data, setData] = useState<RadiusBypassItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)

  // 新增/编辑弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<RadiusBypassItem | null>(null)
  const [form] = Form.useForm<BypassFormValues>()
  const [saving, setSaving] = useState(false)
  const [profiles, setProfiles] = useState<Array<{ id: number; name: string }>>([])
  const [nasItems, setNasItems] = useState<Array<{ id: number; name: string; ipaddr: string }>>([])
  const typeWatch = Form.useWatch('type', form)

  const load = useCallback(async () => {
    if (!canReadRadius) return
    setLoading(true)
    try {
      const res = await radiusApi.listBypass({ page, size, keyword: keyword || undefined })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadRadius, page, size, keyword])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (!canReadRadius) return
    void Promise.all([radiusApi.listProfileOptions(), radiusApi.listNas({ page: 1, size: 200 })])
      .then(([profileItems, nasResult]) => {
        setProfiles(profileItems)
        setNasItems(nasResult.items ?? [])
      })
      .catch(() => undefined)
  }, [canReadRadius])

  const columns: ColumnsType<RadiusBypassItem> = [
    {
      title: t('radius:bypass.columns.type'),
      key: 'type',
      width: 100,
      render: (_: unknown, record) => <Text>{record.type === 'ip' ? t('radius:bypass.form.typeIp') : t('radius:bypass.form.typeMac')}</Text>,
    },
    {
      title: t('radius:bypass.columns.profile'),
      dataIndex: 'profileId',
      key: 'profileId',
      width: 130,
      render: (id: number) => profiles.find((profile) => profile.id === id)?.name ?? `#${id}`,
    },
    {
      title: t('radius:bypass.columns.nas'),
      dataIndex: 'nasId',
      key: 'nasId',
      width: 140,
      responsive: ['md'],
      render: (id?: number | null) => {
        if (!id) return t('radius:bypass.allNas')
        const nas = nasItems.find((item) => item.id === id)
        return nas ? `${nas.name} (${nas.ipaddr})` : `#${id}`
      },
    },
    {
      title: t('radius:bypass.columns.expireTime'),
      dataIndex: 'expireTime',
      key: 'expireTime',
      width: 140,
      responsive: ['lg'],
      render: (value?: string | null) => (value ? renderTime(value) : t('radius:bypass.neverExpires')),
    },
    {
      title: t('radius:bypass.columns.value'),
      dataIndex: 'value',
      key: 'value',
      width: 150,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:common.status'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (val: string) => renderStatusTag(t, val),
    },
    {
      title: t('radius:common.remark'),
      dataIndex: 'remark',
      key: 'remark',
      width: 140,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:bypass.columns.updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 140,
      responsive: ['sm'],
      render: renderTime,
    },
    {
      title: t('radius:common.actions'),
      key: 'actions',
      width: 140,
      align: 'center',
      fixed: 'right',
      render: (_, record) => (
        <Can permission="radius.manage">
          <Space size={4}>
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => openEdit(record)}
            >
              {t('radius:common.edit')}
            </Button>
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record)}
            >
              {t('radius:common.delete')}
            </Button>
          </Space>
        </Can>
      ),
    },
  ]

  const handleSearch = () => {
    setPage(1)
    setKeyword(search.trim())
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ type: 'mac', status: 'enabled' })
    setModalOpen(true)
  }

  const openEdit = (record: RadiusBypassItem) => {
    setEditing(record)
    form.resetFields()
    form.setFieldsValue({
      type: record.type,
      value: record.value,
      profileId: record.profileId,
      nasId: record.nasId ?? undefined,
      expireTime: record.expireTime ? dayjs(record.expireTime) : undefined,
      status: record.status,
      remark: record.remark,
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload: RadiusBypassPayload = {
        type: values.type,
        value: values.value.trim(),
        profileId: values.profileId,
        nasId: values.nasId ?? null,
        expireTime: values.expireTime?.toISOString(),
        status: values.status ?? 'enabled',
        remark: values.remark?.trim() ?? '',
      }
      if (editing) {
        await radiusApi.updateBypass(editing.id, payload)
      } else {
        await radiusApi.createBypass(payload)
      }
      message.success(t('radius:common.saveSuccess'))
      setModalOpen(false)
      setEditing(null)
      await load()
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return // 表单校验失败
      // 其余错误已由拦截器提示
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (record: RadiusBypassItem) => {
    modal.confirm({
      title: t('radius:common.confirmTitle'),
      content: t('radius:bypass.deleteConfirm'),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        await radiusApi.deleteBypass(record.id)
        message.success(t('radius:common.deleteSuccess'))
        await load()
      },
    })
  }

  if (!canReadRadius) {
    return <Result status="403" title="403" subTitle={t('settings:permissionDenied')} />
  }

  return (
    <div>
      <Card variant="outlined">
        <Alert
          type="info"
          showIcon
          title={t('radius:bypass.intro')}
          style={{ marginBottom: 16 }}
        />
        <Toolbar
          left={
            <Can permission="radius.manage">
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                {t('radius:bypass.create')}
              </Button>
            </Can>
          }
          right={
            <>
              <Input.Search
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onSearch={handleSearch}
                placeholder={t('radius:bypass.searchPlaceholder')}
                allowClear
                className="netlab-billing-toolbar-search"
              />
              <Button icon={<ReloadOutlined />} onClick={load} />
            </>
          }
        />

        <div style={{ width: '100%', minWidth: 0, overflow: 'hidden' }}>
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
        </div>
      </Card>

      {/* 新增/编辑免认证规则 */}
      <Modal
        title={editing ? t('radius:bypass.edit') : t('radius:bypass.create')}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditing(null)
        }}
        onOk={handleSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={saving}
        forceRender
        width={{ xs: 'calc(100vw - 32px)', sm: 560 }}
      >
        <Form form={form} layout="vertical" requiredMark={false}>
          <Row gutter={16}>
            <Col xs={24} sm={8}>
              <Form.Item name="type" label={t('radius:bypass.form.type')}>
                <Select
                  options={[
                    { value: 'mac', label: t('radius:bypass.form.typeMac') },
                    { value: 'ip', label: t('radius:bypass.form.typeIp') },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={16}>
              <Form.Item
                name="value"
                label={t('radius:bypass.form.value')}
                tooltip={typeWatch === 'ip' ? t('radius:bypass.form.valueIpTip') : t('radius:bypass.form.valueMacTip')}
                normalize={(value: string) => value?.trim()}
                rules={[
                  { required: true, message: t('radius:bypass.form.valueRequired') },
                  {
                    validator: (_, value?: string) => {
                      if (!value) return Promise.resolve()
                      const valid = typeWatch === 'ip' ? IPV4_PATTERN.test(value) : MAC_PATTERN.test(value)
                      return valid ? Promise.resolve() : Promise.reject(new Error(t(typeWatch === 'ip' ? 'radius:bypass.form.valueIpInvalid' : 'radius:bypass.form.valueMacInvalid')))
                    },
                  },
                ]}
              >
                <Input
                  maxLength={128}
                  placeholder={typeWatch === 'ip' ? t('radius:bypass.form.valueIpPlaceholder') : t('radius:bypass.form.valueMacPlaceholder')}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="profileId"
                label={t('radius:bypass.form.profile')}
                rules={[{ required: true, message: t('radius:bypass.form.profileRequired') }]}
              >
                <Select
                  showSearch={{ optionFilterProp: 'label' }}
                  options={profiles.map((profile) => ({ value: profile.id, label: profile.name }))}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="nasId"
                label={t('radius:bypass.form.nas')}
                rules={typeWatch === 'ip' ? [{ required: true, message: t('radius:bypass.form.nasRequired') }] : []}
              >
                <Select
                  allowClear
                  placeholder={t('radius:bypass.allNas')}
                  options={nasItems.map((nas) => ({ value: nas.id, label: `${nas.name} (${nas.ipaddr})` }))}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="expireTime" label={t('radius:bypass.form.expireTime')}>
                <DatePicker showTime allowClear style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="status" label={t('radius:bypass.form.status')}>
                <Select
                  options={[
                    { value: 'enabled', label: t('radius:common.enabled') },
                    { value: 'disabled', label: t('radius:common.disabled') },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item name="remark" label={t('radius:bypass.form.remark')}>
                <Input.TextArea rows={2} maxLength={255} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  )
}
