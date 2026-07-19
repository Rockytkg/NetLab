import { useCallback, useEffect, useState, type Key } from 'react'
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
import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import type { RadiusAuthLogItem } from '@/types/radius'

const { Text } = Typography

const RESULT_VALUES = ['accept', 'reject']

const RESULT_TAG_COLORS: Record<string, string> = {
  accept: 'green',
  reject: 'red',
}

/** RADIUS 认证日志页：分页列表 + 结果筛选 + 关键词搜索 + 批量删除。 */
export default function RadiusAuthLogsPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { token } = theme.useToken()
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadLogs = can('radius.read')

  const [data, setData] = useState<RadiusAuthLogItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [resultFilter, setResultFilter] = useState('')
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])

  const load = useCallback(async () => {
    if (!canReadLogs) return
    setLoading(true)
    try {
      const res = await radiusApi.listAuthLogs({
        page,
        size,
        keyword,
        result: resultFilter || undefined,
      })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadLogs, page, size, keyword, resultFilter])

  useEffect(() => {
    load()
  }, [load])

  // 可截断列：仅当文本真正溢出被截断时悬停才显示完整内容提示
  const renderEllipsis = (val?: string | null) =>
    val ? (
      <Text ellipsis={{ tooltip: val }} style={{ display: 'block' }}>
        {val}
      </Text>
    ) : (
      '-'
    )

  const columns: ColumnsType<RadiusAuthLogItem> = [
    {
      // 170：「YYYY-MM-DD HH:mm:ss」（19 字符）完整显示
      title: t('radius:authLogs.columns.time'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (val?: string | null) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
    {
      // 120：覆盖常见用户名，超长截断 + tip
      title: t('radius:authLogs.columns.username'),
      dataIndex: 'username',
      key: 'username',
      width: 120,
      render: renderEllipsis,
    },
    {
      // 100：「MSCHAPv2」等认证方式 Tag 完整显示；bypass 显示为本地化标签
      title: t('radius:authLogs.columns.authType'),
      dataIndex: 'authType',
      key: 'authType',
      width: 100,
      render: (val: string) =>
        val ? <Tag>{val === 'bypass' ? t('radius:authLogs.authTypeBypass') : val}</Tag> : '-',
    },
    {
      // 130：IPv4「255.255.255.255」完整显示
      title: t('radius:authLogs.columns.nasAddr'),
      dataIndex: 'nasAddr',
      key: 'nasAddr',
      width: 130,
      render: renderEllipsis,
    },
    {
      // 140：MAC「AA:BB:CC:DD:EE:FF」（17 字符）完整显示
      title: t('radius:authLogs.columns.macAddr'),
      dataIndex: 'macAddr',
      key: 'macAddr',
      width: 140,
      render: renderEllipsis,
    },
    {
      // 90：最长 Tag「拒绝/Rejected」完整显示
      title: t('radius:authLogs.columns.result'),
      dataIndex: 'result',
      key: 'result',
      width: 90,
      render: (val: string) =>
        val ? (
          <Tag color={RESULT_TAG_COLORS[val] ?? 'default'}>
            {t(`radius:authLogs.${val}`, val)}
          </Tag>
        ) : (
          '-'
        ),
    },
    {
      // 240：失败原因多为长文本，截断 + tip
      title: t('radius:authLogs.columns.reason'),
      dataIndex: 'reason',
      key: 'reason',
      width: 240,
      render: renderEllipsis,
    },
  ]

  const handleSearch = () => {
    setPage(1)
    setKeyword(search.trim())
  }

  const handleDelete = () => {
    if (selectedRowKeys.length === 0) return
    modal.confirm({
      title: t('radius:common.confirmTitle'),
      content: t('radius:authLogs.deleteConfirm', { count: selectedRowKeys.length }),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        await radiusApi.deleteAuthLogs(selectedRowKeys.map(Number))
        message.success(t('radius:common.deleteSuccess'))
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
            <Can permission="radius.manage">
              <Button
                danger
                icon={<DeleteOutlined />}
                disabled={selectedRowKeys.length === 0}
                onClick={handleDelete}
              >
                {t('radius:authLogs.deleteSelected')}
              </Button>
            </Can>
          </Space>
          <Space wrap>
            <Select
              value={resultFilter}
              onChange={(val) => {
                setPage(1)
                setResultFilter(val)
              }}
              style={{ width: 140 }}
              options={[
                { value: '', label: t('radius:authLogs.resultAll') },
                ...RESULT_VALUES.map((value) => ({
                  value,
                  label: t(`radius:authLogs.${value}`, value),
                })),
              ]}
            />
            <Input.Search
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onSearch={handleSearch}
              placeholder={t('radius:authLogs.searchPlaceholder')}
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
          // 列宽合计 990 + 选择列 ≈ 1050：容器更宽时按比例分配，更窄时横向滚动；空数据不启用横向滚动
          scroll={data.length > 0 ? { x: 1050 } : undefined}
          tableLayout="fixed"
        />
      </Card>
    </div>
  )
}
