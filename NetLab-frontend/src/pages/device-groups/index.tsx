import { useMemo, useState } from 'react'
import { useOperationsStore } from '@/stores/operationsStore'
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
  ClusterOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import type { DeviceGroup, DeviceStatus, OperationsFilter } from '@/types/operations'
import { DEVICE_STATUS_CONFIG } from '@/types/operations'
import '@/assets/css/device-groups.css'

const { Text } = Typography
const { RangePicker } = DatePicker

export default function DeviceGroupsPage() {
  const { t } = useTranslation(['common', 'operations'])
  const navigate = useNavigate()
  const { token } = theme.useToken()
  const { message } = App.useApp()

  const groups = useOperationsStore((s) => s.deviceGroups)
  const [filter, setFilter] = useState<OperationsFilter>({ status: null, search: '' })
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  const filteredGroups = useMemo(() => {
    return groups
      .filter((group) => {
        if (filter.status && group.status !== filter.status) return false
        if (filter.search && !group.name.toLowerCase().includes(filter.search.toLowerCase())) return false
        return true
      })
      .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
  }, [groups, filter])

  const columns: ColumnsType<DeviceGroup> = [
    {
      title: t('operations:groupName'),
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
      sorter: (a, b) => a.name.localeCompare(b.name),
      render: (name: string, record) => (
        <Button type="link" style={{ padding: 0 }} onClick={() => navigate(`/device-groups/${record.id}`)}>
          {name}
        </Button>
      ),
    },
    {
      title: t('operations:status'),
      dataIndex: 'status',
      key: 'status',
      width: 110,
      filters: Object.entries(DEVICE_STATUS_CONFIG).map(([value, cfg]) => ({
        text: t(cfg.labelKey),
        value,
      })),
      onFilter: (value, record) => record.status === value,
      sorter: (a, b) => a.status.localeCompare(b.status),
      render: (status: DeviceStatus) => {
        const cfg = DEVICE_STATUS_CONFIG[status]
        return <Tag color={cfg.color}>{t(cfg.labelKey)}</Tag>
      },
    },
    {
      title: t('operations:deviceCount'),
      dataIndex: 'deviceCount',
      key: 'deviceCount',
      width: 100,
      align: 'right',
      sorter: (a, b) => a.deviceCount - b.deviceCount,
    },
    {
      title: t('operations:onlineCount'),
      dataIndex: 'onlineCount',
      key: 'onlineCount',
      width: 100,
      align: 'right',
      sorter: (a, b) => a.onlineCount - b.onlineCount,
    },
    {
      title: t('operations:alertCount'),
      dataIndex: 'alertCount',
      key: 'alertCount',
      width: 100,
      align: 'right',
      sorter: (a, b) => a.alertCount - b.alertCount,
      render: (value: number) => <Text type={value > 0 ? 'danger' : 'secondary'}>{value}</Text>,
    },
    {
      title: t('operations:cpuUsage'),
      dataIndex: 'cpuUsage',
      key: 'cpuUsage',
      width: 140,
      sorter: (a, b) => a.cpuUsage - b.cpuUsage,
      render: (value: number) => (
        <Progress percent={value} size="small" status={value > 80 ? 'exception' : 'active'} style={{ minWidth: 100 }} />
      ),
    },
    {
      title: t('operations:memUsage'),
      dataIndex: 'memUsage',
      key: 'memUsage',
      width: 140,
      sorter: (a, b) => a.memUsage - b.memUsage,
      render: (value: number) => (
        <Progress percent={value} size="small" strokeColor={value > 80 ? token.colorError : token.colorSuccess} style={{ minWidth: 100 }} />
      ),
    },
    {
      title: t('operations:updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      responsive: ['lg'],
      defaultSortOrder: 'descend',
      sorter: (a, b) => new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime(),
      render: (value: string) => <Text type="secondary">{new Date(value).toLocaleDateString()}</Text>,
    },
    {
      title: t('operations:actions'),
      key: 'actions',
      width: 180,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small" wrap>
          <Button type="link" size="small" icon={<ClusterOutlined />} onClick={() => navigate(`/device-groups/${record.id}`)}>
            {t('operations:openGroup')}
          </Button>
          <Popconfirm
            title={t('operations:confirmDeleteGroup')}
            onConfirm={() => message.success(t('operations:deleteGroupSuccess'))}
            okButtonProps={{ danger: true }}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ width: '100%' }}>
      {selectedRowKeys.length > 0 && (
        <Alert
          className="netlab-dashboard-batch-bar"
          type="info"
          showIcon
          title={t('operations:selectedCount', { count: selectedRowKeys.length })}
          action={
            <Space>
              <Button size="small" icon={<PlayCircleOutlined />}>
                {t('operations:batchEnableMonitoring')}
              </Button>
              <Button size="small" icon={<PauseCircleOutlined />}>
                {t('operations:batchDisableMonitoring')}
              </Button>
              <Popconfirm
                title={t('operations:confirmDeleteGroup')}
                onConfirm={() => {
                  message.success(t('operations:deleteGroupSuccess'))
                  setSelectedRowKeys([])
                }}
                okButtonProps={{ danger: true }}
              >
                <Button size="small" danger icon={<DeleteOutlined />}>
                  {t('operations:batchDelete')}
                </Button>
              </Popconfirm>
            </Space>
          }
        />
      )}

      <div className="netlab-dashboard-filters">
        <Input.Search
          placeholder={t('operations:searchGroupPlaceholder')}
          allowClear
          onSearch={(value) => setFilter((current) => ({ ...current, search: value }))}
          onChange={(event) => {
            if (!event.target.value) setFilter((current) => ({ ...current, search: '' }))
          }}
        />
        <Select
          placeholder={t('operations:filterByStatus')}
          allowClear
          style={{ width: 140 }}
          onChange={(value) => setFilter((current) => ({ ...current, status: value || null }))}
          options={[
            { label: t('operations:allStatus'), value: '' },
            ...Object.entries(DEVICE_STATUS_CONFIG).map(([value, cfg]) => ({
              label: t(cfg.labelKey),
              value,
            })),
          ]}
        />
        <RangePicker placeholder={[t('operations:dateRangeStart'), t('operations:dateRangeEnd')]} />
        <Space wrap>
          <Button icon={<ReloadOutlined />}>{t('common:refresh')}</Button>
          <Button type="primary" icon={<PlusOutlined />}>
            {t('operations:createGroup')}
          </Button>
        </Space>
      </div>

      <div className="netlab-dashboard-table">
        <Table<DeviceGroup>
          rowKey="id"
          columns={columns}
          dataSource={filteredGroups}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys: React.Key[]) => setSelectedRowKeys(keys as string[]),
          }}
          pagination={{
            placement: ['bottomCenter'],
            defaultPageSize: 20,
            showSizeChanger: true,
            showTotal: (total) => t('operations:totalGroups', { total }),
          }}
          scroll={{ x: 1200 }}
          size="middle"
          locale={{
            emptyText: (
              <div className="netlab-dashboard-empty">
                <Empty description={t('operations:noGroups')}>
                  <Button type="primary" icon={<PlusOutlined />}>
                    {t('operations:createFirstGroup')}
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
