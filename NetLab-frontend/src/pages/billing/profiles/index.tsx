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
  Tabs,
  Typography,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import Toolbar from '@/pages/billing/components/Toolbar'
import { formatRate } from '@/pages/billing/format'
import { renderStatusTag } from '@/pages/billing/shared'
import type { RadiusProfileItem, RadiusProfilePayload } from '@/types/radius'

const { Text } = Typography

/** 套餐表单值；InputNumber 清空后为 null，提交时归一化为 0。 */
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

  const unlimited = t('radius:common.unlimited')

  const columns: ColumnsType<RadiusProfileItem> = [
    {
      title: t('radius:profiles.columns.name'),
      dataIndex: 'name',
      key: 'name',
      width: 160,
      ellipsis: { showTitle: true },
    },
    {
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
      width: 120,
      render: (val: number) => (val > 0 ? val : unlimited),
    },
    {
      title: t('radius:profiles.columns.addrPool'),
      dataIndex: 'addrPool',
      key: 'addrPool',
      width: 140,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:profiles.columns.bind'),
      key: 'bind',
      width: 130,
      responsive: ['sm'],
      render: (_, record) => {
        const tags: string[] = []
        if (record.bindMac) tags.push('MAC')
        if (record.bindVlan) tags.push('VLAN')
        if (tags.length === 0) return '-'
        return (
          <span style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
            {tags.map((tag) => (
              <Text key={tag} code>{tag}</Text>
            ))}
          </span>
        )
      },
    },
    {
      title: t('radius:profiles.columns.userCount'),
      dataIndex: 'userCount',
      key: 'userCount',
      width: 110,
      responsive: ['sm'],
    },
    {
      title: t('radius:common.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (val: string) => renderStatusTag(t, val),
    },
    {
      title: t('radius:profiles.columns.remark'),
      dataIndex: 'remark',
      key: 'remark',
      width: 160,
      ellipsis: { showTitle: true },
      responsive: ['md'],
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
    <div>
      <Card variant="outlined">
        <Toolbar
          left={
            <Can permission="radius.manage">
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                {t('radius:profiles.create')}
              </Button>
            </Can>
          }
          right={
            <>
              <Input.Search
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onSearch={handleSearch}
                placeholder={t('radius:profiles.searchPlaceholder')}
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
        width={{ xs: 'calc(100vw - 32px)', sm: 560, md: 720 }}
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
          <Tabs
            defaultActiveKey="basic"
            items={[
              {
                key: 'basic', label: t('radius:profiles.sections.basic'), children: <Row gutter={[16, 0]}>
                  <Col xs={24} sm={12}><Form.Item name="name" label={t('radius:profiles.form.name')} normalize={(value: string) => value?.trim()} rules={[{ required: true, message: t('radius:profiles.form.nameRequired') }]}><Input maxLength={64} /></Form.Item></Col>
                  <Col xs={24} sm={12}><Form.Item name="status" label={t('radius:profiles.form.status')}><Select options={[{ value: 'enabled', label: t('radius:common.enabled') }, { value: 'disabled', label: t('radius:common.disabled') }]} /></Form.Item></Col>
                  <Col xs={24}><Form.Item name="domain" label={t('radius:profiles.form.domain')}><Input maxLength={64} /></Form.Item></Col>
                  <Col xs={24}><Form.Item name="remark" label={t('radius:profiles.form.remark')}><Input.TextArea maxLength={255} rows={2} /></Form.Item></Col>
                </Row>,
              },
              {
                key: 'service', label: t('radius:profiles.sections.service'), children: <Row gutter={[16, 0]}>
                  <Col xs={24} sm={8}><Form.Item name="upRate" label={t('radius:profiles.form.upRate')}><InputNumber min={0} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
                  <Col xs={24} sm={8}><Form.Item name="downRate" label={t('radius:profiles.form.downRate')}><InputNumber min={0} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
                  <Col xs={24} sm={8}><Form.Item name="activeNum" label={t('radius:profiles.form.activeNum')}><InputNumber min={0} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
                  <Col xs={24} sm={12}><Form.Item name="bindMac" label={t('radius:profiles.form.bindMac')}><Select options={[{ value: true, label: t('radius:common.enabled') }, { value: false, label: t('radius:common.disabled') }]} /></Form.Item></Col>
                  <Col xs={24} sm={12}><Form.Item name="bindVlan" label={t('radius:profiles.form.bindVlan')}><Select options={[{ value: true, label: t('radius:common.enabled') }, { value: false, label: t('radius:common.disabled') }]} /></Form.Item></Col>
                </Row>,
              },
              {
                key: 'network', label: t('radius:profiles.sections.network'), children: <Row gutter={[16, 0]}>
                  <Col xs={24} sm={12}><Form.Item name="addrPool" label={t('radius:profiles.form.addrPool')}><Input maxLength={64} /></Form.Item></Col>
                  <Col xs={24} sm={12}><Form.Item name="ipv6PrefixPool" label={t('radius:profiles.form.ipv6PrefixPool')} tooltip={t('radius:profiles.form.ipv6PrefixPoolTip')}><Input maxLength={64} /></Form.Item></Col>
                  <Col xs={24}><Form.Item name="delegatedIpv6PrefixPool" label={t('radius:profiles.form.delegatedIpv6PrefixPool')} tooltip={t('radius:profiles.form.delegatedIpv6PrefixPoolTip')} rules={[({ getFieldValue }) => ({ validator(_, value?: string) { const other = (getFieldValue('ipv6PrefixPool') as string | undefined)?.trim(); const current = value?.trim(); return current && other && current === other ? Promise.reject(new Error(t('radius:profiles.form.poolsDuplicate'))) : Promise.resolve() } })]}><Input maxLength={64} /></Form.Item></Col>
                </Row>,
              },
            ]}
          />
        </Form>
      </Modal>
    </div>
  )
}
