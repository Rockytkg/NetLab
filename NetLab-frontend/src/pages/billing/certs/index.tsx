import { useCallback, useEffect, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Card,
  Collapse,
  Descriptions,
  Divider,
  Form,
  Input,
  Modal,
  Result,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import { radiusApi } from '@/services/radius'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
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

/** 证书管理页：分页列表 + 类型筛选 + 新增/编辑/导出/删除。 */
export default function RadiusCertsPage() {
  const { t } = useTranslation(['radius', 'common', 'settings'])
  const { token } = theme.useToken()
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
  const [form] = Form.useForm<CertFormValues>()
  const [saving, setSaving] = useState(false)
  const certTypeWatch = Form.useWatch('certType', form)

  // 导出确认弹窗（服务器证书选择是否包含私钥）
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

  // 可截断列：仅在文本真正溢出时悬停显示完整内容
  const renderEllipsis = (val?: string | null) =>
    val ? (
      <Text ellipsis={{ tooltip: val }} style={{ display: 'block' }}>
        {val}
      </Text>
    ) : (
      '-'
    )

  const certTypeTag = (certType: string) =>
    certType === 'server' ? (
      <Tag color="blue">{t('radius:certs.typeServer')}</Tag>
    ) : (
      <Tag color="purple">{t('radius:certs.typeCa')}</Tag>
    )

  const columns: ColumnsType<RadiusCertItem> = [
    {
      title: t('radius:certs.columns.name'),
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (val: string, record) => (
        <Space size={token.marginXXS}>
          {renderEllipsis(val)}
          {certTypeTag(record.certType)}
        </Space>
      ),
    },
    {
      title: t('radius:certs.columns.subject'),
      dataIndex: 'subject',
      key: 'subject',
      width: 220,
      render: renderEllipsis,
    },
    {
      // 170：「YYYY-MM-DD HH:mm:ss」完整显示；已过期红色高亮
      title: t('radius:certs.columns.notAfter'),
      dataIndex: 'notAfter',
      key: 'notAfter',
      width: 170,
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
      width: 90,
      render: (val: boolean) =>
        val ? (
          <Tag color="success">{t('radius:certs.hasKeyYes')}</Tag>
        ) : (
          <Tag>{t('radius:certs.hasKeyNo')}</Tag>
        ),
    },
    {
      title: t('radius:common.remark'),
      dataIndex: 'remark',
      key: 'remark',
      width: 160,
      render: renderEllipsis,
    },
    {
      title: t('radius:common.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (val: string) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
    {
      title: t('radius:common.actions'),
      key: 'actions',
      width: 210,
      fixed: 'right',
      render: (_, record) => (
        <Space size={token.marginXXS}>
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
          </Can>
        </Space>
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
      if (editingCert) {
        // 编辑：certType 不可修改；证书材料留空表示不替换
        if (values.certPem?.trim()) payload.certPem = values.certPem.trim()
        if (values.keyPem?.trim()) payload.keyPem = values.keyPem.trim()
        await radiusApi.updateCert(editingCert.id, payload)
      } else {
        payload.certType = values.certType
        payload.certPem = values.certPem?.trim() ?? ''
        if (values.keyPem?.trim()) payload.keyPem = values.keyPem.trim()
        await radiusApi.createCert(payload)
      }
      message.success(t('radius:common.saveSuccess'))
      setModalOpen(false)
      setEditingCert(null)
      await load()
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return // 表单校验失败
      // 其余错误已由拦截器提示
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

  // 服务器证书且持有私钥时，先确认是否包含私钥导出；其余直接导出
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
    <div style={{ width: '100%' }}>
      <Card variant="outlined">
        <Space
          style={{ marginBottom: token.margin, width: '100%', justifyContent: 'space-between' }}
          wrap
        >
          <Space wrap>
            <Can permission="radius.manage">
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                {t('radius:certs.create')}
              </Button>
            </Can>
          </Space>
          <Space wrap>
            <Select
              value={certTypeFilter}
              onChange={(val) => {
                setPage(1)
                setCertTypeFilter(val)
              }}
              style={{ width: 150 }}
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
              style={{ width: 240 }}
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
          // 列宽合计 1350：容器更宽时按比例分配，更窄时横向滚动；空数据不启用横向滚动
          scroll={data.length > 0 ? { x: 1350 } : undefined}
          tableLayout="fixed"
        />
      </Card>

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
        width={640}
      >
        <Form form={form} layout="vertical" requiredMark={false}>
          <Form.Item
            name="name"
            label={t('radius:certs.form.name')}
            normalize={(value: string) => value?.trim()}
            rules={[{ required: true, message: t('radius:certs.form.nameRequired') }]}
          >
            <Input maxLength={128} />
          </Form.Item>
          {editingCert ? (
            // 证书类型创建后不可修改
            <Form.Item label={t('radius:certs.form.certType')}>
              {certTypeTag(editingCert.certType)}
            </Form.Item>
          ) : (
            <Form.Item name="certType" label={t('radius:certs.form.certType')}>
              <Select
                options={[
                  { value: 'server', label: t('radius:certs.typeServer') },
                  { value: 'ca', label: t('radius:certs.typeCa') },
                ]}
              />
            </Form.Item>
          )}
          {editingCert && (
            <>
              {/* 元数据默认折叠：仅替换材料时按需展开查看 */}
              <Collapse
                style={{ marginBottom: token.margin }}
                items={[
                  {
                    key: 'metadata',
                    label: t('radius:certs.metadataTitle'),
                    children: (
                      <Descriptions column={1} size="small" bordered>
                        <Descriptions.Item label={t('radius:certs.metadata.subject')}>
                          {editingCert.subject || '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.issuer')}>
                          {editingCert.issuer || '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.serial')}>
                          {editingCert.serial || '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.fingerprint')}>
                          {editingCert.fingerprint || '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.notBefore')}>
                          {editingCert.notBefore
                            ? dayjs(editingCert.notBefore).format('YYYY-MM-DD HH:mm:ss')
                            : '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.metadata.notAfter')}>
                          {editingCert.notAfter
                            ? dayjs(editingCert.notAfter).format('YYYY-MM-DD HH:mm:ss')
                            : '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('radius:certs.columns.hasKey')}>
                          {editingCert.hasKey
                            ? t('radius:certs.hasKeyYes')
                            : t('radius:certs.hasKeyNo')}
                        </Descriptions.Item>
                      </Descriptions>
                    ),
                  },
                ]}
              />
              <Divider titlePlacement="start">{t('radius:certs.replaceTitle')}</Divider>
            </>
          )}
          <Form.Item
            name="certPem"
            label={t('radius:certs.form.certPem')}
            extra={editingCert ? t('radius:certs.form.replaceTip') : undefined}
            rules={
              editingCert
                ? []
                : [{ required: true, message: t('radius:certs.form.certPemRequired') }]
            }
          >
            <Input.TextArea rows={5} placeholder="-----BEGIN CERTIFICATE-----" />
          </Form.Item>
          <Form.Item
            name="keyPem"
            label={t('radius:certs.form.keyPem')}
            tooltip={t('radius:certs.form.keyPemTip')}
            extra={editingCert ? t('radius:certs.form.replaceTip') : undefined}
            rules={
              !editingCert && certTypeWatch === 'server'
                ? [{ required: true, message: t('radius:certs.form.keyPemRequired') }]
                : []
            }
          >
            <Input.TextArea rows={5} placeholder="-----BEGIN PRIVATE KEY-----" />
          </Form.Item>
          <Form.Item name="remark" label={t('radius:certs.form.remark')}>
            <Input.TextArea rows={2} maxLength={255} />
          </Form.Item>
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
        <Space orientation="vertical" size={token.margin} style={{ width: '100%' }}>
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
        </Space>
      </Modal>
    </div>
  )
}
