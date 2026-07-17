# Commit Decision History

> 此文件是 `commits.jsonl` 的人类可读视图，可由工具重生成。
> Canonical store: `commits.jsonl` (JSONL, append-only)

| Date | Context-Id | Commit | Summary | Decisions | Changes | Breaking |
|------|-----------|--------|---------|-----------|---------|----------|
| 2026-07-18 | 7c5d66da | feat | feat: 实现 RBAC 权限模型、认证重构与用户管理增强 | 采用 RBAC (Casbin) 替代基于 Rank 的角色授权体系，移除 user.rank 参与业务决策<br/>用户导入改为前端浏览器解析 xlsx/csv（使用 xlsx 库），后端仅接收 JSON 数组，移除后端表格文件解析<br/>受保护接口统一按资源分组（/auth, /rbac, /settings, /users），使用 RequireRBAC 绑定 Casbin resource/action<br/>所有受保护接口必须显式提供 Casbin resource/action，缺失时拒绝响应<br/>移除用户视图 isAdmin 字段及管理界面中 admin 角色选项<br/>同级用户可互相编辑但受后端 RBAC 边界约束<br/>移除 migrations/001_init.sql，数据库初始化改为 GORM AutoMigrate<br/>种子数据 ForceEmailChange 改为 true，存量 DB 幂等回填<br/>管理界面列表可见自己：maxViewRank 由 rank<actor 改为 rank<=actor<br/>内置管理员 isDefaultAdmin 免验证码豁免已在 auth_service 和 SecurityRequiredPage 中实现 | ➕ RBAC 模型/服务/DTO/Handler/路由<br/>➕ password_service + 强度验证<br/>➕ user_import_export<br/>➕ verification_service<br/>➕ JWT signer.go<br/>➕ 前端 Can/usePermission/rbac<br/>➕ 前端 auth-normalize/password-strength<br/>➕ .context/ 上下文<br/>➕ .ccg/ 任务追踪<br/>➖ migrations/001_init.sql<br/>➖ 前端 router/index.tsx 旧路由<br/>➖ types/operations.ts<br/>✏️ auth_handler/admin_handler 重构<br/>✏️ auth_service/token_service 重构<br/>✏️ 用户管理页面重写<br/>✏️ 设置页面调整<br/>✏️ authStore/operationsStore<br/>✏️ middleware auth/signature<br/>✏️ i18n 同步 | 移除 migrations 目录，改为 GORM AutoMigrate |
