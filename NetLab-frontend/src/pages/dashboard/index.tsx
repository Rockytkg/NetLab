import { useMemo, useState } from 'react'
import { useLabStore } from '@/stores/labStore'
import {
  Alert,
  App,
  Button,
  DatePicker,
  Empty,
  Input,
  Popconfirm,
  Progress,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  PlusOutlined,
  ReloadOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  DeleteOutlined,
  ExperimentOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import type { LabItem, LabFilter, LabStatus } from '@/types/lab'
import { LAB_STATUS_CONFIG } from '@/types/lab'

const { Title, Text } = Typography
const { RangePicker } = DatePicker

export default function DashboardPage() {
  const { t } = useTranslation(['common', 'lab', 'menu'])
  const navigate = useNavigate()
  const { token } = theme.useToken()
  const { message } = App.useApp()

  // 实验室列表 —— 来自 labStore（接入 API 后由后端填充）
  const labs = useLabStore((s) => s.labs)
  const [filter, setFilter] = useState<LabFilter>({ status: null, search: '' })
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // 过滤并排序后的数据
  const filteredLabs = useMemo(() => {
    return labs
      .filter((lab) => {
        if (filter.status && lab.status !== filter.status) return false
        if (filter.search && !lab.name.toLowerCase().includes(filter.search.toLowerCase()))
          return false
        return true
      })
      .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
  }, [labs, filter])

  // ── 表格列 ──
  const columns: ColumnsType<LabItem> = [
    {
      title: t('lab:labName'),
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
      sorter: (a, b) => a.name.localeCompare(b.name),
      render: (name: string, record) => (
        <Button
          type="link"
          style={{ padding: 0 }}
          onClick={() => navigate(`/lab/${record.id}`)}
        >
          {name}
        </Button>
      ),
    },
    {
      title: t('lab:status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      filters: Object.entries(LAB_STATUS_CONFIG).map(([value, cfg]) => ({
        text: t(cfg.labelKey),
        value,
      })),
      onFilter: (value, record) => record.status === value,
      sorter: (a, b) => a.status.localeCompare(b.status),
      render: (status: LabStatus) => {
        const cfg = LAB_STATUS_CONFIG[status]
        return <Tag color={cfg.color}>{t(cfg.labelKey)}</Tag>
      },
    },
    {
      title: t('lab:nodeCount'),
      dataIndex: 'nodeCount',
      key: 'nodeCount',
      width: 90,
      align: 'right',
      sorter: (a, b) => a.nodeCount - b.nodeCount,
      render: (n: number) => <Text>{n}</Text>,
    },
    {
      title: t('lab:cpuUsage'),
      dataIndex: 'cpuUsage',
      key: 'cpuUsage',
      width: 140,
      sorter: (a, b) => a.cpuUsage - b.cpuUsage,
      render: (val: number) => (
        <Progress
          percent={val}
          size="small"
          status={val > 80 ? 'exception' : 'active'}
          style={{ minWidth: 100 }}
        />
      ),
    },
    {
      title: t('lab:memUsage'),
      dataIndex: 'memUsage',
      key: 'memUsage',
      width: 140,
      sorter: (a, b) => a.memUsage - b.memUsage,
      render: (val: number) => (
        <Progress
          percent={val}
          size="small"
          strokeColor={val > 80 ? token.colorError : token.colorSuccess}
          style={{ minWidth: 100 }}
        />
      ),
    },
    {
      title: t('lab:createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      responsive: ['lg'],
      sorter: (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
      render: (d: string) => (
        <Text type="secondary">{new Date(d).toLocaleDateString()}</Text>
      ),
    },
    {
      title: t('lab:updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      responsive: ['xl'],
      defaultSortOrder: 'descend',
      sorter: (a, b) => new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime(),
      render: (d: string) => (
        <Text type="secondary">{new Date(d).toLocaleDateString()}</Text>
      ),
    },
    {
      title: t('lab:actions'),
      key: 'actions',
      width: 180,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small" wrap>
          <Button
            type="link"
            size="small"
            icon={<ExperimentOutlined />}
            onClick={() => navigate(`/lab/${record.id}`)}
          >
            {t('lab:openLab')}
          </Button>
          <Popconfirm
            title={t('lab:confirmDelete')}
            onConfirm={() => message.success(t('lab:deleteSuccess'))}
            okButtonProps={{ danger: true }}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  // ── 行选择 ──
  const rowSelection = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys as string[]),
  }

  return (
    <div style={{ width: '100%' }}>
      {/* ── 页面头部 ── */}
      <div className="netlab-page-header">
        <div>
          <Title level={3}>{t('menu:dashboard')}</Title>
          <Text type="secondary">{t('lab:createFirstLab')}</Text>
        </div>
        <Space wrap>
          <Button icon={<ReloadOutlined />}>{t('common:refresh')}</Button>
          <Button type="primary" icon={<PlusOutlined />}>
            {t('lab:createLab')}
          </Button>
        </Space>
      </div>

      {/* ── 批量操作栏 ── */}
      {selectedRowKeys.length > 0 && (
        <Alert
          className="netlab-dashboard-batch-bar"
          type="info"
          showIcon
          message={t('lab:selectedCount', { count: selectedRowKeys.length })}
          action={
            <Space>
              <Button size="small" icon={<PlayCircleOutlined />}>
                {t('lab:batchStart')}
              </Button>
              <Button size="small" icon={<PauseCircleOutlined />}>
                {t('lab:batchStop')}
              </Button>
              <Popconfirm
                title={t('lab:confirmDelete')}
                onConfirm={() => {
                  message.success(t('lab:deleteSuccess'))
                  setSelectedRowKeys([])
                }}
                okButtonProps={{ danger: true }}
              >
                <Button size="small" danger icon={<DeleteOutlined />}>
                  {t('lab:batchDelete')}
                </Button>
              </Popconfirm>
            </Space>
          }
        />
      )}

      {/* ── 过滤栏 ── */}
      <div className="netlab-dashboard-filters">
        <Input.Search
          placeholder={t('lab:searchPlaceholder')}
          allowClear
          onSearch={(v) => setFilter((f) => ({ ...f, search: v }))}
          onChange={(e) => {
            if (!e.target.value) setFilter((f) => ({ ...f, search: '' }))
          }}
        />
        <Select
          placeholder={t('lab:filterByStatus')}
          allowClear
          style={{ width: 140 }}
          onChange={(v) => setFilter((f) => ({ ...f, status: v || null }))}
          options={[
            { label: t('lab:allStatus'), value: '' },
            ...Object.entries(LAB_STATUS_CONFIG).map(([value, cfg]) => ({
              label: t(cfg.labelKey),
              value,
            })),
          ]}
        />
        <RangePicker placeholder={[t('lab:dateRangeStart'), t('lab:dateRangeEnd')]} />
      </div>

      {/* ── 表格 ── */}
      <div className="netlab-dashboard-table">
        <Table<LabItem>
          rowKey="id"
          columns={columns}
          dataSource={filteredLabs}
          rowSelection={rowSelection}
          pagination={{
            placement: ['bottomCenter'],
            defaultPageSize: 20,
            showSizeChanger: true,
            showTotal: (total) => t('lab:totalLabs', { total }),
          }}
          scroll={{ x: 1100 }}
          size="middle"
          locale={{
            emptyText: (
              <div className="netlab-dashboard-empty">
                <Empty description={t('lab:noLabs')}>
                  <Button type="primary" icon={<PlusOutlined />}>
                    {t('lab:createFirstLab')}
                  </Button>
                </Empty>
              </div>
            ),
          }}
        />
      </div>
    </div>
  )
}
