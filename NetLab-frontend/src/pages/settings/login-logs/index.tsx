import { useCallback, useEffect, useMemo, useState, type Key } from 'react'
import {
  App,
  Button,
  Card,
  Input,
  Result,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { DeleteOutlined, DownloadOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { logApi } from '@/services/log'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import type { LoginLogItem } from '@/types/log'

const { Text } = Typography

/** 后端 loginType 值 → i18n key（后端 `2fa` 对应前端 `twoFactor`） */
const LOGIN_TYPE_KEYS: Record<string, string> = {
  password: 'password',
  '2fa': 'twoFactor',
  recovery: 'recovery',
  passkey: 'passkey',
  oauth: 'oauth',
}

const STATUS_TAG_COLORS: Record<string, string> = {
  success: 'green',
  failed: 'red',
  pending: 'gold',
}

const LOGIN_TYPE_VALUES = ['password', '2fa', 'recovery', 'passkey', 'oauth']
const STATUS_VALUES = ['success', 'failed', 'pending']

/** 登录日志页：分页列表 + 状态/登录方式筛选 + 关键词搜索 + 选中导出/删除。 */
export default function LoginLogsPage() {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadLogs = can('log.read')

  const [data, setData] = useState<LoginLogItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])

  const load = useCallback(async () => {
    if (!canReadLogs) return
    setLoading(true)
    try {
      const res = await logApi.listLoginLogs({
        page,
        size,
        keyword,
        status: statusFilter || undefined,
        loginType: typeFilter || undefined,
      })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadLogs, page, size, keyword, statusFilter, typeFilter])

  useEffect(() => {
    load()
  }, [load])

  const selectedLogs = useMemo(() => {
    const selected = new Set(selectedRowKeys)
    return data.filter((item) => selected.has(item.id))
  }, [data, selectedRowKeys])

  // 可截断列（时间/用户名/IP/操作系统/浏览器/指纹/UA）：
  // Typography ellipsis 内置测量，仅当文本真正溢出被截断时悬停才显示完整内容提示
  const renderEllipsis = (val: string) =>
    val ? (
      <Text ellipsis={{ tooltip: val }} style={{ display: 'block' }}>
        {val}
      </Text>
    ) : (
      '-'
    )

  const columns: ColumnsType<LoginLogItem> = [
    {
      // 175：en-US "7/18/2026, 12:23:39 PM"（22 字符）可完整显示
      title: t('settings:loginLogs.columns.time'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 175,
      render: (val: string) => renderEllipsis(new Date(val).toLocaleString()),
    },
    {
      // 100：约 5–8 字符，覆盖常见用户名，超长截断 + tip
      title: t('settings:loginLogs.columns.username'),
      dataIndex: 'username',
      key: 'username',
      width: 100,
      render: renderEllipsis,
    },
    {
      // 100：最长 Tag「二次验证」完整显示
      title: t('settings:loginLogs.columns.type'),
      dataIndex: 'loginType',
      key: 'loginType',
      width: 100,
      render: (val: string) => (
        <Tag>{t(`settings:loginLogs.type.${LOGIN_TYPE_KEYS[val] ?? val}`, val)}</Tag>
      ),
    },
    {
      // 110：最长 Tag「待二次验证」(5 字) + Tag/单元格内边距 ≈ 108
      title: t('settings:loginLogs.columns.status'),
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (val: string) => (
        <Tag color={STATUS_TAG_COLORS[val] ?? 'default'}>
          {t(`settings:loginLogs.status.${val}`, val)}
        </Tag>
      ),
    },
    {
      // 130：IPv4「255.255.255.255」完整显示
      title: t('settings:loginLogs.columns.ip'),
      dataIndex: 'ip',
      key: 'ip',
      width: 130,
      render: renderEllipsis,
    },
    {
      // 100：紧凑列，「Windows 11 (amd64)」类值截断 + tip
      title: t('settings:loginLogs.columns.os'),
      dataIndex: 'os',
      key: 'os',
      width: 100,
      render: renderEllipsis,
    },
    {
      // 90：紧凑列，「Chrome 126」可显，更长值截断 + tip
      title: t('settings:loginLogs.columns.browser'),
      dataIndex: 'browser',
      key: 'browser',
      width: 90,
      render: renderEllipsis,
    },
    {
      // 80：预留归属地，当前恒为 '-'
      title: t('settings:loginLogs.columns.location'),
      dataIndex: 'location',
      key: 'location',
      width: 80,
      render: (val: string) => val || '-',
    },
    {
      // 150：32 位 hex 指纹必然截断 + tip
      title: t('settings:loginLogs.columns.fingerprint'),
      dataIndex: 'fingerprint',
      key: 'fingerprint',
      width: 150,
      render: renderEllipsis,
    },
    {
      // 150：与指纹列一致，长 UA 截断 + tip
      title: t('settings:loginLogs.columns.userAgent'),
      dataIndex: 'userAgent',
      key: 'userAgent',
      width: 150,
      render: renderEllipsis,
    },
  ]

  const handleSearch = () => {
    setPage(1)
    setKeyword(search.trim())
  }

  const handleExport = () => {
    if (selectedLogs.length === 0) return
    logApi.exportLoginLogs(
      selectedLogs,
      [
        t('settings:loginLogs.columns.time'),
        t('settings:loginLogs.columns.username'),
        t('settings:loginLogs.columns.type'),
        t('settings:loginLogs.columns.status'),
        t('settings:loginLogs.columns.ip'),
        t('settings:loginLogs.columns.location'),
        t('settings:loginLogs.columns.os'),
        t('settings:loginLogs.columns.browser'),
        t('settings:loginLogs.columns.fingerprint'),
        t('settings:loginLogs.columns.userAgent'),
      ],
      {
        status: Object.fromEntries(
          STATUS_VALUES.map((value) => [value, t(`settings:loginLogs.status.${value}`, value)]),
        ),
        type: Object.fromEntries(
          LOGIN_TYPE_VALUES.map((value) => [
            value,
            t(`settings:loginLogs.type.${LOGIN_TYPE_KEYS[value]}`, value),
          ]),
        ),
      },
      `login-logs-${new Date().toISOString().slice(0, 10)}.xlsx`,
    )
  }

  const handleDelete = () => {
    if (selectedRowKeys.length === 0) return
    modal.confirm({
      title: t('settings:loginLogs.deleteConfirmTitle'),
      content: t('settings:loginLogs.deleteConfirm', { count: selectedRowKeys.length }),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        const res = await logApi.deleteLoginLogs(selectedRowKeys.map(Number))
        message.success(t('settings:loginLogs.deleted', { count: res.deleted }))
        setSelectedRowKeys([])
        await load()
      },
    })
  }

  if (!canReadLogs) {
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
            <Button
              icon={<DownloadOutlined />}
              disabled={selectedRowKeys.length === 0}
              onClick={handleExport}
            >
              {t('settings:loginLogs.exportSelected')}
            </Button>
            <Can permission="log.delete">
              <Button
                danger
                icon={<DeleteOutlined />}
                disabled={selectedRowKeys.length === 0}
                onClick={handleDelete}
              >
                {t('settings:loginLogs.deleteSelected')}
              </Button>
            </Can>
          </Space>
          <Space wrap>
            <Select
              value={statusFilter}
              onChange={(val) => {
                setPage(1)
                setStatusFilter(val)
              }}
              style={{ width: 140 }}
              options={[
                { value: '', label: t('settings:loginLogs.statusAll') },
                ...STATUS_VALUES.map((value) => ({
                  value,
                  label: t(`settings:loginLogs.status.${value}`, value),
                })),
              ]}
            />
            <Select
              value={typeFilter}
              onChange={(val) => {
                setPage(1)
                setTypeFilter(val)
              }}
              style={{ width: 140 }}
              options={[
                { value: '', label: t('settings:loginLogs.typeAll') },
                ...LOGIN_TYPE_VALUES.map((value) => ({
                  value,
                  label: t(`settings:loginLogs.type.${LOGIN_TYPE_KEYS[value]}`, value),
                })),
              ]}
            />
            <Input.Search
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onSearch={handleSearch}
              placeholder={t('settings:loginLogs.searchPlaceholder')}
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
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
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
          // 列宽合计 1185 + 选择列 ≈ 1250：容器更宽时多余宽度按比例分配给各列，
          // 容器更窄时出现横向滚动条，列不被压缩
          scroll={{ x: 1250 }}
          tableLayout="fixed"
        />
      </Card>
    </div>
  )
}
