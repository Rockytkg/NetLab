import { useCallback, useEffect, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  Modal,
  Result,
  Select,
  Space,
  Table,
  Tabs,
  Typography,
  Upload,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import Toolbar from '@/pages/billing/components/Toolbar'
import { billingDetailRow, renderTime } from '@/pages/billing/shared'
import BillingDetailModal from '@/pages/billing/components/BillingDetailModal'
import type { RadiusCertItem, RadiusCertPayload, RadiusCertType } from '@/types/radius'

const { Text } = Typography

/** 证书表单值；编辑时 certPem/keyPem 留空表示不替换现有材料。 */
interface CertFormValues {
  name: string
  certType: RadiusCertType
  certPem?: string
  keyPem?: string
  remark?: string
}

/** 读取文件内容为文本。 */
function readFileAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = () => reject(new Error('Failed to read file'))
    reader.readAsText(file)
  })
}

/** PEM 文本域 + 上传按钮组合；与 antd Form 集成。 */
function PemField({
  value,
  onChange,
  label,
  placeholder,
}: {
  value?: string
  onChange?: (val: string) => void
  label: string
  placeholder?: string
}) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <Input.TextArea
        rows={3}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        placeholder={placeholder}
      />
      <Upload
        showUploadList={false}
        customRequest={async (options) => {
          try {
            const file = options.file as File
            const content = await readFileAsText(file)
            onChange?.(content)
          } catch {
            // 读取失败静默忽略
          }
        }}
      >
        <Button type="dashed" size="small" icon={<UploadOutlined />} block>
          {label}
        </Button>
      </Upload>
    </div>
  )
}

/** 证书管理页：分页列表 + 类型筛选 + 新增/编辑/导出/删除。 */
export default function RadiusCertsPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadRadius = can('radius.read')

  const [data, setData] = useState<RadiusCertItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [certTypeFilter, setCertTypeFilter] = useState('')
  const [loading, setLoading] = useState(false)

  // 新增/编辑弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editingCert, setEditingCert] = useState<RadiusCertItem | null>(null)
  const [detail, setDetail] = useState<RadiusCertItem | null>(null)
  const [form] = Form.useForm<CertFormValues>()
  const [saving, setSaving] = useState(false)
  const [activeFormSection, setActiveFormSection] = useState('general')
  const certTypeWatch = Form.useWatch('certType', form)

  // 导出确认弹窗
  const [exportTarget, setExportTarget] = useState<RadiusCertItem | null>(null)
  const [exportIncludeKey, setExportIncludeKey] = useState(false)
  const [exporting, setExporting] = useState(false)

  const load = useCallback(async () => {
    if (!canReadRadius) return
    setLoading(true)
    try {
      const res = await radiusApi.listCerts({
        page,
        size,
        keyword: keyword || undefined,
        certType: certTypeFilter || undefined,
      })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadRadius, page, size, keyword, certTypeFilter])

  useEffect(() => {
    load()
  }, [load])

  const certTypeTag = (certType: string) =>
    certType === 'server' ? (
      <Text code style={{ color: 'var(--ant-blue-6)' }}>{t('radius:certs.typeServer')}</Text>
    ) : (
      <Text code style={{ color: 'var(--ant-purple-6)' }}>{t('radius:certs.typeCa')}</Text>
    )

  const columns: ColumnsType<RadiusCertItem> = [
    {
      title: t('radius:certs.columns.name'),
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (val: string, record) => (
        <span style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
          {certTypeTag(record.certType)}
          <Text ellipsis style={{ flex: 1 }}>{val}</Text>
        </span>
      ),
    },
    {
      title: t('radius:certs.columns.subject'),
      dataIndex: 'subject',
      key: 'subject',
      width: 200,
      ellipsis: { showTitle: true },
    },
    {
      title: t('radius:certs.columns.notAfter'),
      dataIndex: 'notAfter',
      key: 'notAfter',
      width: 160,
      render: (val: string) => {
        if (!val) return '-'
        const expired = dayjs(val).isBefore(dayjs())
        return (
          <Text type={expired ? 'danger' : undefined}>
            {dayjs(val).format('YYYY-MM-DD HH:mm:ss')}
          </Text>
        )
      },
    },
    {
      title: t('radius:certs.columns.hasKey'),
      dataIndex: 'hasKey',
      key: 'hasKey',
      width: 100,
      responsive: ['sm'],
      render: (val: boolean) =>
        val ? (
          <Text type="success">{t('radius:certs.hasKeyYes')}</Text>
        ) : (
          <Text type="secondary">{t('radius:certs.hasKeyNo')}</Text>
        ),
    },
    {
      title: t('radius:common.remark'),
      dataIndex: 'remark',
      key: 'remark',
      width: 140,
      ellipsis: { showTitle: true },
      responsive: ['md'],
    },
    {
      title: t('radius:common.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      responsive: ['lg'],
      render: renderTime,
    },
    {
      title: t('radius:common.actions'),
      key: 'actions',
      width: 200,
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
              icon={<DownloadOutlined />}
              onClick={() => handleExport(record)}
            >
              {t('radius:certs.export')}
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
    setEditingCert(null)
    form.resetFields()
    form.setFieldsValue({ certType: 'server' })
    setActiveFormSection('general')
    setModalOpen(true)
  }

  const openEdit = (record: RadiusCertItem) => {
    setEditingCert(record)
    form.resetFields()
    form.setFieldsValue({
      name: record.name,
      certType: record.certType,
      certPem: undefined,
      keyPem: undefined,
      remark: record.remark,
    })
    setActiveFormSection('general')
    setModalOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload: RadiusCertPayload = {
        name: values.name.trim(),
        remark: values.remark?.trim() ?? '',
      }
      const certPem = (values.certPem || '').trim()
      const keyPem = (values.keyPem || '').trim()
      if (editingCert) {
        if (certPem) payload.certPem = certPem
        if (keyPem) payload.keyPem = keyPem
        await radiusApi.updateCert(editingCert.id, payload)
      } else {
        payload.certType = values.certType
        payload.certPem = certPem
        if (keyPem) payload.keyPem = keyPem
        await radiusApi.createCert(payload)
      }
      message.success(t('radius:common.saveSuccess'))
      setModalOpen(false)
      setEditingCert(null)
      await load()
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (record: RadiusCertItem) => {
    modal.confirm({
      title: t('radius:common.confirmTitle'),
      content: t('radius:certs.deleteConfirm'),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        await radiusApi.deleteCert(record.id)
        message.success(t('radius:common.deleteSuccess'))
        await load()
      },
    })
  }

  const handleExport = (record: RadiusCertItem) => {
    if (record.certType === 'server' && record.hasKey) {
      setExportIncludeKey(false)
      setExportTarget(record)
      return
    }
    void doExport(record, false)
  }

  const doExport = async (record: RadiusCertItem, includeKey: boolean) => {
    setExporting(true)
    try {
      await radiusApi.exportCert(record.id, record.name, includeKey)
      message.success(t('radius:certs.exportSuccess'))
      setExportTarget(null)
    } catch {
      // 拦截器已提示错误
    } finally {
      setExporting(false)
    }
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
                {t('radius:certs.create')}
              </Button>
            </Can>
          }
          right={
            <>
              <Select
                value={certTypeFilter}
                onChange={(val) => {
                  setPage(1)
                  setCertTypeFilter(val)
                }}
                className="netlab-billing-toolbar-select"
                options={[
                  { value: '', label: t('radius:certs.typeAll') },
                  { value: 'server', label: t('radius:certs.typeServer') },
                  { value: 'ca', label: t('radius:certs.typeCa') },
                ]}
              />
              <Input.Search
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onSearch={handleSearch}
                placeholder={t('radius:certs.searchPlaceholder')}
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
          onRow={billingDetailRow(setDetail)}
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
          scroll={{ x: 1140 }}
          tableLayout="fixed"
        />
      </Card>

      <BillingDetailModal title={detail?.name ?? ''} open={!!detail} onClose={() => setDetail(null)} items={detail ? [
        { label: t('radius:certs.columns.name'), value: detail.name },
        { label: t('radius:certs.form.certType'), value: certTypeTag(detail.certType) },
        { label: t('radius:certs.metadata.subject'), value: detail.subject },
        { label: t('radius:certs.metadata.issuer'), value: detail.issuer },
        { label: t('radius:certs.metadata.serial'), value: detail.serial },
        { label: t('radius:certs.metadata.fingerprint'), value: detail.fingerprint },
        { label: t('radius:certs.metadata.notBefore'), value: renderTime(detail.notBefore) },
        { label: t('radius:certs.metadata.notAfter'), value: renderTime(detail.notAfter) },
        { label: t('radius:certs.columns.hasKey'), value: detail.hasKey ? t('radius:certs.hasKeyYes') : t('radius:certs.hasKeyNo') },
        { label: t('radius:common.remark'), value: detail.remark },
        { label: t('radius:common.createdAt'), value: renderTime(detail.createdAt) },
        { label: t('radius:common.updatedAt'), value: renderTime(detail.updatedAt) },
      ] : []} />

      {/* 新增/编辑证书 */}
      <Modal
        title={editingCert ? t('radius:certs.edit') : t('radius:certs.create')}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditingCert(null)
          form.resetFields()
        }}
        onOk={handleSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={saving}
        forceRender
        width={{ xs: 'calc(100vw - 32px)', sm: 560, md: 640 }}
      >
        <Form form={form} layout="vertical" requiredMark={false}>
          <Tabs
            activeKey={activeFormSection}
            onChange={setActiveFormSection}
            items={[
              {
                key: 'general',
                label: t('radius:certs.sections.general'),
                children: (
                  <>
                    <Form.Item name="name" label={t('radius:certs.form.name')} normalize={(value: string) => value?.trim()} rules={[{ required: true, message: t('radius:certs.form.nameRequired') }]}>
                      <Input maxLength={128} />
                    </Form.Item>
                    {editingCert ? (
                      <Form.Item label={t('radius:certs.form.certType')}>{certTypeTag(editingCert.certType)}</Form.Item>
                    ) : (
                      <Form.Item name="certType" label={t('radius:certs.form.certType')}>
                        <Select options={[{ value: 'server', label: t('radius:certs.typeServer') }, { value: 'ca', label: t('radius:certs.typeCa') }]} />
                      </Form.Item>
                    )}
                    <Form.Item name="remark" label={t('radius:certs.form.remark')}>
                      <Input.TextArea rows={2} maxLength={255} />
                    </Form.Item>
                  </>
                ),
              },
              {
                key: 'material',
                label: t('radius:certs.sections.material'),
                children: (
                  <>
                    <Form.Item name="certPem" label={t('radius:certs.form.certPem')} extra={editingCert ? t('radius:certs.form.replaceTip') : undefined} rules={editingCert ? [] : [{ required: true, message: t('radius:certs.form.certPemRequired') }]}>
                      <PemField label={t('radius:certs.uploadFile')} placeholder="-----BEGIN CERTIFICATE-----" />
                    </Form.Item>
                    <Form.Item name="keyPem" label={t('radius:certs.form.keyPem')} tooltip={t('radius:certs.form.keyPemTip')} extra={editingCert ? t('radius:certs.form.replaceTip') : undefined} rules={!editingCert && certTypeWatch === 'server' ? [{ required: true, message: t('radius:certs.form.keyPemRequired') }] : []}>
                      <PemField label={t('radius:certs.uploadFile')} placeholder="-----BEGIN PRIVATE KEY-----" />
                    </Form.Item>
                  </>
                ),
              },
              ...(editingCert
                ? [{
                    key: 'metadata',
                    label: t('radius:certs.sections.metadata'),
                    children: (
                      <Descriptions column={1} size="small" bordered>
                        <Descriptions.Item label={t('radius:certs.metadata.subject')}>{editingCert.subject || '-'}</Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.issuer')}>{editingCert.issuer || '-'}</Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.serial')}>{editingCert.serial || '-'}</Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.fingerprint')}>{editingCert.fingerprint || '-'}</Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.notBefore')}>{editingCert.notBefore ? dayjs(editingCert.notBefore).format('YYYY-MM-DD HH:mm:ss') : '-'}</Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.notAfter')}>{editingCert.notAfter ? dayjs(editingCert.notAfter).format('YYYY-MM-DD HH:mm:ss') : '-'}</Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.columns.hasKey')}>{editingCert.hasKey ? t('radius:certs.hasKeyYes') : t('radius:certs.hasKeyNo')}</Descriptions.Item>
                      </Descriptions>
                    ),
                  }]
                : []),
            ]}
          />
        </Form>
      </Modal>

      {/* 导出确认：服务器证书可选择是否包含私钥 */}
      <Modal
        title={t('radius:certs.exportTitle')}
        open={!!exportTarget}
        onCancel={() => setExportTarget(null)}
        onOk={() => exportTarget && doExport(exportTarget, exportIncludeKey)}
        okText={t('radius:certs.export')}
        cancelText={t('common:cancel')}
        confirmLoading={exporting}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Select
            value={exportIncludeKey}
            onChange={(val) => setExportIncludeKey(val)}
            style={{ width: '100%' }}
            options={[
              { value: false, label: t('radius:certs.exportWithoutKey') },
              { value: true, label: t('radius:certs.exportWithKey') },
            ]}
          />
          {exportIncludeKey && (
            <Alert type="warning" showIcon title={t('radius:certs.exportKeyWarning')} />
          )}
        </div>
      </Modal>
    </div>
  )
}
