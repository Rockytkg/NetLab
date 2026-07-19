import { useCallback, useEffect, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Form,
  Input,
  Modal,
  Result,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import type { RadiusBypassItem, RadiusBypassPayload, RadiusBypassType } from '@/types/radius'

const { Text } = Typography

/** 免认证表单值。 */
interface BypassFormValues {
  type: RadiusBypassType
  value: string
  status: string
  remark?: string
}

const MAC_PATTERN = /^[0-9a-fA-F]{2}([:-][0-9a-fA-F]{2}){5}$/

/** 校验 IPv4/IPv6 地址或 CIDR 网段。 */
const isValidIpOrCidr = (raw: string): boolean => {
  const [ip, prefix, ...rest] = raw.split('/')
  if (rest.length > 0 || !ip) return false
  const isV4 =
    /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.test(ip) &&
    ip.split('.').every((part) => Number(part) <= 255)
  const isV6 = ip.includes(':') && /^[0-9a-fA-F:]+$/.test(ip)
  if (!isV4 && !isV6) return false
  if (prefix === undefined) return true
  if (!/^\d+$/.test(prefix)) return false
  const max = isV4 ? 32 : 128
  const num = Number(prefix)
  return num >= 0 && num <= max
}

/** 免认证规则页：命中规则的终端跳过认证直接放行（MAC 或 IP 匹配）。 */
export default function BypassPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { token } = theme.useToken()
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

  // 可截断列：仅在文本真正溢出时悬停显示完整内容
  const renderEllipsis = (val?: string | null) =>
    val ? (
      <Text ellipsis={{ tooltip: val }} style={{ display: 'block' }}>
        {val}
      </Text>
    ) : (
      '-'
    )

  const columns: ColumnsType<RadiusBypassItem> = [
    {
      title: t('radius:bypass.columns.type'),
      dataIndex: 'type',
      key: 'type',
      width: 140,
      render: (val: RadiusBypassType) => (
        <Tag color={val === 'mac' ? 'geekblue' : 'purple'}>
          {val === 'mac' ? t('radius:bypass.form.typeMac') : t('radius:bypass.form.typeIp')}
        </Tag>
      ),
    },
    {
      title: t('radius:bypass.columns.value'),
      dataIndex: 'value',
      key: 'value',
      width: 220,
      render: (val: string) => renderEllipsis(val),
    },
    {
      title: t('radius:common.status'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (val: string) => (
        <Tag color={val === 'enabled' ? 'success' : 'error'}>
          {val === 'enabled' ? t('radius:common.enabled') : t('radius:common.disabled')}
        </Tag>
      ),
    },
    {
      title: t('radius:common.remark'),
      dataIndex: 'remark',
      key: 'remark',
      width: 200,
      render: (val: string) => renderEllipsis(val),
    },
    {
      title: t('radius:bypass.columns.updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 170,
      render: (val: string) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
    {
      title: t('radius:common.actions'),
      key: 'actions',
      width: 150,
      fixed: 'right',
      render: (_, record) => (
        <Space size={token.marginXXS}>
          <Can permission="radius.manage">
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
          </Can>
        </Space>
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
    <div style={{ width: '100%' }}>
      <Card variant="outlined">
        <Alert
          type="info"
          showIcon
          title={t('radius:bypass.intro')}
          style={{ marginBottom: token.margin }}
        />
        <Space
          style={{ marginBottom: token.margin, width: '100%', justifyContent: 'space-between' }}
          wrap
        >
          <Space wrap>
            <Can permission="radius.manage">
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                {t('radius:bypass.create')}
              </Button>
            </Can>
          </Space>
          <Space wrap>
            <Input.Search
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onSearch={handleSearch}
              placeholder={t('radius:bypass.searchPlaceholder')}
              allowClear
              style={{ width: 240 }}
            />
            <Button icon={<ReloadOutlined />} onClick={load} />
          </Space>
        </Space>

        <Table
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
          // 列宽合计 970：容器更宽时按比例分配，更窄时横向滚动；空数据不启用横向滚动
          scroll={data.length > 0 ? { x: 970 } : undefined}
          tableLayout="fixed"
        />
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
        width={560}
      >
        <Form form={form} layout="vertical" requiredMark={false}>
          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item name="type" label={t('radius:bypass.form.type')}>
                <Select
                  options={[
                    { value: 'mac', label: t('radius:bypass.form.typeMac') },
                    { value: 'ip', label: t('radius:bypass.form.typeIp') },
                  ]}
                />
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
              <Form.Item
                name="value"
                label={t('radius:bypass.form.value')}
                tooltip={
                  typeWatch === 'ip'
                    ? t('radius:bypass.form.valueIpTip')
                    : t('radius:bypass.form.valueMacTip')
                }
                normalize={(value: string) => value?.trim()}
                rules={[
                  { required: true, message: t('radius:bypass.form.valueRequired') },
                  {
                    validator: (_, value?: string) => {
                      if (!value) return Promise.resolve()
                      if (typeWatch === 'ip') {
                        return isValidIpOrCidr(value)
                          ? Promise.resolve()
                          : Promise.reject(new Error(t('radius:bypass.form.valueIpInvalid')))
                      }
                      return MAC_PATTERN.test(value)
                        ? Promise.resolve()
                        : Promise.reject(new Error(t('radius:bypass.form.valueMacInvalid')))
                    },
                  },
                ]}
              >
                <Input
                  maxLength={128}
                  placeholder={
                    typeWatch === 'ip'
                      ? t('radius:bypass.form.valueIpPlaceholder')
                      : t('radius:bypass.form.valueMacPlaceholder')
                  }
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
