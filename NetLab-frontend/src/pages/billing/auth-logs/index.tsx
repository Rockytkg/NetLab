import { useCallback, useEffect, useState, type Key } from 'react'
import {
  App,
  Button,
  Card,
  Input,
  Result,
  Select,
  Table,
  Typography,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import Toolbar from '@/pages/billing/components/Toolbar'
import { renderTime } from '@/pages/billing/shared'
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

  const columns: ColumnsType<RadiusAuthLogItem> = [
    {
      title: t('radius:authLogs.columns.time'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: renderTime,
    },
    {
      title: t('radius:authLogs.columns.username'),
      dataIndex: 'username',
      key: 'username',
      width: 120,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:authLogs.columns.authType'),
      dataIndex: 'authType',
      key: 'authType',
      width: 110,
      responsive: ['sm'],
      render: (val: string) =>
        val ? <Text>{val === 'bypass' ? t('radius:authLogs.authTypeBypass') : val}</Text> : '-',
    },
    {
      title: t('radius:authLogs.columns.nasAddr'),
      dataIndex: 'nasAddr',
      key: 'nasAddr',
      width: 130,
      ellipsis: { showTitle: true },
      responsive: ['md'],
    },
    {
      title: t('radius:authLogs.columns.macAddr'),
      dataIndex: 'macAddr',
      key: 'macAddr',
      width: 140,
      ellipsis: { showTitle: true },
      responsive: ['md'],
    },
    {
      title: t('radius:authLogs.columns.result'),
      dataIndex: 'result',
      key: 'result',
      width: 100,
      render: (val: string) =>
        val ? (
          <Text style={{ color: RESULT_TAG_COLORS[val] ?? undefined }}>
            {t(`radius:authLogs.${val}`, val)}
          </Text>
        ) : (
          '-'
        ),
    },
    {
      title: t('radius:authLogs.columns.reason'),
      dataIndex: 'reason',
      key: 'reason',
      width: 200,
      ellipsis: { showTitle: true },
      responsive: ['lg'],
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
    <div>
      <Card variant="outlined">
        <Toolbar
          left={
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
          }
          right={
            <>
              <Select
                value={resultFilter}
                onChange={(val) => {
                  setPage(1)
                  setResultFilter(val)
                }}
                className="netlab-billing-toolbar-select"
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
          scroll={{ x: 1000 }}
          tableLayout="fixed"
        />
      </Card>
    </div>
  )
}
