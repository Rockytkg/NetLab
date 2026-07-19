import { useCallback, useEffect, useState } from 'react'
import {
  App,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
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
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import { RADIUS_VENDOR_CODES, type RadiusNasItem, type RadiusNasPayload } from '@/types/radius'

const { Text } = Typography

/** NAS 设备表单值；secret 仅在填写时提交（编辑留空表示不修改）。 */
type NasFormValues = {
  name: string
  vendorCode?: string
  ipaddr: string
  identifier?: string
  hostname?: string
  secret?: string
  coaPort?: number
  model?: string
  status?: string
  tags?: string
  remark?: string
}

/** NAS 设备管理页：分页列表 + 关键词搜索 + 新增/编辑/删除。 */
export default function NasPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { token } = theme.useToken()
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadNas = can('radius.read')

  const [data, setData] = useState<RadiusNasItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)

  // 新增/编辑弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editingNas, setEditingNas] = useState<RadiusNasItem | null>(null)
  const [form] = Form.useForm<NasFormValues>()
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    if (!canReadNas) return
    setLoading(true)
    try {
      const res = await radiusApi.listNas({ page, size, keyword })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadNas, page, size, keyword])

  useEffect(() => {
    load()
  }, [load])

  // 厂商代码 → 展示名；未知代码原样展示，空代码命中「标准/默认」
  const vendorLabel = (code: string) => {
    const hit = RADIUS_VENDOR_CODES.find((v) => v.value === code)
    return hit ? t(hit.labelKey) : code || '-'
  }

  // 可截断列：Typography ellipsis 内置测量，仅当文本真正溢出被截断时悬停才显示完整内容提示
  const renderEllipsis = (val?: string | null) =>
    val ? (
      <Text ellipsis={{ tooltip: val }} style={{ display: 'block' }}>
        {val}
      </Text>
    ) : (
      '-'
    )

  const columns: ColumnsType<RadiusNasItem> = [
    {
      title: t('radius:nas.columns.name'),
      dataIndex: 'name',
      key: 'name',
      width: 140,
      render: (val: string) => renderEllipsis(val),
    },
    {
      title: t('radius:nas.columns.vendor'),
      dataIndex: 'vendorCode',
      key: 'vendorCode',
      width: 110,
      render: (val: string) => <Tag>{vendorLabel(val)}</Tag>,
    },
    {
      title: t('radius:nas.columns.ipaddr'),
      dataIndex: 'ipaddr',
      key: 'ipaddr',
      width: 130,
      render: (val: string) => renderEllipsis(val),
    },
    {
      title: t('radius:nas.columns.identifier'),
      dataIndex: 'identifier',
      key: 'identifier',
      width: 140,
      render: (val: string) => renderEllipsis(val),
    },
    {
      title: t('radius:nas.columns.model'),
      dataIndex: 'model',
      key: 'model',
      width: 120,
      render: (val: string) => renderEllipsis(val),
    },
    {
      title: t('radius:nas.columns.coaPort'),
      dataIndex: 'coaPort',
      key: 'coaPort',
      width: 100,
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
      width: 160,
      render: (val: string) => renderEllipsis(val),
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
              {t('common:edit')}
            </Button>
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record)}
            >
              {t('common:delete')}
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
    setEditingNas(null)
    form.resetFields()
    form.setFieldsValue({ vendorCode: '', coaPort: 3799, status: 'enabled' })
    setModalOpen(true)
  }

  const openEdit = (record: RadiusNasItem) => {
    setEditingNas(record)
    form.resetFields()
    form.setFieldsValue({
      name: record.name,
      vendorCode: record.vendorCode,
      ipaddr: record.ipaddr,
      identifier: record.identifier,
      hostname: record.hostname,
      secret: undefined,
      coaPort: record.coaPort || 3799,
      model: record.model,
      status: record.status,
      tags: record.tags,
      remark: record.remark,
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload: RadiusNasPayload = {
        name: values.name.trim(),
        vendorCode: values.vendorCode ?? '',
        ipaddr: values.ipaddr.trim(),
        identifier: values.identifier?.trim() ?? '',
        hostname: values.hostname?.trim() ?? '',
        coaPort: values.coaPort ?? 3799,
        model: values.model?.trim() ?? '',
        status: values.status ?? 'enabled',
        tags: values.tags?.trim() ?? '',
        remark: values.remark?.trim() ?? '',
      }
      const secret = values.secret?.trim()
      if (secret) {
        payload.secret = secret
      }
      if (editingNas) {
        await radiusApi.updateNas(editingNas.id, payload)
        message.success(t('radius:common.saveSuccess'))
        setModalOpen(false)
        setEditingNas(null)
        await load()
      } else {
        await radiusApi.createNas(payload)
        message.success(t('radius:common.saveSuccess'))
        setModalOpen(false)
        form.resetFields()
        const reloadCurrentPage = page === 1
        setPage(1)
        if (reloadCurrentPage) {
          await load()
        }
      }
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return // 表单校验失败
      // 其余错误已由拦截器提示
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (record: RadiusNasItem) => {
    modal.confirm({
      title: t('radius:common.confirmTitle'),
      content: t('radius:nas.deleteConfirm'),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        await radiusApi.deleteNas(record.id)
        message.success(t('radius:common.deleteSuccess'))
        await load()
      },
    })
  }

  // 生成 16 位字母数字随机共享密钥
  const handleGenerateSecret = () => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
    const bytes = new Uint8Array(16)
    crypto.getRandomValues(bytes)
    form.setFieldValue('secret', Array.from(bytes, (b) => chars[b % chars.length]).join(''))
  }

  if (!canReadNas) {
    return <Result status="403" title="403" subTitle={t('settings:permissionDenied')} />
  }

  return (
    <div style={{ width: '100%' }}>
      <Card variant="outlined">
        <Space
          style={{ marginBottom: token.margin, width: '100%', justifyContent: 'space-between' }}
          wrap
        >
          <Space wrap>
            <Can permission="radius.manage">
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                {t('radius:nas.create')}
              </Button>
            </Can>
          </Space>
          <Space wrap>
            <Input.Search
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onSearch={handleSearch}
              placeholder={t('radius:nas.searchPlaceholder')}
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
            // radius 命名空间无 total 文案，复用 settings 的通用「共 N 条记录」
            showTotal: (tt) => t('settings:loginLogs.total', { total: tt }),
          }}
          // 列宽合计 1140：容器更宽时按比例分配，更窄时横向滚动；空数据不启用横向滚动
          scroll={data.length > 0 ? { x: 1140 } : undefined}
          tableLayout="fixed"
        />
      </Card>

      {/* 新增/编辑 NAS 设备 */}
      <Modal
        title={editingNas ? t('radius:nas.edit') : t('radius:nas.create')}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditingNas(null)
          form.resetFields()
        }}
        onOk={handleSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={saving}
        forceRender
        width={640}
      >
        <Form form={form} layout="vertical" requiredMark={false}>
          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item
                name="name"
                label={t('radius:nas.form.name')}
                rules={[{ required: true, message: t('radius:nas.form.nameRequired') }]}
              >
                <Input maxLength={64} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="vendorCode" label={t('radius:nas.form.vendorCode')}>
                <Select
                  showSearch
                  optionFilterProp="label"
                  options={RADIUS_VENDOR_CODES.map((v) => ({ value: v.value, label: t(v.labelKey) }))}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="ipaddr"
                label={t('radius:nas.form.ipaddr')}
                rules={[{ required: true, message: t('radius:nas.form.ipaddrRequired') }]}
              >
                <Input maxLength={128} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="identifier"
                label={t('radius:nas.form.identifier')}
                tooltip={t('radius:nas.form.identifierTip')}
              >
                <Input maxLength={128} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="hostname" label={t('radius:nas.form.hostname')}>
                <Input maxLength={128} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="model" label={t('radius:nas.form.model')}>
                <Input maxLength={64} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="coaPort" label={t('radius:nas.form.coaPort')}>
                <InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="status" label={t('radius:nas.form.status')}>
                <Select
                  options={[
                    { value: 'enabled', label: t('radius:common.enabled') },
                    { value: 'disabled', label: t('radius:common.disabled') },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item label={t('radius:nas.form.secret')} required={!editingNas}>
                <Space.Compact block>
                  <Form.Item
                    name="secret"
                    noStyle
                    rules={[
                      // 编辑留空表示不修改；async-validator 对空值跳过非 required 规则
                      ...(editingNas
                        ? []
                        : [{ required: true, message: t('radius:nas.form.secretRequired') }]),
                      { min: 6, message: t('radius:nas.form.secretMin') },
                    ]}
                  >
                    <Input.Password
                      maxLength={128}
                      autoComplete="new-password"
                      placeholder={editingNas ? t('radius:nas.form.secretEditTip') : undefined}
                    />
                  </Form.Item>
                  <Button onClick={handleGenerateSecret}>
                    {t('radius:nas.form.generateSecret')}
                  </Button>
                </Space.Compact>
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item name="tags" label={t('radius:nas.form.tags')}>
                <Input maxLength={128} />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item name="remark" label={t('radius:nas.form.remark')}>
                <Input.TextArea rows={3} maxLength={255} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  )
}
