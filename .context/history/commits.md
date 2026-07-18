# 提交历史

> 自动生成于 2026-07-18 10:33:16

## 2026-07-18 — 角色从扁平字符串重构为关系化 Role 表

- **类型**: refactor
- **分支**: main
- **文件数**: 27 (+230/-113)
- **决策**:
  - 角色表使用 role 保存角色标识、roleName 保存展示名称；用户表使用 role_id 关联角色，认证接口通过角色标识计算权限后返回角色名
  - JWT claims、Casbin 策略主体和权限查询统一使用角色 ID；角色标识仅作为角色/用户创建与导入时的输入键
  - auth_handler 中 8 处重复的 PermKeysForRole 调用统一抽取为 applyRoleInfo() 方法，同时承担角色 ID→角色名的回填职责
  - Repository 层新增 roleID() 辅助方法，将角色标识统一解析为 role_id，用于 UpdateManagedFields/BatchUpdateRole 等写操作
  - User 模型 Role 字段标记 gorm:"-" 不再持久化，改为 AfterFind 从 Role 表实时回填角色标识
- **测试**: rbac_test.go 适配 CreateRole/UpdateRole 新增的 role/roleName 参数

## 2026-07-17 — ♻️ refactor(admin): 清扫冗余：移除未使用代码并抽取公共加载逻辑

- **类型**: refactor
- **分支**: main
- **文件数**: 4 (+8/-68)
- **决策**:
  - 移除 auth middleware 中已废弃的 ContextKeyUsername/ContextKeySessionID 常量和 GetUsername 函数：这些符号在 RBAC 迁移后不再使用
  - 移除 User 模型 IsLocked/IsPrivileged/IsSuperAdmin 方法：状态/角色判断已统一由 Casbin 和 RBAC 层处理
  - 抽取设置页 loadSettings 公共加载逻辑：消除手动 Promise IIFE 重复，添加 alive 标志位防止卸载后状态更新
- **Bug 修复**:
  - 设置页 useEffect 中的 IIFE 异步操作在组件卸载后仍可能触发 setState，导致 React 警告

## 2026-07-18 — 🐛 fix(admin): 修复用户管理认证缺陷并清理冗余

- **类型**: fix
- **分支**: main
- **文件数**: 17 (+232/-163)
- **决策**:
  - GetSystemConfig 错误时透出真实错误而非默认值：旧实现返回固定默认配置会掩盖配置错误，导致前端在配置加载失败时产生误判。改为透传错误让调用方感知
  - RecoveryCodes 实现 SQL Scanner/Value 接口：GORM 对 []string JSONB 字段的 nil/空数组处理与 PostgreSQL 驱动不一致，导致恢复码写入读回后出现序列化异常。通过自定义类型显式控制 JSONB 的 marshal/unmarshal
  - 导入字段全部 required：旧实现允许空值导致批量导入 1001/1005 故障。前端解析表格后若字段为空，后端不再默认填充
  - 移除 excelize 依赖和 import-template 路由：导入模板功能已由前端（XLSX.js）处理，后端不再需要 Excel 生成能力
  - PasskeyPanel 三态处理：enabled 改为 boolean|undefined，避免安全策略配置加载中时错误显示 Passkey 已启用
  - profile 页刷新用户信息：换绑邮箱后前端主动刷新用户信息以保持状态同步
- **Bug 修复**:
  - GetSystemConfig 出错时总是返回默认值(true/true/[])，前端无法感知配置加载失败
  - RecoveryCodes GORM JSONB 字段 nil 与 [] 混用导致序列化异常
  - 批量导入字段可为空导致 1001/1005 故障
  - PasskeyPanel 用 !enabled 判断禁用状态，但 undefined 也会被判定为禁用
  - profile 页换绑邮箱后不刷新用户信息导致页面状态与后端不同步
- **测试**: 新增 RecoveryCodes Scan/Value 单元测试(user_test.go)，删除废弃的 import/export 测试(user_import_export_test.go)

## 2026-07-18 — fix: 修复认证流程缺陷并清理前端冗余

- **类型**: fix
- **分支**: main
- **文件数**: 0 (+0/-0)
- **决策**:
  - ForceEmailChange 种子数据默认改为 true 以触发首次换绑邮箱流程
  - 用户管理列表可见自己：放宽 maxViewRank 过滤条件
  - 后端导入导出补充校验逻辑
  - passkey 服务优化
  - 移除前端调试日志（OAuthSection, RegisterForm）
  - 精简 HeaderBar 和 SideMenu 布局代码
- **测试**: ['user_profile_validation_test.go（新增）']

## 2026-07-18 — feat: 实现 RBAC 权限模型、认证重构与用户管理增强

- **类型**: feat
- **分支**: main
- **文件数**: 0 (+0/-0)
- **决策**:
  - 采用 RBAC (Casbin) 替代基于 Rank 的角色授权体系，移除 user.rank 参与业务决策
  - 用户导入改为前端浏览器解析 xlsx/csv（使用 xlsx 库），后端仅接收 JSON 数组，移除后端表格文件解析
  - 受保护接口统一按资源分组（/auth, /rbac, /settings, /users），使用 RequireRBAC 绑定 Casbin resource/action
  - 所有受保护接口必须显式提供 Casbin resource/action，缺失时拒绝响应
  - 移除用户视图 isAdmin 字段及管理界面中 admin 角色选项
  - 同级用户可互相编辑但受后端 RBAC 边界约束
  - 移除 migrations/001_init.sql，数据库初始化改为 GORM AutoMigrate
  - 种子数据 ForceEmailChange 改为 true，存量 DB 幂等回填
  - 管理界面列表可见自己：maxViewRank 由 rank<actor 改为 rank<=actor
  - 内置管理员 isDefaultAdmin 免验证码豁免已在 auth_service 和 SecurityRequiredPage 中实现
- **测试**: ['security_actions_test', 'user_admin_service_test', 'user_import_export_test', 'rbac_test', 'password_validation_test', 'hash_test']
