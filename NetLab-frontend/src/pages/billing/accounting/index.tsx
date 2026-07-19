import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Card,
  DatePicker,
  Descriptions,
  Drawer,
  Input,
  Result,
  Space,
  Table,
  Tag,
  Typography,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { EyeOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import dayjs, { type Dayjs } from 'dayjs'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import type { RadiusAccountingItem } from '@/types/radius'
import { formatBytes, formatDuration } from '../format'

const { Text } = Typography
const { RangePicker } = DatePicker

/** RangePicker 的值：起止两个 dayjs，未选为 null。 */
type RangeValue = [Dayjs | null, Dayjs | null] | null

// 终止原因着色：正常终止绿色，异常/强制终止红色，其余黄色
const TERMINATE_GREEN = ['User-Request', 'Session-Timeout', 'Idle-Timeout']
const TERMINATE_RED = ['Admin-Reset', 'Lost-Carrier', 'Port-Error', 'NAS-Error']

function terminateCauseColor(cause: string): string {
  if (TERMINATE_GREEN.includes(cause)) return 'success'
  if (TERMINATE_RED.includes(cause)) return 'error'
  return 'warning'
}

/** 记账记录页：按用户名 + 时间范围筛选的 RADIUS 记账分页列表。 */
export default function AccountingPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { token } = theme.useToken()
  const { can } = usePermission()
  const canReadRadius = can('radius.read')

  const [data, setData] = useState<RadiusAccountingItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [loading, setLoading] = useState(false)
  // 输入中的筛选值（点搜索后才生效）
  const [usernameInput, setUsernameInput] = useState('')
  const [range, setRange] = useState<RangeValue>(null)
  // 已生效的筛选条件
  const [username, setUsername] = useState('')
  const [startTime, setStartTime] = useState('')
  const [endTime, setEndTime] = useState('')
  // 详情抽屉
  const [detail, setDetail] = useState<RadiusAccountingItem | null>(null)

  const load = useCallback(async () => {
    if (!canReadRadius) return
    setLoading(true)
    try {
      const res = await radiusApi.listAccounting({
        page,
        size,
        username: username || undefined,
        startTime: startTime || undefined,
        endTime: endTime || undefined,
      })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadRadius, page, size, username, startTime, endTime])

  useEffect(() => {
    load()
  }, [load])

  // 可截断列：Typography ellipsis 内置测量，文本溢出被截断时悬停显示完整内容
  const renderEllipsis = (val: string) =>
    val ? (
      <Text ellipsis={{ tooltip: val }} style={{ display: 'block' }}>
        {val}
      </Text>
    ) : (
      '-'
    )

  const renderTime = (val?: string | null) =>
    val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'

  const columns: ColumnsType<RadiusAccountingItem> = [
    {
      title: t('radius:accounting.columns.username'),
      dataIndex: 'username',
      key: 'username',
      width: 140,
      render: renderEllipsis,
    },
    {
      title: t('radius:accounting.columns.nasAddr'),
      dataIndex: 'nasAddr',
      key: 'nasAddr',
      width: 130,
      render: renderEllipsis,
    },
    {
      title: t('radius:accounting.columns.framedIp'),
      dataIndex: 'framedIpaddr',
      key: 'framedIpaddr',
      width: 130,
      render: renderEllipsis,
    },
    {
      title: t('radius:accounting.columns.macAddr'),
      dataIndex: 'macAddr',
      key: 'macAddr',
      width: 140,
      render: renderEllipsis,
    },
    {
      title: t('radius:accounting.columns.startTime'),
      dataIndex: 'acctStartTime',
      key: 'acctStartTime',
      width: 170,
      render: renderTime,
    },
    {
      title: t('radius:accounting.columns.stopTime'),
      dataIndex: 'acctStopTime',
      key: 'acctStopTime',
      width: 170,
      render: renderTime,
    },
    {
      title: t('radius:accounting.columns.sessionTime'),
      dataIndex: 'acctSessionTime',
      key: 'acctSessionTime',
      width: 110,
      render: (val: number) => formatDuration(val),
    },
    {
      title: t('radius:accounting.columns.upload'),
      dataIndex: 'acctInputTotal',
      key: 'acctInputTotal',
      width: 110,
      render: (val: number) => formatBytes(val),
    },
    {
      title: t('radius:accounting.columns.download'),
      dataIndex: 'acctOutputTotal',
      key: 'acctOutputTotal',
      width: 110,
      render: (val: number) => formatBytes(val),
    },
    {
      title: t('radius:accounting.columns.sessionId'),
      dataIndex: 'acctSessionId',
      key: 'acctSessionId',
      width: 180,
      render: renderEllipsis,
    },
    {
      title: t('radius:common.actions'),
      key: 'actions',
      width: 90,
      fixed: 'right',
      render: (_, record) => (
        <Button
          type="text"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => setDetail(record)}
        >
          {t('radius:accounting.detail')}
        </Button>
      ),
    },
  ]

  const handleSearch = () => {
    setPage(1)
    setUsername(usernameInput.trim())
    // RangePicker 值转 RFC3339；showTime 下结束时间取用户所选精确时刻即可
    setStartTime(range?.[0] ? range[0].toISOString() : '')
    setEndTime(range?.[1] ? range[1].toISOString() : '')
  }

  const handleReset = () => {
    setPage(1)
    setUsernameInput('')
    setRange(null)
    setUsername('')
    setStartTime('')
    setEndTime('')
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
            <Input
              value={usernameInput}
              onChange={(e) => setUsernameInput(e.target.value)}
              onPressEnter={handleSearch}
              placeholder={t('radius:accounting.searchUsername')}
              allowClear
              style={{ width: 200 }}
            />
            <RangePicker
              showTime
              value={range}
              onChange={(val) => setRange(val)}
              placeholder={[t('radius:accounting.timeRange'), t('radius:accounting.timeRange')]}
            />
          </Space>
          <Space wrap>
            <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
              {t('radius:common.search')}
            </Button>
            <Button onClick={handleReset}>{t('radius:common.reset')}</Button>
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
          // 列宽合计 1480：容器更宽时按比例分配，更窄时横向滚动；空数据不启用横向滚动
          scroll={data.length > 0 ? { x: 1480 } : undefined}
          tableLayout="fixed"
        />
      </Card>

      {/* 记账详情：全字段分区展示 */}
      <Drawer
        title={t('radius:accounting.detailTitle', { username: detail?.username ?? '' })}
        open={!!detail}
        onClose={() => setDetail(null)}
        width={520}
      >
        {detail && (
          <Space orientation="vertical" size={token.margin} style={{ width: '100%' }}>
            <Descriptions
              title={t('radius:accounting.sections.device')}
              column={1}
              bordered
              size="small"
            >
              <Descriptions.Item label={t('radius:accounting.fields.framedIpaddr')}>
                {detail.framedIpaddr || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.framedNetmask')}>
                {detail.framedNetmask || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.macAddr')}>
                {detail.macAddr || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.framedIpv6Address')}>
                {detail.framedIpv6Address || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.framedIpv6Prefix')}>
                {detail.framedIpv6Prefix || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.delegatedIpv6Prefix')}>
                {detail.delegatedIpv6Prefix || '-'}
              </Descriptions.Item>
            </Descriptions>

            <Descriptions
              title={t('radius:accounting.sections.nas')}
              column={1}
              bordered
              size="small"
            >
              <Descriptions.Item label={t('radius:accounting.fields.nasAddr')}>
                {detail.nasAddr || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.nasId')}>
                {detail.nasId || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.nasPort')}>
                {detail.nasPort ?? '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.nasPortId')}>
                {detail.nasPortId || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.nasPortType')}>
                {detail.nasPortType ?? '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.serviceType')}>
                {detail.serviceType ?? '-'}
              </Descriptions.Item>
            </Descriptions>

            <Descriptions
              title={t('radius:accounting.sections.time')}
              column={1}
              bordered
              size="small"
            >
              <Descriptions.Item label={t('radius:accounting.fields.acctStartTime')}>
                {renderTime(detail.acctStartTime)}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.acctStopTime')}>
                {renderTime(detail.acctStopTime)}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.acctSessionTime')}>
                {formatDuration(detail.acctSessionTime)}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.sessionTimeout')}>
                {detail.sessionTimeout ? formatDuration(detail.sessionTimeout) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.lastUpdate')}>
                {renderTime(detail.lastUpdate)}
              </Descriptions.Item>
            </Descriptions>

            <Descriptions
              title={t('radius:accounting.sections.traffic')}
              column={1}
              bordered
              size="small"
            >
              <Descriptions.Item label={t('radius:accounting.fields.acctInputTotal')}>
                {formatBytes(detail.acctInputTotal)}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.acctOutputTotal')}>
                {formatBytes(detail.acctOutputTotal)}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.totalTraffic')}>
                {formatBytes((detail.acctInputTotal || 0) + (detail.acctOutputTotal || 0))}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.acctInputPackets')}>
                {detail.acctInputPackets ?? '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.acctOutputPackets')}>
                {detail.acctOutputPackets ?? '-'}
              </Descriptions.Item>
            </Descriptions>

            <Descriptions
              title={t('radius:accounting.sections.other')}
              column={1}
              bordered
              size="small"
            >
              <Descriptions.Item label={t('radius:accounting.fields.acctSessionId')}>
                {detail.acctSessionId || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.nasClass')}>
                {detail.nasClass || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('radius:accounting.fields.acctTerminateCause')}>
                {detail.acctTerminateCause ? (
                  <Tag color={terminateCauseColor(detail.acctTerminateCause)}>
                    {detail.acctTerminateCause}
                  </Tag>
                ) : (
                  '-'
                )}
              </Descriptions.Item>
            </Descriptions>
          </Space>
        )}
      </Drawer>
    </div>
  )
}
