import { useCallback, useEffect, useMemo, useState, type Key } from 'react'
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  List,
  Modal,
  Result,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  Upload,
  App,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import type { UploadProps } from 'antd'
import {
  SafetyCertificateOutlined,
  KeyOutlined,
  UploadOutlined,
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  EditOutlined,
  DownloadOutlined,
  UserAddOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import { useAuthStore } from '@/stores/authStore'
import type { AdminUserView, CreateUserParams, ImportSummary, UpdateUserParams } from '@/types/settings'

const { Text } = Typography

const ASSIGNABLE_ROLES = ['admin', 'editor', 'viewer'] as const
type UserRoleValue = (typeof ASSIGNABLE_ROLES)[number]
type CreateUserFormValues = Omit<CreateUserParams, 'role'> & { role: UserRoleValue }
type UpdateUserFormValues = Omit<UpdateUserParams, 'role'> & { role: UserRoleValue }

function generateStrongPassword() {
  const upper = 'ABCDEFGHJKLMNPQRSTUVWXYZ'
  const lower = 'abcdefghijkmnopqrstuvwxyz'
  const digits = '23456789'
  const symbols = '!@#$%^&*'
  const all = upper + lower + digits + symbols
  const pick = (chars: string) => chars[randomIndex(chars.length)]
  const chars = [pick(upper), pick(lower), pick(digits), pick(symbols)]
  while (chars.length < 16) {
    chars.push(pick(all))
  }
  for (let i = chars.length - 1; i > 0; i--) {
    const j = randomIndex(i + 1)
    ;[chars[i], chars[j]] = [chars[j], chars[i]]
  }
  return chars.join('')
}

function randomIndex(max: number) {
  const bytes = new Uint32Array(1)
  window.crypto.getRandomValues(bytes)
  return bytes[0] % max
}

/** 用户管理页（仅 admin）：分页列表 + 批量改角色/重置密码/CSV 导入。 */
export default function UsersPage() {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message, modal } = App.useApp()
  const currentRole = useAuthStore((s) => s.userInfo?.role)
  const isAdmin = currentRole === 'admin' || currentRole === 'super_admin'

  const [data, setData] = useState<AdminUserView[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<string | undefined>()
  const [roleFilter, setRoleFilter] = useState<string | undefined>()
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])

  // 单用户添加弹窗
  const [createOpen, setCreateOpen] = useState(false)
  const [createForm] = Form.useForm<CreateUserFormValues>()
  const [createSaving, setCreateSaving] = useState(false)

  // 角色弹窗
  const [roleOpen, setRoleOpen] = useState(false)
  const [roleValue, setRoleValue] = useState<UserRoleValue>('viewer')
  const [roleSaving, setRoleSaving] = useState(false)

  // 重置密码弹窗
  const [pwOpen, setPwOpen] = useState(false)
  const [pwForm] = Form.useForm<{ newPassword: string }>()
  const [pwSaving, setPwSaving] = useState(false)

  // 单用户编辑弹窗
  const [editOpen, setEditOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<AdminUserView | null>(null)
  const [editForm] = Form.useForm<UpdateUserFormValues>()
  const [editSaving, setEditSaving] = useState(false)

  // 导入弹窗
  const [importOpen, setImportOpen] = useState(false)
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState<ImportSummary | null>(null)
  const [importPreview, setImportPreview] = useState<string[][]>([])

  const load = useCallback(async () => {
    if (!isAdmin) return
    setLoading(true)
    try {
      const res = await adminApi.listUsers({ page, size, keyword, status: statusFilter, role: roleFilter })
      setData(res.items ?? [])
      setTotal(res.total ?? 0)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [isAdmin, page, size, keyword, statusFilter, roleFilter])

  useEffect(() => {
    load()
  }, [load])

  const selectedUsers = useMemo(
    () => data.filter((u) => selectedRowKeys.includes(u.id)),
    [data, selectedRowKeys],
  )
  const hasSelection = selectedRowKeys.length > 0

  const columns: ColumnsType<AdminUserView> = [
    {
      title: t('settings:users.columns.username'),
      dataIndex: 'username',
      key: 'username',
      render: (val: string, record) => (
        <Space>
          <Text strong>{val}</Text>
          {record.isAdmin && <Tag color="gold">{t('settings:users.adminTag')}</Tag>}
        </Space>
      ),
    },
    {
      title: t('settings:users.columns.email'),
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: t('settings:users.columns.role'),
      dataIndex: 'role',
      key: 'role',
      render: (val: string) => (
        <Tag color={val === 'admin' ? 'gold' : 'blue'}>
          {t(`settings:profile.role.${val}`, val)}
        </Tag>
      ),
    },
    {
      title: t('settings:users.columns.status'),
      dataIndex: 'status',
      key: 'status',
      render: (val: string) => (
        <Tag color={val === 'active' ? 'success' : val === 'locked' ? 'warning' : 'default'}>
          {t(`settings:users.status.${val}`, val)}
        </Tag>
      ),
    },
    {
      title: t('settings:users.columns.lastLogin'),
      dataIndex: 'lastLoginAt',
      key: 'lastLoginAt',
      render: (val: string | null) => (val ? new Date(val).toLocaleString() : '-'),
    },
    {
      title: t('settings:users.columns.actions'),
      key: 'actions',
      fixed: 'right',
      render: (_, record) => (
        <Space size={token.marginXXS}>
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEdit(record)}
          >
            {t('common:edit')}
          </Button>
        </Space>
      ),
    },
  ]

  const handleSearch = () => {
    setPage(1)
    setKeyword(search.trim())
  }

  const openCreate = () => {
    createForm.resetFields()
    createForm.setFieldsValue({ role: 'viewer' })
    setCreateOpen(true)
  }

  const handleCreateSave = async () => {
    try {
      const values = await createForm.validateFields()
      setCreateSaving(true)
      await adminApi.createUser({
        username: values.username.trim(),
        email: values.email.trim(),
        password: values.password,
        role: values.role,
      })
      message.success(t('settings:users.userCreated'))
      setCreateOpen(false)
      createForm.resetFields()
      setSelectedRowKeys([])
      const reloadCurrentPage = page === 1
      setPage(1)
      if (reloadCurrentPage) {
        await load()
      }
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return
    } finally {
      setCreateSaving(false)
    }
  }

  const openEdit = (user: AdminUserView) => {
    setEditingUser(user)
    editForm.setFieldsValue({
      email: user.email,
      role: (ASSIGNABLE_ROLES.includes(user.role as UserRoleValue) ? user.role : 'viewer') as UserRoleValue,
      status: user.status,
    })
    setEditOpen(true)
  }

  const handleEditSave = async () => {
    if (!editingUser) return
    try {
      const values = await editForm.validateFields()
      setEditSaving(true)
      await adminApi.updateUser(editingUser.id, {
        email: values.email,
        role: values.role,
        status: values.status,
      })
      message.success(t('settings:users.userUpdated'))
      setEditOpen(false)
      setEditingUser(null)
      await load()
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return
    } finally {
      setEditSaving(false)
    }
  }

  const openRoleModal = () => {
    setRoleValue('viewer')
    setRoleOpen(true)
  }

  const handleBatchDelete = () => {
    modal.confirm({
      title: t('settings:users.deleteConfirmTitle'),
      content: t('settings:users.deleteConfirm', { count: selectedUsers.length }),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        await adminApi.batchDeleteUsers(selectedUsers.map((u) => u.id))
        message.success(t('settings:users.deleted'))
        setSelectedRowKeys([])
        await load()
      },
    })
  }

  const handleRoleSave = async () => {
    if (!roleValue) {
      message.warning(t('settings:users.roleRequired'))
      return
    }
    setRoleSaving(true)
    try {
      await adminApi.batchUpdateRole(
        selectedUsers.map((u) => u.id),
        roleValue,
      )
      message.success(t('settings:users.roleUpdated'))
      setRoleOpen(false)
      setSelectedRowKeys([])
      await load()
    } catch {
      // 拦截器已提示错误
    } finally {
      setRoleSaving(false)
    }
  }

  const handleResetSave = async () => {
    try {
      const { newPassword } = await pwForm.validateFields()
      setPwSaving(true)
      await adminApi.batchResetPassword(
        selectedUsers.map((u) => u.id),
        newPassword,
      )
      message.success(t('settings:users.passwordReset'))
      setPwOpen(false)
      pwForm.resetFields()
      setSelectedRowKeys([])
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return // 表单校验失败
      // 其余错误已由拦截器提示
    } finally {
      setPwSaving(false)
    }
  }

  const downloadTemplate = () => {
    const csv = [
      'username(required),email(required),role(viewer|editor optional),password(optional min 8 with letters and numbers)',
      'alice,alice@example.com,viewer,Password123',
      'bob,bob@example.com,editor,Password123',
    ].join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'netlab-users-template.csv'
    a.click()
    URL.revokeObjectURL(url)
  }

  const parsePreview = async (file: File) => {
    const text = await file.text()
    const rows = text
      .split(/\r?\n/)
      .map((line) => line.split(',').map((cell) => cell.trim()))
      .filter((row) => row.some(Boolean))
      .slice(0, 6)
    setImportPreview(rows)
  }

  const uploadProps: UploadProps = {
    accept: '.csv',
    maxCount: 1,
    showUploadList: false,
    beforeUpload: async (file) => {
      if (!file.name.toLowerCase().endsWith('.csv')) {
        message.warning(t('settings:users.invalidFile'))
        return false
      }
      setImporting(true)
      setImportResult(null)
      try {
        await parsePreview(file)
        const summary = await adminApi.importUsers(file)
        setImportResult(summary)
        if (summary.created > 0) {
          message.success(t('settings:users.importDone', { created: summary.created }))
          await load()
        }
      } catch {
        // 拦截器已提示错误
      } finally {
        setImporting(false)
      }
      return false // 阻止默认上传
    },
  }

  if (!isAdmin) {
    return <Result status="403" title="403" subTitle={t('settings:adminOnly')} />
  }

  return (
    <div style={{ width: '100%' }}>
      <Card variant="outlined">
        <Space
          style={{ marginBottom: token.margin, width: '100%', justifyContent: 'space-between' }}
          wrap
        >
          <Space wrap>
            <Button type="primary" icon={<UserAddOutlined />} onClick={openCreate}>
              {t('settings:users.addUser')}
            </Button>
            <Button
              icon={<SafetyCertificateOutlined />}
              disabled={!hasSelection}
              onClick={openRoleModal}
            >
              {t('settings:users.changeRole')}
            </Button>
            <Button icon={<KeyOutlined />} disabled={!hasSelection} onClick={() => setPwOpen(true)}>
              {t('settings:users.resetPassword')}
            </Button>
            <Button
              danger
              icon={<DeleteOutlined />}
              disabled={!hasSelection}
              onClick={handleBatchDelete}
            >
              {t('settings:users.delete')}
            </Button>
            {hasSelection && (
              <Text type="secondary">
                {t('settings:users.selectedCount', { count: selectedRowKeys.length })}
              </Text>
            )}
          </Space>
          <Space wrap>
            <Button icon={<UploadOutlined />} onClick={() => setImportOpen(true)}>
              {t('settings:users.import')}
            </Button>
            <Select
              allowClear
              value={statusFilter}
              onChange={(val) => {
                setPage(1)
                setStatusFilter(val)
              }}
              placeholder={t('settings:users.statusFilter')}
              style={{ width: 140 }}
              options={['active', 'disabled', 'locked'].map((value) => ({
                value,
                label: t(`settings:users.status.${value}`, value),
              }))}
            />
            <Select
              allowClear
              value={roleFilter}
              onChange={(val) => {
                setPage(1)
                setSelectedRowKeys([])
                setRoleFilter(val)
              }}
              placeholder={t('settings:users.roleFilter')}
              style={{ width: 140 }}
              options={ASSIGNABLE_ROLES.map((value) => ({
                value,
                label: t(`settings:profile.role.${value}`, value),
              }))}
            />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onPressEnter={handleSearch}
              placeholder={t('settings:users.searchPlaceholder')}
              prefix={<SearchOutlined />}
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
            showTotal: (tt) => t('settings:users.total', { total: tt }),
          }}
          scroll={{ x: 'max-content' }}
        />
      </Card>

      {/* 单用户添加 */}
      <Modal
        title={t('settings:users.addUser')}
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false)
          createForm.resetFields()
        }}
        onOk={handleCreateSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={createSaving}
        forceRender
      >
        <Form form={createForm} layout="vertical" requiredMark={false}>
          <Form.Item
            name="username"
            label={t('settings:users.columns.username')}
            normalize={(value: string) => value?.trim()}
            rules={[
              { required: true, message: t('settings:users.usernameRequired') },
              { min: 3, message: t('settings:users.usernameLength') },
              { max: 64, message: t('settings:users.usernameLength') },
              { pattern: /^[A-Za-z0-9_-]+$/, message: t('settings:users.usernameInvalid') },
            ]}
          >
            <Input maxLength={64} autoComplete="username" />
          </Form.Item>
          <Form.Item
            name="email"
            label={t('settings:users.columns.email')}
            normalize={(value: string) => value?.trim()}
            rules={[
              { required: true, message: t('settings:users.emailRequired') },
              { type: 'email', message: t('settings:users.emailInvalid') },
            ]}
          >
            <Input maxLength={255} autoComplete="email" />
          </Form.Item>
          <Form.Item
            name="role"
            label={t('settings:users.columns.role')}
            rules={[{ required: true, message: t('settings:users.roleRequired') }]}
          >
            <Select
              options={ASSIGNABLE_ROLES.map((r) => ({
                value: r,
                label: t(`settings:profile.role.${r}`, r),
              }))}
            />
          </Form.Item>
          <Form.Item label={t('settings:users.initialPassword')} required>
            <Space.Compact style={{ width: '100%' }}>
              <Form.Item
                name="password"
                noStyle
                rules={[
                  { required: true, message: t('settings:changePassword.newRequired') },
                  { min: 8, message: t('settings:changePassword.minLength') },
                  {
                    pattern: /^(?=.*[A-Za-z])(?=.*\d).+$/,
                    message: t('settings:users.passwordComplexity'),
                  },
                ]}
              >
                <Input.Password autoComplete="new-password" maxLength={128} />
              </Form.Item>
              <Button
                icon={<ThunderboltOutlined />}
                onClick={() => createForm.setFieldValue('password', generateStrongPassword())}
              >
                {t('settings:users.generatePassword')}
              </Button>
            </Space.Compact>
          </Form.Item>
        </Form>
      </Modal>

      {/* 单用户编辑 */}
      <Modal
        title={t('settings:users.editUser')}
        open={editOpen}
        onCancel={() => setEditOpen(false)}
        onOk={handleEditSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={editSaving}
      >
        <Form form={editForm} layout="vertical" requiredMark={false}>
          <Form.Item
            name="email"
            label={t('settings:users.columns.email')}
            rules={[
              { required: true, message: t('settings:users.emailRequired') },
              { type: 'email', message: t('settings:users.emailInvalid') },
            ]}
          >
            <Input maxLength={255} />
          </Form.Item>
          <Form.Item
            name="role"
            label={t('settings:users.columns.role')}
            rules={[{ required: true, message: t('settings:users.roleRequired') }]}
          >
            <Select
              options={ASSIGNABLE_ROLES.map((r) => ({
                value: r,
                label: t(`settings:profile.role.${r}`, r),
              }))}
            />
          </Form.Item>
          <Form.Item
            name="status"
            label={t('settings:users.columns.status')}
            rules={[{ required: true, message: t('settings:users.statusRequired') }]}
          >
            <Select
              options={['active', 'disabled', 'locked'].map((value) => ({
                value,
                label: t(`settings:users.status.${value}`, value),
              }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 批量改角色 */}
      <Modal
        title={t('settings:users.changeRole')}
        open={roleOpen}
        onCancel={() => setRoleOpen(false)}
        onOk={handleRoleSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={roleSaving}
      >
        <Space orientation="vertical" size={token.margin} style={{ width: '100%' }}>
          <Text type="secondary">
            {t('settings:users.changeRoleHint', { count: selectedRowKeys.length })}
          </Text>
          <Select
            value={roleValue}
            onChange={setRoleValue}
            style={{ width: '100%' }}
            options={ASSIGNABLE_ROLES.map((r) => ({
              value: r,
              label: t(`settings:profile.role.${r}`, r),
            }))}
          />
        </Space>
      </Modal>

      {/* 批量重置密码 */}
      <Modal
        title={t('settings:users.resetPassword')}
        open={pwOpen}
        onCancel={() => setPwOpen(false)}
        onOk={handleResetSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={pwSaving}
      >
        <Text type="secondary">
          {t('settings:users.resetHint', { count: selectedRowKeys.length })}
        </Text>
        <Form form={pwForm} layout="vertical" style={{ marginTop: token.margin }}>
          <Form.Item
            label={t('settings:users.newPassword')}
            required
          >
            <Space.Compact style={{ width: '100%' }}>
              <Form.Item
                name="newPassword"
                noStyle
                rules={[
                  { required: true, message: t('settings:changePassword.newRequired') },
                  { min: 8, message: t('settings:changePassword.minLength') },
                  {
                    pattern: /^(?=.*[A-Za-z])(?=.*\d).+$/,
                    message: t('settings:users.passwordComplexity'),
                  },
                ]}
              >
                <Input.Password autoComplete="new-password" maxLength={128} />
              </Form.Item>
              <Button
                icon={<ThunderboltOutlined />}
                onClick={() => pwForm.setFieldValue('newPassword', generateStrongPassword())}
              >
                {t('settings:users.generatePassword')}
              </Button>
            </Space.Compact>
          </Form.Item>
        </Form>
      </Modal>

      {/* CSV 导入 */}
      <Modal
        title={t('settings:users.import')}
        open={importOpen}
        onCancel={() => {
          setImportOpen(false)
          setImportResult(null)
          setImportPreview([])
        }}
        footer={null}
      >
        <Space orientation="vertical" size={token.margin} style={{ width: '100%' }}>
          <Alert type="info" showIcon title={t('settings:users.importFormat')} />
          <Space.Compact block>
            <Button icon={<DownloadOutlined />} onClick={downloadTemplate}>
              {t('settings:users.downloadTemplate')}
            </Button>
            <Upload {...uploadProps}>
              <Button type="primary" icon={<UploadOutlined />} loading={importing}>
                {t('settings:users.selectFile')}
              </Button>
            </Upload>
          </Space.Compact>

          {importPreview.length > 0 && (
            <Card size="small" title={t('settings:users.preview')}>
              <List
                size="small"
                dataSource={importPreview}
                renderItem={(row, index) => (
                  <List.Item style={{ paddingInline: 0 }}>
                    <Text type={index === 0 ? 'secondary' : undefined}>{row.join(' | ')}</Text>
                  </List.Item>
                )}
              />
            </Card>
          )}

          {importResult && (
            <Alert
              type={importResult.errors.length > 0 ? 'warning' : 'success'}
              showIcon
              title={t('settings:users.importResult', {
                created: importResult.created,
                skipped: importResult.skipped,
              })}
              description={
                importResult.errors.length > 0 ? (
                  <List
                    size="small"
                    dataSource={importResult.errors}
                    renderItem={(e) => (
                      <List.Item style={{ paddingInline: 0 }}>
                        <Text type="danger" style={{ fontSize: token.fontSizeSM }}>
                          {e}
                        </Text>
                      </List.Item>
                    )}
                  />
                ) : null
              }
            />
          )}
        </Space>
      </Modal>
    </div>
  )
}
