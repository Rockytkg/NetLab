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
import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import { formatRate } from '@/pages/billing/format'
import type { RadiusProfileItem, RadiusProfilePayload } from '@/types/radius'

const { Text } = Typography

/** 套餐表单值；InputNumber 清空后为 null，提交时归一化为 0；
 * status 直接以 enabled/disabled 字符串承载（下拉选择），绑定项为布尔下拉。 */
interface ProfileFormValues {
  name: string
  upRate?: number | null
  downRate?: number | null
  activeNum?: number | null
  addrPool?: string
  ipv6PrefixPool?: string
  delegatedIpv6PrefixPool?: string
  domain?: string
  bindMac?: boolean
  bindVlan?: boolean
  status?: string
  remark?: string
}

/** 策略套餐页：分页列表 + 新建/编辑弹窗 + 行内删除。 */
export default function ProfilesPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { token } = theme.useToken()
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadRadius = can('radius.read')

  const [data, setData] = useState<RadiusProfileItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)

  // 新建/编辑共用一个弹窗，editing 为空表示新建
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<RadiusProfileItem | null>(null)
  const [form] = Form.useForm<ProfileFormValues>()
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    if (!canReadRadius) return
    setLoading(true)
    try {
      const res = await radiusApi.listProfiles({ page, size, keyword })
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

  // 可截断列：Typography ellipsis 内置测量，仅在文本真正溢出时悬停显示完整内容
  const renderEllipsis = (val: string) =>
    val ? (
      <Text ellipsis={{ tooltip: val }} style={{ display: 'block' }}>
        {val}
      </Text>
    ) : (
      '-'
    )

  const unlimited = t('radius:common.unlimited')

  const columns: ColumnsType<RadiusProfileItem> = [
    {
      title: t('radius:profiles.columns.name'),
      dataIndex: 'name',
      key: 'name',
      width: 160,
      render: renderEllipsis,
    },
    {
      // 上行 / 下行速率，0 表示不限
      title: t('radius:profiles.columns.rate'),
      key: 'rate',
      width: 170,
      render: (_, record) =>
        `${formatRate(record.upRate, unlimited)} / ${formatRate(record.downRate, unlimited)}`,
    },
    {
      title: t('radius:profiles.columns.activeNum'),
      dataIndex: 'activeNum',
      key: 'activeNum',
      width: 100,
      render: (val: number) => (val > 0 ? val : unlimited),
    },
    {
      title: t('radius:profiles.columns.addrPool'),
      dataIndex: 'addrPool',
      key: 'addrPool',
      width: 150,
      render: renderEllipsis,
    },
    {
      title: t('radius:profiles.columns.bind'),
      key: 'bind',
      width: 130,
      render: (_, record) => {
        const tags: string[] = []
        if (record.bindMac) tags.push('MAC')
        if (record.bindVlan) tags.push('VLAN')
        if (tags.length === 0) return '-'
        return (
          <Space size={token.marginXXS} wrap>
            {tags.map((tag) => (
              <Tag key={tag}>{tag}</Tag>
            ))}
          </Space>
        )
      },
    },
    {
      title: t('radius:profiles.columns.userCount'),
      dataIndex: 'userCount',
      key: 'userCount',
      width: 110,
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
      title: t('radius:profiles.columns.remark'),
      dataIndex: 'remark',
      key: 'remark',
      width: 180,
      render: renderEllipsis,
    },
    {
      title: t('radius:common.actions'),
      key: 'actions',
      width: 140,
      fixed: 'right',
      render: (_, record) => (
        <Can permission="radius.manage">
          <Space size={token.marginXXS}>
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
    setFormOpen(true)
  }

  const openEdit = (item: RadiusProfileItem) => {
    setEditing(item)
    form.resetFields()
    form.setFieldsValue({
      name: item.name,
      upRate: item.upRate,
      downRate: item.downRate,
      activeNum: item.activeNum,
      addrPool: item.addrPool,
      ipv6PrefixPool: item.ipv6PrefixPool,
      delegatedIpv6PrefixPool: item.delegatedIpv6PrefixPool,
      domain: item.domain,
      bindMac: item.bindMac,
      bindVlan: item.bindVlan,
      status: item.status,
      remark: item.remark,
    })
    setFormOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload: RadiusProfilePayload = {
        name: values.name.trim(),
        upRate: values.upRate ?? 0,
        downRate: values.downRate ?? 0,
        activeNum: values.activeNum ?? 0,
        addrPool: values.addrPool?.trim() ?? '',
        ipv6PrefixPool: values.ipv6PrefixPool?.trim() ?? '',
        delegatedIpv6PrefixPool: values.delegatedIpv6PrefixPool?.trim() ?? '',
        domain: values.domain?.trim() ?? '',
        bindMac: values.bindMac ?? false,
        bindVlan: values.bindVlan ?? false,
        status: values.status ?? 'enabled',
        remark: values.remark?.trim() ?? '',
      }
      if (editing) {
        await radiusApi.updateProfile(editing.id, payload)
      } else {
        await radiusApi.createProfile(payload)
      }
      message.success(t('radius:common.saveSuccess'))
      setFormOpen(false)
      setEditing(null)
      await load()
    } catch (err) {
      // 表单校验失败无需提示；其余错误已由拦截器提示
      if ((err as { errorFields?: unknown }).errorFields) return
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (item: RadiusProfileItem) => {
    modal.confirm({
      title: t('radius:common.confirmTitle'),
      content: t('radius:profiles.deleteConfirm'),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        // 套餐仍被用户引用时后端返回错误，拦截器会提示
        await radiusApi.deleteProfile(item.id)
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
        <Space
          style={{ marginBottom: token.margin, width: '100%', justifyContent: 'space-between' }}
          wrap
        >
          <Space wrap>
            <Can permission="radius.manage">
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                {t('radius:profiles.create')}
              </Button>
            </Can>
          </Space>
          <Space wrap>
            <Input.Search
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onSearch={handleSearch}
              placeholder={t('radius:profiles.searchPlaceholder')}
              allowClear
              style={{ width: 220 }}
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
          // 列宽合计 1230：容器更宽时按比例分配，更窄时横向滚动；空数据不启用横向滚动
          scroll={data.length > 0 ? { x: 1230 } : undefined}
          tableLayout="fixed"
        />
      </Card>

      {/* 新建/编辑套餐 */}
      <Modal
        title={editing ? t('radius:profiles.edit') : t('radius:profiles.create')}
        open={formOpen}
        onCancel={() => {
          setFormOpen(false)
          setEditing(null)
        }}
        onOk={handleSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={saving}
        forceRender
        width={720}
      >
        <Form
          form={form}
          layout="vertical"
          requiredMark={false}
          initialValues={{
            upRate: 1024,
            downRate: 1024,
            activeNum: 1,
            bindMac: false,
            bindVlan: false,
            status: 'enabled',
          }}
        >
          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item
                name="name"
                label={t('radius:profiles.form.name')}
                normalize={(value: string) => value?.trim()}
                rules={[{ required: true, message: t('radius:profiles.form.nameRequired') }]}
              >
                <Input maxLength={64} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="status" label={t('radius:profiles.form.status')}>
                <Select
                  options={[
                    { value: 'enabled', label: t('radius:common.enabled') },
                    { value: 'disabled', label: t('radius:common.disabled') },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item name="upRate" label={t('radius:profiles.form.upRate')}>
                <InputNumber min={0} precision={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item name="downRate" label={t('radius:profiles.form.downRate')}>
                <InputNumber min={0} precision={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item name="activeNum" label={t('radius:profiles.form.activeNum')}>
                <InputNumber min={0} precision={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="domain" label={t('radius:profiles.form.domain')}>
                <Input maxLength={64} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="addrPool" label={t('radius:profiles.form.addrPool')}>
                <Input maxLength={64} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="ipv6PrefixPool"
                label={t('radius:profiles.form.ipv6PrefixPool')}
                tooltip={t('radius:profiles.form.ipv6PrefixPoolTip')}
              >
                <Input maxLength={64} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="delegatedIpv6PrefixPool"
                label={t('radius:profiles.form.delegatedIpv6PrefixPool')}
                tooltip={t('radius:profiles.form.delegatedIpv6PrefixPoolTip')}
                rules={[
                  ({ getFieldValue }) => ({
                    validator(_, value?: string) {
                      const other = (getFieldValue('ipv6PrefixPool') as string | undefined)?.trim()
                      const current = value?.trim()
                      if (current && other && current === other) {
                        return Promise.reject(new Error(t('radius:profiles.form.poolsDuplicate')))
                      }
                      return Promise.resolve()
                    },
                  }),
                ]}
              >
                <Input maxLength={64} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="bindMac" label={t('radius:profiles.form.bindMac')}>
                <Select
                  options={[
                    { value: true, label: t('radius:common.enabled') },
                    { value: false, label: t('radius:common.disabled') },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="bindVlan" label={t('radius:profiles.form.bindVlan')}>
                <Select
                  options={[
                    { value: true, label: t('radius:common.enabled') },
                    { value: false, label: t('radius:common.disabled') },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item name="remark" label={t('radius:profiles.form.remark')}>
                <Input.TextArea maxLength={255} rows={3} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  )
}
