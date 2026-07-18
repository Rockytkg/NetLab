package router

// 认证路由契约集中定义于此，避免路径与权限
// 在 handler、客户端与路由注册之间重复定义。
const authRoutePrefix = "/auth"

func authRoute(path string) string {
	return authRoutePrefix + path
}
