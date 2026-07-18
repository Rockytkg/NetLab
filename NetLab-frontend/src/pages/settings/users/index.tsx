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
import { rbacApi } from '@/services/rbac'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import { createPasswordStrengthRule } from '@/utils/password-strength'
import type { AdminUserView, CreateUserParams, ExportUsersParams, ImportSummary, RoleView, UpdateUserParams } from '@/types/settings'

const { Text } = Typography

type UserRoleValue = string
type CreateUserFormValues = Omit<CreateUserParams, 'role'> & { role: UserRoleValue }
type UpdateUserFormValues = Omit<UpdateUserParams, 'role'> & { role: UserRoleValue }
/** 可分配角色：value 用角色标识（提交给 API），界面展示角色名 */
type AssignableRole = Pick<RoleView, 'role' | 'roleName'>

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

/** 用户管理页：分页列表 + 批量改角色/重置密码/表格导入导出。 */
export default function UsersPage() {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadUsers = can('user.read')
  const [assignableRoles, setAssignableRoles] = useState<AssignableRole[]>([{ role: 'admin', roleName: 'admin' }])

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
  const [exporting, setExporting] = useState(false)
  const [templateDownloading, setTemplateDownloading] = useState(false)

  const load = useCallback(async () => {
    if (!canReadUsers) return
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
  }, [canReadUsers, page, size, keyword, statusFilter, roleFilter])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    rbacApi.listRoles().then((roles) => setAssignableRoles(roles.map(({ role, roleName }) => ({ role, roleName })))).catch(() => undefined)
  }, [])

  // 数据范围由后端 RBAC 资源权限控制。
  const selectedUsers = useMemo(
    () => data.filter((u) => selectedRowKeys.includes(u.id)),
    [data, selectedRowKeys],
  )
  const hasSelection = selectedUsers.length > 0

  // 下拉选项统一展示角色名，value 仍是角色标识，提交/筛选时直接传给 API
  const roleOptions = useMemo(
    () => assignableRoles.map((r) => ({ value: r.role, label: r.roleName || r.role })),
    [assignableRoles],
  )
  // 编辑弹窗补上用户当前角色，避免其不在可分配列表时回显成角色标识
  const editRoleOptions = useMemo(() => {
    if (!editingUser || roleOptions.some((o) => o.value === editingUser.role)) return roleOptions
    return [...roleOptions, { value: editingUser.role, label: editingUser.roleName || editingUser.role }]
  }, [roleOptions, editingUser])

  const columns: ColumnsType<AdminUserView> = [
    {
      title: t('settings:users.columns.username'),
      dataIndex: 'username',
      key: 'username',
      render: (val: string) => (
        <Space>
          <Text strong>{val}</Text>
        </Space>
      ),
    },
    {
      title: t('settings:users.columns.email'),
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: t('settings:users.columns.nickname'),
      dataIndex: 'nickname',
      key: 'nickname',
    },
    {
      title: t('settings:users.columns.phone'),
      dataIndex: 'phone',
      key: 'phone',
    },
    {
      title: t('settings:users.columns.roleName'),
      dataIndex: 'roleName',
      key: 'roleName',
      render: (val: string) => (
        <Tag color="blue">
          {val}
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
      title: t('settings:users.columns.twoFactor'),
      dataIndex: 'twoFactorEnabled',
      key: 'twoFactorEnabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'success' : 'default'}>
          {enabled ? t('settings:users.twoFactorEnabled') : t('settings:users.twoFactorDisabled')}
        </Tag>
      ),
    },
    {
      title: t('settings:users.columns.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (val: string) => new Date(val).toLocaleString(),
    },
    {
      title: t('settings:users.columns.actions'),
      key: 'actions',
      fixed: 'right',
      render: (_, record) => (
        <Space size={token.marginXXS}>
          <Can permission="user.update">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => openEdit(record)}
            >
              {t('common:edit')}
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

  const handleExport = async () => {
    if (!hasSelection) return
    setExporting(true)
    try {
      const params: ExportUsersParams = { userIds: selectedUsers.map((u) => u.id) }
      await adminApi.exportUsers(params, [
        t('settings:users.columns.username'),
        t('settings:users.columns.nickname'),
        t('settings:users.columns.phone'),
        t('settings:users.columns.email'),
        t('settings:users.columns.roleId'),
        t('settings:users.columns.role'),
        t('settings:users.columns.roleName'),
        t('settings:users.columns.status'),
        t('settings:users.columns.createdAt'),
      ])
      message.success(t('settings:users.exportSuccess'))
    } catch {
      // 拦截器已提示错误
    } finally {
      setExporting(false)
    }
  }

  const openCreate = () => {
    createForm.resetFields()
    createForm.setFieldsValue({ role: assignableRoles[0]?.role ?? 'admin' })
    setCreateOpen(true)
  }

  const handleCreateSave = async () => {
    try {
      const values = await createForm.validateFields()
      setCreateSaving(true)
      await adminApi.createUser({
        username: values.username.trim(),
        nickname: values.nickname.trim(),
        phone: values.phone.trim(),
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
      nickname: user.nickname,
      phone: user.phone,
      email: user.email,
      role: user.role,
      status: user.status,
      disableTwoFactor: undefined,
    })
    setEditOpen(true)
  }

  const handleEditSave = async () => {
    if (!editingUser) return
    try {
      const values = await editForm.validateFields()
      setEditSaving(true)
      await adminApi.updateUser(editingUser.id, {
        nickname: values.nickname.trim(),
        phone: values.phone.trim(),
        email: values.email,
        role: values.role,
        status: values.status,
        disableTwoFactor: values.disableTwoFactor === true,
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

  const downloadTemplate = async () => {
    setTemplateDownloading(true)
    try {
      adminApi.downloadImportTemplate([
        t('settings:users.columns.username'),
        t('settings:users.columns.nickname'),
        t('settings:users.columns.phone'),
        t('settings:users.columns.email'),
        t('settings:users.columns.roleId'),
        t('settings:users.columns.role'),
        t('settings:password'),
      ])
    } catch {
      // 拦截器已提示错误
    } finally {
      setTemplateDownloading(false)
    }
  }

  const uploadProps: UploadProps = {
    accept: '.xlsx,.xls,.csv',
    maxCount: 1,
    showUploadList: false,
    beforeUpload: async (file) => {
      const ext = file.name.toLowerCase()
      if (!ext.endsWith('.xlsx') && !ext.endsWith('.xls') && !ext.endsWith('.csv')) {
        message.warning(t('settings:users.invalidImportFile'))
        return false
      }
      setImporting(true)
      setImportResult(null)
      try {
        const summary = await adminApi.importUsers(file)
        setImportResult(summary)
        if (summary.created > 0) {
          message.success(t('settings:users.importDone', { created: summary.created }))
          await load()
        }
      } catch (error) {
        if (error instanceof Error && error.message !== 'canceled') {
          message.error(t('settings:users.invalidImportData'))
        }
      } finally {
        setImporting(false)
      }
      return false // 阻止默认上传
    },
  }

  if (!canReadUsers) {
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
            <Can permission="user.create"><Button type="primary" icon={<UserAddOutlined />} onClick={openCreate}>
              {t('settings:users.addUser')}
            </Button></Can>
            <Can permission="user.update"><Button icon={<KeyOutlined />} disabled={!hasSelection} onClick={() => setPwOpen(true)}>
              {t('settings:users.resetPassword')}
            </Button></Can>
            <Can permission="user.read"><Button
              icon={<DownloadOutlined />}
              disabled={!hasSelection}
              loading={exporting}
              onClick={handleExport}
            >
              {t('settings:users.export')}
            </Button></Can>
            <Can permission="user.delete"><Button
              danger
              icon={<DeleteOutlined />}
              disabled={!hasSelection}
              onClick={handleBatchDelete}
            >
              {t('settings:users.delete')}
            </Button></Can>
            {hasSelection && (
              <Text type="secondary">
                {t('settings:users.selectedCount', { count: selectedUsers.length })}
              </Text>
            )}
          </Space>
          <Space wrap>
            <Can permission="user.import"><Button icon={<UploadOutlined />} onClick={() => setImportOpen(true)}>
              {t('settings:users.importUsers')}
            </Button></Can>
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
              options={roleOptions}
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
            name="nickname"
            label={t('settings:users.columns.nickname')}
            rules={[{ required: true, message: t('settings:users.nicknameRequired') }]}
          >
            <Input maxLength={64} />
          </Form.Item>
          <Form.Item
            name="phone"
            label={t('settings:users.columns.phone')}
            rules={[{ required: true, message: t('settings:users.phoneRequired') }, { pattern: /^1[3-9]\d{9}$/, message: t('settings:users.phoneInvalid') }]}
          >
            <Input maxLength={11} autoComplete="tel" />
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
              options={roleOptions}
            />
          </Form.Item>
          <Form.Item label={t('settings:users.initialPassword')} required>
            <Space.Compact style={{ width: '100%' }}>
              <Form.Item
                name="password"
                noStyle
                rules={[
                  { required: true, message: t('settings:changePassword.newRequired') },
                  createPasswordStrengthRule({
                    t,
                  }),
                ]}
              >
                <Input.Password autoComplete="new-password" maxLength={72} />
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
            name="nickname"
            label={t('settings:users.columns.nickname')}
            rules={[{ required: true, message: t('settings:users.nicknameRequired') }]}
          >
            <Input maxLength={64} />
          </Form.Item>
          <Form.Item
            name="phone"
            label={t('settings:users.columns.phone')}
            rules={[{ required: true, message: t('settings:users.phoneRequired') }, { pattern: /^1[3-9]\d{9}$/, message: t('settings:users.phoneInvalid') }]}
          >
            <Input maxLength={11} autoComplete="tel" />
          </Form.Item>
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
              options={editRoleOptions}
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
          {editingUser?.twoFactorEnabled && (
            <Form.Item
              name="disableTwoFactor"
              label={t('settings:users.disableTwoFactor')}
              extra={t('settings:users.disableTwoFactorHint')}
            >
              <Select
                allowClear
                placeholder={t('settings:users.twoFactorEnabled')}
                options={[{
                  value: true,
                  label: t('settings:users.twoFactorDisabled'),
                }]}
              />
            </Form.Item>
          )}
        </Form>
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
          {t('settings:users.resetHint', { count: selectedUsers.length })}
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
                  createPasswordStrengthRule({
                    t,
                  }),
                ]}
              >
                <Input.Password autoComplete="new-password" maxLength={72} />
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

      {/* 用户导入 */}
      <Modal
        title={t('settings:users.importUsers')}
        open={importOpen}
        onCancel={() => {
          setImportOpen(false)
          setImportResult(null)
        }}
        footer={null}
      >
        <Space orientation="vertical" size={token.margin} style={{ width: '100%' }}>
          <Alert type="info" showIcon title={t('settings:users.importUsersFormat')} />
          <Space.Compact block>
            <Can permission="user.import"><Button icon={<DownloadOutlined />} onClick={downloadTemplate} loading={templateDownloading}>
              {t('settings:users.downloadTemplate')}
            </Button></Can>
            <Can permission="user.import"><Upload {...uploadProps}>
              <Button type="primary" icon={<UploadOutlined />} loading={importing}>
                {t('settings:users.importUsers')}
              </Button>
            </Upload></Can>
          </Space.Compact>

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
