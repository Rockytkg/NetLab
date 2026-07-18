import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  App,
  Button,
  Card,
  Form,
  Input,
  Modal,
  Result,
  Space,
  Spin,
  Table,
  Tag,
  Tree,
  Typography,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import type { TreeDataNode } from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { rbacApi } from '@/services/rbac'
import { usePermission } from '@/hooks/usePermission'
import Can from '@/components/auth/Can'
import type { PermissionView, RoleView } from '@/types/settings'

const { Text } = Typography

interface CreateRoleFormValues {
  role: string
  roleName: string
  description?: string
}

interface UpdateRoleFormValues {
  roleName: string
  description?: string
}

/** 系统管理菜单下的资源节点，与左侧菜单结构对应。 */
const ADMIN_RESOURCES = [
  { resource: 'setting', menuTitleKey: 'settings' },
  { resource: 'user', menuTitleKey: 'userManagement' },
  { resource: 'rbac', menuTitleKey: 'roleManagement' },
  { resource: 'log', menuTitleKey: 'loginLogs' },
] as const

/** 账户相关的资源节点（不属于系统管理菜单）。 */
const ACCOUNT_RESOURCES = ['auth']

interface PermissionTreeProps {
  permissions: PermissionView[]
  value: string[]
  onChange: (value: string[]) => void
}

/** 权限树：按左侧菜单层级（系统管理 > 系统设置/用户管理/角色管理/登录日志）分组展示权限。 */
function PermissionTree({ permissions, value, onChange }: PermissionTreeProps) {
  const { t } = useTranslation(['settings', 'menu'])
  const permissionCodes = useMemo(() => new Set(permissions.map((item) => item.code)), [permissions])

  const treeData = useMemo<TreeDataNode[]>(() => {
    const byResource = new Map<string, PermissionView[]>()
    permissions.forEach((item) => {
      const list = byResource.get(item.resource) ?? []
      list.push(item)
      byResource.set(item.resource, list)
    })
    const toLeaves = (items: PermissionView[]): TreeDataNode[] =>
      items.map((item) => ({
        key: item.code,
        title: t(`settings:roles.permissionNames.${item.code}`, item.description || item.code),
      }))

    const nodes: TreeDataNode[] = []
    const adminChildren = ADMIN_RESOURCES.filter(({ resource }) =>
      byResource.has(resource),
    ).map(({ resource, menuTitleKey }) => ({
      key: `menu:${resource}`,
      title: t(`menu:${menuTitleKey}`),
      children: toLeaves(byResource.get(resource) ?? []),
    }))
    if (adminChildren.length > 0) {
      nodes.push({
        key: 'menu:administration',
        title: t('menu:administration'),
        children: adminChildren,
      })
    }
    const accountItems = ACCOUNT_RESOURCES.flatMap((resource) => byResource.get(resource) ?? [])
    if (accountItems.length > 0) {
      nodes.push({
        key: 'menu:account',
        title: t('settings:roles.accountGroup'),
        children: toLeaves(accountItems),
      })
    }
    // 未映射到菜单分组的资源兜底展示，避免新权限被隐藏
    const mapped = new Set<string>([
      ...ADMIN_RESOURCES.map((item) => item.resource),
      ...ACCOUNT_RESOURCES,
    ])
    const unmapped = permissions.filter((item) => !mapped.has(item.resource))
    if (unmapped.length > 0) {
      nodes.push({
        key: 'menu:other',
        title: t('settings:roles.otherGroup'),
        children: toLeaves(unmapped),
      })
    }
    return nodes
  }, [permissions, t])

  return (
    <Tree
      checkable
      defaultExpandAll
      selectable={false}
      treeData={treeData}
      checkedKeys={value}
      onCheck={(checked) => {
        const keys = Array.isArray(checked) ? checked : checked.checked
        onChange(keys.map(String).filter((key) => permissionCodes.has(key)))
      }}
    />
  )
}

/** 角色管理页：角色列表 + 新增/编辑/删除角色 + 配置角色权限。 */
export default function RolesPage() {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message, modal } = App.useApp()
  const { can } = usePermission()
  const canReadRbac = can('rbac.read')

  const [roles, setRoles] = useState<RoleView[]>([])
  const [permissions, setPermissions] = useState<PermissionView[]>([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')

  // 新增角色弹窗
  const [createOpen, setCreateOpen] = useState(false)
  const [createForm] = Form.useForm<CreateRoleFormValues>()
  const [createSaving, setCreateSaving] = useState(false)

  // 编辑角色弹窗
  const [editOpen, setEditOpen] = useState(false)
  const [editingRole, setEditingRole] = useState<RoleView | null>(null)
  const [editForm] = Form.useForm<UpdateRoleFormValues>()
  const [editSaving, setEditSaving] = useState(false)

  // 权限配置弹窗
  const [permOpen, setPermOpen] = useState(false)
  const [permRole, setPermRole] = useState<RoleView | null>(null)
  const [permSelected, setPermSelected] = useState<string[]>([])
  const [permLoading, setPermLoading] = useState(false)
  const [permSaving, setPermSaving] = useState(false)

  const load = useCallback(async () => {
    if (!canReadRbac) return
    setLoading(true)
    try {
      const [roleList, permissionList] = await Promise.all([
        rbacApi.listRoles(),
        rbacApi.listPermissions(),
      ])
      setRoles(roleList ?? [])
      setPermissions(permissionList ?? [])
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [canReadRbac])

  useEffect(() => {
    load()
  }, [load])

  const filteredRoles = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    if (!keyword) return roles
    return roles.filter((role) =>
      [role.role, role.roleName, role.description ?? ''].some((field) =>
        field.toLowerCase().includes(keyword),
      ),
    )
  }, [roles, search])

  const openCreate = () => {
    createForm.resetFields()
    setCreateOpen(true)
  }

  const handleCreateSave = async () => {
    try {
      const values = await createForm.validateFields()
      setCreateSaving(true)
      await rbacApi.createRole(values.role.trim(), values.roleName.trim(), values.description?.trim())
      message.success(t('settings:roles.roleCreated'))
      setCreateOpen(false)
      createForm.resetFields()
      await load()
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return
    } finally {
      setCreateSaving(false)
    }
  }

  const openEdit = (role: RoleView) => {
    setEditingRole(role)
    editForm.setFieldsValue({ roleName: role.roleName, description: role.description })
    setEditOpen(true)
  }

  const handleEditSave = async () => {
    if (!editingRole) return
    try {
      const values = await editForm.validateFields()
      setEditSaving(true)
      await rbacApi.updateRole(editingRole.id, values.roleName.trim(), values.description?.trim())
      message.success(t('settings:roles.roleUpdated'))
      setEditOpen(false)
      setEditingRole(null)
      await load()
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return
    } finally {
      setEditSaving(false)
    }
  }

  const openPermissions = async (role: RoleView) => {
    setPermRole(role)
    setPermOpen(true)
    setPermSelected([])
    setPermLoading(true)
    try {
      const refs = await rbacApi.getRolePermissions(role.id)
      setPermSelected(refs.map((ref) => ref.code))
    } catch {
      // 拦截器已提示错误
    } finally {
      setPermLoading(false)
    }
  }

  const handlePermSave = async () => {
    if (!permRole) return
    setPermSaving(true)
    try {
      await rbacApi.setRolePermissions(permRole.id, permSelected)
      message.success(t('settings:roles.permissionsUpdated'))
      setPermOpen(false)
      setPermRole(null)
    } catch {
      // 拦截器已提示错误
    } finally {
      setPermSaving(false)
    }
  }

  const handleDelete = (role: RoleView) => {
    modal.confirm({
      title: t('settings:roles.deleteConfirmTitle'),
      content: t('settings:roles.deleteConfirm', { name: role.roleName || role.role }),
      okText: t('common:confirm'),
      cancelText: t('common:cancel'),
      okButtonProps: { danger: true },
      async onOk() {
        await rbacApi.deleteRole(role.id)
        message.success(t('settings:roles.deleted'))
        await load()
      },
    })
  }

  const columns: ColumnsType<RoleView> = [
    {
      title: t('settings:roles.columns.role'),
      dataIndex: 'role',
      key: 'role',
      render: (val: string, record) => (
        <Space>
          <Text strong>{val}</Text>
          {record.hidden && <Tag>{t('settings:roles.hiddenTag')}</Tag>}
        </Space>
      ),
    },
    {
      title: t('settings:roles.columns.roleName'),
      dataIndex: 'roleName',
      key: 'roleName',
    },
    {
      title: t('settings:roles.columns.description'),
      dataIndex: 'description',
      key: 'description',
      render: (val?: string) => val || '-',
    },
    {
      title: t('settings:roles.columns.type'),
      dataIndex: 'type',
      key: 'type',
      render: (val: RoleView['type']) => (
        <Tag color={val === 'builtin' ? 'gold' : 'blue'}>
          {t(`settings:roles.type.${val}`, val)}
        </Tag>
      ),
    },
    {
      title: t('settings:roles.columns.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (val: string) => new Date(val).toLocaleString(),
    },
    {
      title: t('settings:roles.columns.actions'),
      key: 'actions',
      fixed: 'right',
      render: (_, record) => {
        // 内置角色由系统维护，不可改名、改权限或删除。
        const immutable = record.type === 'builtin'
        return (
          <Space size={token.marginXXS}>
            <Can permission="rbac.write">
              <Button
                type="text"
                size="small"
                icon={<SafetyCertificateOutlined />}
                disabled={immutable}
                onClick={() => openPermissions(record)}
              >
                {t('settings:roles.configurePermissions')}
              </Button>
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                disabled={immutable}
                onClick={() => openEdit(record)}
              >
                {t('common:edit')}
              </Button>
              <Button
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                disabled={immutable}
                onClick={() => handleDelete(record)}
              >
                {t('settings:roles.delete')}
              </Button>
            </Can>
          </Space>
        )
      },
    },
  ]

  if (!canReadRbac) {
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
            <Can permission="rbac.write">
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                {t('settings:roles.addRole')}
              </Button>
            </Can>
          </Space>
          <Space wrap>
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('settings:roles.searchPlaceholder')}
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
          dataSource={filteredRoles}
          loading={loading}
          pagination={false}
          scroll={{ x: 'max-content' }}
        />
      </Card>

      {/* 新增角色 */}
      <Modal
        title={t('settings:roles.addRole')}
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
            name="role"
            label={t('settings:roles.columns.role')}
            extra={t('settings:roles.roleCodeHint')}
            normalize={(value: string) => value?.trim()}
            rules={[
              { required: true, message: t('settings:roles.roleRequired') },
              { min: 2, message: t('settings:roles.roleLength') },
              { max: 64, message: t('settings:roles.roleLength') },
              { pattern: /^[^\s.]+$/, message: t('settings:roles.roleInvalid') },
            ]}
          >
            <Input maxLength={64} />
          </Form.Item>
          <Form.Item
            name="roleName"
            label={t('settings:roles.columns.roleName')}
            normalize={(value: string) => value?.trim()}
            rules={[{ required: true, message: t('settings:roles.roleNameRequired') }]}
          >
            <Input maxLength={128} />
          </Form.Item>
          <Form.Item name="description" label={t('settings:roles.columns.description')}>
            <Input.TextArea maxLength={255} rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑角色 */}
      <Modal
        title={t('settings:roles.editRole')}
        open={editOpen}
        onCancel={() => {
          setEditOpen(false)
          setEditingRole(null)
        }}
        onOk={handleEditSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={editSaving}
        forceRender
      >
        <Form form={editForm} layout="vertical" requiredMark={false}>
          <Form.Item
            name="roleName"
            label={t('settings:roles.columns.roleName')}
            normalize={(value: string) => value?.trim()}
            rules={[{ required: true, message: t('settings:roles.roleNameRequired') }]}
          >
            <Input maxLength={128} />
          </Form.Item>
          <Form.Item name="description" label={t('settings:roles.columns.description')}>
            <Input.TextArea maxLength={255} rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 配置角色权限 */}
      <Modal
        title={t('settings:roles.permissionsTitle', { name: permRole?.roleName ?? '' })}
        open={permOpen}
        onCancel={() => {
          setPermOpen(false)
          setPermRole(null)
        }}
        onOk={handlePermSave}
        okText={t('common:confirm')}
        cancelText={t('common:cancel')}
        confirmLoading={permSaving}
      >
        {permLoading ? (
          <div style={{ display: 'flex', justifyContent: 'center', padding: token.paddingLG }}>
            <Spin />
          </div>
        ) : (
          <PermissionTree
            permissions={permissions}
            value={permSelected}
            onChange={setPermSelected}
          />
        )}
      </Modal>
    </div>
  )
}
