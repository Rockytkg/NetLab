# Commit Decision History

> 此文件是 `commits.jsonl` 的人类可读视图，可由工具重生成。
> Canonical store: `commits.jsonl` (JSONL, append-only)

| Date | Context-Id | Commit | Summary | Decisions | Tests |
|------|-----------|--------|---------|-----------|-------|
| 2026-07-18 | 7c5d66da | feat | feat: 实现 RBAC 权限模型、认证重构与用户管理增强 | 采用 RBAC (Casbin) 替代基于 Rank 的角色授权体系，移除 user.rank 参与业务决策<br/>用户导入改为前端浏览器解析 xlsx/csv（使用 xlsx 库），后端仅接收 JSON 数组，移除后端表格文件解析<br/>受保护接口统一按资源分组（/auth, /rbac, /settings, /users），使用 RequireRBAC 绑定 Casbin resource/action<br/>所有受保护接口必须显式提供 Casbin resource/action，缺失时拒绝响应<br/>移除用户视图 isAdmin 字段及管理界面中 admin 角色选项<br/>同级用户可互相编辑但受后端 RBAC 边界约束<br/>移除 migrations/001_init.sql，数据库初始化改为 GORM AutoMigrate<br/>种子数据 ForceEmailChange 改为 true，存量 DB 幂等回填<br/>管理界面列表可见自己：maxViewRank 由 rank<actor 改为 rank<=actor<br/>内置管理员 isDefaultAdmin 免验证码豁免已在 auth_service 和 SecurityRequiredPage 中实现 | security_actions_test<br/>user_admin_service_test<br/>user_import_export_test<br/>rbac_test<br/>password_validation_test<br/>hash_test |
| 2026-07-18 | e18dc695 | fix | fix: 修复认证流程缺陷并清理前端冗余 | ForceEmailChange 种子数据默认改为 true 以触发首次换绑邮箱流程<br/>用户管理列表可见自己：放宽 maxViewRank 过滤条件<br/>后端导入导出补充校验逻辑<br/>passkey 服务优化<br/>移除前端调试日志（OAuthSection, RegisterForm）<br/>精简 HeaderBar 和 SideMenu 布局代码 | user_profile_validation_test.go（新增） |
| 2026-07-18 | aac2fc3c | fix | 🐛 fix(admin): 修复用户管理认证缺陷并清理冗余 | GetSystemConfig 错误时透出真实错误而非默认值<br/>RecoveryCodes 实现 SQL Scanner/Value 接口修复 JSONB 序列化<br/>导入字段全部 required 修复批量导入 1001/1005<br/>移除 excelize 依赖和 import-template 路由<br/>PasskeyPanel 三态处理避免误判<br/>profile 页刷新用户信息保持同步 | RecoveryCodes Scan/Value 单元测试 |
