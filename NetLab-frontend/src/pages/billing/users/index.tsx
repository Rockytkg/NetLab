import { useCallback, useEffect, useState } from 'react'
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
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
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import dayjs, { type Dayjs } from 'dayjs'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import Toolbar from '@/pages/billing/components/Toolbar'
import { formatRate } from '../format'
import { renderStatusTag, renderTime } from '@/pages/billing/shared'
import type { RadiusProfileOption, RadiusUserItem, RadiusUserPayload } from '@/types/radius'

const { Text } = Typography

/** 编辑表单值：expireTime 用 Dayjs 承载，提交时再转 ISO 字符串。 */
interface UserFormValues {
  username: string
  password?: string
  realname?: string
  status?: string
  email?: string
  mobile?: string
  address?: string
  profileId?: number
  profileLinkMode: number
  expireTime?: Dayjs
  ipAddr?: string
  ipV6Addr?: string
  addrPool?: string
  ipv6PrefixPool?: string
  delegatedIpv6Prefix?: string
  delegatedIpv6PrefixPool?: string
  domain?: string
  upRate?: number
  downRate?: number
  activeNum?: number
  bindMac?: boolean
  macAddr?: string
  bindVlan?: boolean
  vlanid1?: number
  vlanid2?: number
  remark?: string
}

const MAC_PATTERN = /^[0-9a-fA-F]{2}([:-][0-9a-fA-F]{2}){5}$/

const splitMacInput = (raw?: string): string[] => {
  if (!raw) return []
  const seen = new Set<string>()
  const result: string[] = []
  raw
    .split(/[\s,;]+/)
    .map((item) => item.trim())
    .filter(Boolean)
    .forEach((item) => {
      const mac = item.replace(/-/g, ':').toLowerCase()
      if (!seen.has(mac)) {
        seen.add(mac)
        result.push(mac)
      }
    })
  return result
}

const normalizeMacInput = (raw?: string): string => splitMacInput(raw).join(',')

const macListToLines = (raw?: string): string | undefined => {
  const items = splitMacInput(raw)
  return items.length ? items.join('\n') : undefined
}

const FIELD_SECTION: Record<string, string> = {
  username: 'auth',
  password: 'auth',
  macAddr: 'binding',
}

/** RADIUS 认证用户页：分页列表 + 状态筛选 + 创建/编辑/删除。 */
export default function RadiusUsersPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadRadius = can('radius.read')

  const [data, setData] = useState<RadiusUserItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [loading, setLoading] = useState(false)

  const [modalOpen, setModalOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<RadiusUserItem | null>(null)
  const [form] = Form.useForm<UserFormValues>()
  const [saving, setSaving] = useState(false)
  const [profileOptions, setProfileOptions] = useState<RadiusProfileOption[]>([])
  const [activeSection, setActiveSection] = useState('auth')
  const bindMacOn = Form.useWatch('bindMac', form)
  const bindVlanOn = Form.useWatch('bindVlan', form)

  const load = useCallback(async () => {
    if (!canReadRadius) return
    setLoading(true)
    try {
      const res = await radiusApi.listUsers({
        page,
        size,
        keyword: keyword || undefined,
        status: statusFilter || undefined,
      })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadRadius, page, size, keyword, statusFilter])

  useEffect(() => {
    load()
  }, [load])

  const unlimited = t('radius:common.unlimited')

  const columns: ColumnsType<RadiusUserItem> = [
    {
      title: t('radius:users.columns.username'),
      dataIndex: 'username',
      key: 'username',
      width: 130,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:users.columns.realname'),
      dataIndex: 'realname',
      key: 'realname',
      width: 100,
      ellipsis: { showTitle: true },
      responsive: ['xxl'],
    },
    {
      title: t('radius:users.columns.mobile'),
      dataIndex: 'mobile',
      key: 'mobile',
      width: 120,
      responsive: ['xxl'],
      render: (val: string) => val || '-',
    },
    {
      title: t('radius:users.columns.profile'),
      dataIndex: 'profileName',
      key: 'profileName',
      width: 120,
      ellipsis: { showTitle: true },
      responsive: ['sm'],
    },
    {
      title: t('radius:users.columns.rate'),
      key: 'rate',
      width: 150,
      render: (_, record) =>
        `${formatRate(record.upRate, unlimited)} / ${formatRate(record.downRate, unlimited)}`,
    },
    {
      title: t('radius:users.columns.activeNum'),
      dataIndex: 'activeNum',
      key: 'activeNum',
      width: 100,
      render: (val: number) => (val > 0 ? val : unlimited),
    },
    {
      title: t('radius:users.columns.macAddr'),
      dataIndex: 'macAddr',
      key: 'macAddr',
      width: 110,
      ellipsis: { showTitle: true },
      responsive: ['md'],
      render: (val: string) =>
        val ? <Text ellipsis>{val.split(',').join(', ')}</Text> : '-',
    },
    {
      title: t('radius:users.columns.expireTime'),
      dataIndex: 'expireTime',
      key: 'expireTime',
      width: 140,
      render: renderTime,
    },
    {
      title: t('radius:users.columns.status'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (val: string) => renderStatusTag(t, val),
    },
    {
      title: t('radius:users.columns.online'),
      dataIndex: 'onlineCount',
      key: 'onlineCount',
      width: 90,
      render: (val: number) => <Text type={val > 0 ? 'success' : 'secondary'}>{val}</Text>,
    },
    {
      title: t('radius:users.columns.lastOnline'),
      dataIndex: 'lastOnline',
      key: 'lastOnline',
      width: 150,
      responsive: ['xxl'],
      render: renderTime,
    },
    {
      title: t('radius:common.actions'),
      key: 'actions',
      width: 140,
      align: 'center',
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
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

  const loadProfileOptions = async () => {
    try {
      const options = await radiusApi.listProfileOptions()
      setProfileOptions(options ?? [])
    } catch {
      // 拦截器已提示错误
    }
  }

  const openCreate = () => {
    setEditingUser(null)
    form.resetFields()
    form.setFieldsValue({
      username: undefined,
      password: undefined,
      status: 'enabled',
      profileId: undefined,
      profileLinkMode: 1,
      upRate: 0,
      downRate: 0,
      activeNum: 0,
      bindMac: false,
      bindVlan: false,
      vlanid1: 0,
      vlanid2: 0,
      expireTime: undefined,
    })
    loadProfileOptions()
    setModalOpen(true)
  }

  const openEdit = (item: RadiusUserItem) => {
    setEditingUser(item)
    form.resetFields()
    form.setFieldsValue({
      username: item.username,
      password: undefined,
      realname: item.realname,
      status: item.status,
      email: item.email,
      mobile: item.mobile,
      address: item.address,
      profileId: item.profileId ?? undefined,
      profileLinkMode: item.profileLinkMode,
      expireTime: item.expireTime ? dayjs(item.expireTime) : undefined,
      ipAddr: item.ipAddr,
      ipV6Addr: item.ipV6Addr,
      addrPool: item.addrPool,
      ipv6PrefixPool: item.ipv6PrefixPool,
      delegatedIpv6Prefix: item.delegatedIpv6Prefix,
      delegatedIpv6PrefixPool: item.delegatedIpv6PrefixPool,
      domain: item.domain,
      upRate: item.upRate,
      downRate: item.downRate,
      activeNum: item.activeNum,
      bindMac: item.bindMac,
      macAddr: macListToLines(item.macAddr),
      bindVlan: item.bindVlan,
      vlanid1: item.vlanid1,
      vlanid2: item.vlanid2,
      remark: item.remark,
    })
    loadProfileOptions()
    setModalOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload: RadiusUserPayload = {
        username: values.username.trim(),
        status: values.status ?? 'enabled',
        profileId: values.profileId ?? null,
        profileLinkMode: values.profileLinkMode,
        realname: values.realname ?? '',
        email: values.email ?? '',
        mobile: values.mobile ?? '',
        address: values.address ?? '',
        upRate: values.upRate ?? 0,
        downRate: values.downRate ?? 0,
        activeNum: values.activeNum ?? 0,
        ipAddr: values.ipAddr ?? '',
        ipV6Addr: values.ipV6Addr ?? '',
        addrPool: values.addrPool ?? '',
        ipv6PrefixPool: values.ipv6PrefixPool ?? '',
        delegatedIpv6Prefix: values.delegatedIpv6Prefix ?? '',
        delegatedIpv6PrefixPool: values.delegatedIpv6PrefixPool ?? '',
        domain: values.domain ?? '',
        bindMac: values.bindMac ?? false,
        bindVlan: values.bindVlan ?? false,
        macAddr: normalizeMacInput(values.macAddr),
        vlanid1: values.vlanid1 ?? 0,
        vlanid2: values.vlanid2 ?? 0,
        remark: values.remark ?? '',
      }
      if (values.expireTime) {
        payload.expireTime = values.expireTime.toISOString()
      }
      if (values.password) {
        payload.password = values.password
      }
      if (editingUser) {
        await radiusApi.updateUser(editingUser.id, payload)
      } else {
        await radiusApi.createUser(payload)
      }
      message.success(t('radius:common.saveSuccess'))
      setModalOpen(false)
      setEditingUser(null)
      await load()
    } catch (err) {
      const validation = err as { errorFields?: { name?: (string | number)[] }[] }
      const firstField = validation.errorFields?.[0]?.name?.[0]
      if (firstField !== undefined) {
        const section = FIELD_SECTION[String(firstField)]
        if (section) setActiveSection(section)
        return
      }
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (item: RadiusUserItem) => {
    modal.confirm({
      title: t('radius:common.confirmTitle'),
      content: t('radius:users.deleteConfirm'),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        await radiusApi.deleteUser(item.id)
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
                {t('radius:users.create')}
              </Button>
            </Can>
          }
          right={
            <>
              <Select
                value={statusFilter}
                onChange={(val) => {
                  setPage(1)
                  setStatusFilter(val)
                }}
                className="netlab-billing-toolbar-select"
                options={[
                  { value: '', label: t('radius:common.status') },
                  { value: 'enabled', label: t('radius:common.enabled') },
                  { value: 'disabled', label: t('radius:common.disabled') },
                ]}
              />
              <Input.Search
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onSearch={handleSearch}
                placeholder={t('radius:users.searchPlaceholder')}
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
            showTotal: (tt) => t('settings:users.total', { total: tt }),
          }}
          scroll={{ x: 1250 }}
          tableLayout="fixed"
        />
      </Card>

      {/* 创建/编辑认证用户 */}
      <Modal
        title={editingUser ? t('radius:users.edit') : t('radius:users.create')}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditingUser(null)
        }}
        onOk={handleSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={saving}
        forceRender
        width={{ xs: 'calc(100vw - 32px)', sm: 560, md: 720 }}
      >
        <Form form={form} layout="vertical" requiredMark={false}>
          <Tabs
            activeKey={activeSection}
            onChange={setActiveSection}
            items={[
              {
                key: 'auth',
                label: t('radius:users.sections.auth'),
                children: (
                  <Row gutter={16}>
                    <Col xs={24} sm={12}>
                      <Form.Item
                        name="username"
                        label={t('radius:users.form.username')}
                        normalize={(value: string) => value?.trim()}
                        rules={[{ required: true, message: t('radius:users.form.usernameRequired') }]}
                      >
                        <Input maxLength={64} autoComplete="off" disabled={!!editingUser} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={12}>
                      <Form.Item
                        name="password"
                        label={t('radius:users.form.password')}
                        rules={
                          editingUser
                            ? []
                            : [{ required: true, message: t('radius:users.form.passwordRequired') }]
                        }
                      >
                        <Input.Password
                          maxLength={128}
                          autoComplete="new-password"
                          placeholder={editingUser ? t('radius:users.form.passwordEditTip') : undefined}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={12}>
                      <Form.Item name="realname" label={t('radius:users.form.realname')}>
                        <Input maxLength={64} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={12}>
                      <Form.Item name="status" label={t('radius:users.form.status')}>
                        <Select
                          options={[
                            { value: 'enabled', label: t('radius:common.enabled') },
                            { value: 'disabled', label: t('radius:common.disabled') },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'service',
                label: t('radius:users.sections.service'),
                children: (
                  <Row gutter={16}>
                    <Col xs={24} sm={8}>
                      <Form.Item
                        name="profileId"
                        label={t('radius:users.form.profile')}
                        tooltip={t('radius:users.form.profileTip')}
                      >
                        <Select
                          allowClear
                          options={profileOptions.map((option) => ({ value: option.id, label: option.name }))}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item name="profileLinkMode" label={t('radius:users.form.linkMode')}>
                        <Select
                          options={[
                            { value: 1, label: t('radius:users.form.linkModeDynamic') },
                            { value: 0, label: t('radius:users.form.linkModeStatic') },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item
                        name="expireTime"
                        label={t('radius:users.form.expireTime')}
                        extra={t('radius:users.form.expireTimeTip')}
                      >
                        <DatePicker
                          showTime
                          style={{ width: '100%' }}
                          placeholder={t('radius:users.form.expireTimeTip')}
                          presets={[
                            { label: t('radius:users.form.expirePresets.oneMonth'), value: dayjs().add(1, 'month') },
                            { label: t('radius:users.form.expirePresets.threeMonths'), value: dayjs().add(3, 'month') },
                            { label: t('radius:users.form.expirePresets.sixMonths'), value: dayjs().add(6, 'month') },
                            { label: t('radius:users.form.expirePresets.oneYear'), value: dayjs().add(1, 'year') },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item name="upRate" label={t('radius:users.form.upRate')}>
                        <InputNumber min={0} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item name="downRate" label={t('radius:users.form.downRate')}>
                        <InputNumber min={0} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item
                        name="activeNum"
                        label={t('radius:users.form.activeNum')}
                        tooltip={t('radius:users.form.activeNumTip')}
                      >
                        <InputNumber min={0} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'contact',
                label: t('radius:users.sections.contact'),
                children: (
                  <Row gutter={16}>
                    <Col xs={24} sm={12}>
                      <Form.Item name="email" label={t('radius:users.form.email')}>
                        <Input maxLength={255} autoComplete="email" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={12}>
                      <Form.Item name="mobile" label={t('radius:users.form.mobile')}>
                        <Input maxLength={32} autoComplete="tel" />
                      </Form.Item>
                    </Col>
                    <Col xs={24}>
                      <Form.Item name="address" label={t('radius:users.form.address')}>
                        <Input maxLength={255} />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'network',
                label: t('radius:users.sections.network'),
                children: (
                  <Row gutter={16}>
                    <Col xs={24} sm={8}>
                      <Form.Item name="ipAddr" label={t('radius:users.form.ipAddr')}>
                        <Input maxLength={64} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item name="ipV6Addr" label={t('radius:users.form.ipV6Addr')}>
                        <Input maxLength={64} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item name="addrPool" label={t('radius:users.form.addrPool')}>
                        <Input maxLength={64} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item
                        name="ipv6PrefixPool"
                        label={t('radius:users.form.ipv6PrefixPool')}
                        tooltip={t('radius:users.form.ipv6PrefixPoolTip')}
                      >
                        <Input maxLength={64} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item
                        name="delegatedIpv6Prefix"
                        label={t('radius:users.form.delegatedIpv6Prefix')}
                      >
                        <Input maxLength={64} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item
                        name="delegatedIpv6PrefixPool"
                        label={t('radius:users.form.delegatedIpv6PrefixPool')}
                        tooltip={t('radius:users.form.delegatedIpv6PrefixPoolTip')}
                      >
                        <Input maxLength={64} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item name="domain" label={t('radius:users.form.domain')}>
                        <Input maxLength={64} />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'binding',
                label: t('radius:users.sections.binding'),
                children: (
                  <Row gutter={16}>
                    <Col xs={24} sm={8}>
                      <Form.Item name="bindMac" label={t('radius:users.form.bindMac')}>
                        <Select
                          options={[
                            { value: true, label: t('radius:common.enabled') },
                            { value: false, label: t('radius:common.disabled') },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Form.Item name="bindVlan" label={t('radius:users.form.bindVlan')}>
                        <Select
                          options={[
                            { value: true, label: t('radius:common.enabled') },
                            { value: false, label: t('radius:common.disabled') },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24}>
                      <Form.Item
                        name="macAddr"
                        label={t('radius:users.form.macAddr')}
                        tooltip={t('radius:users.form.macAddrTip')}
                        rules={[
                          {
                            validator: (_, value?: string) => {
                              const invalid = splitMacInput(value).some((mac) => !MAC_PATTERN.test(mac))
                              return invalid
                                ? Promise.reject(new Error(t('radius:users.form.macAddrInvalid')))
                                : Promise.resolve()
                            },
                          },
                        ]}
                      >
                        <Input.TextArea
                          autoSize={{ minRows: 2, maxRows: 5 }}
                          maxLength={2048}
                          disabled={!bindMacOn}
                          placeholder={t('radius:users.form.macAddrPlaceholder')}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={12}>
                      <Form.Item name="vlanid1" label={t('radius:users.form.vlanid1')}>
                        <InputNumber min={0} max={4094} style={{ width: '100%' }} disabled={!bindVlanOn} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} sm={12}>
                      <Form.Item name="vlanid2" label={t('radius:users.form.vlanid2')}>
                        <InputNumber min={0} max={4094} style={{ width: '100%' }} disabled={!bindVlanOn} />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
              {
                key: 'remark',
                label: t('radius:users.sections.remark'),
                children: (
                  <Row gutter={16}>
                    <Col xs={24}>
                      <Form.Item name="remark" label={t('radius:users.form.remark')}>
                        <Input.TextArea rows={2} maxLength={255} />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
            ]}
          />
        </Form>
      </Modal>
    </div>
  )
}
