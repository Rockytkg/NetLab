package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"netlab-backend/pkg/apperrors"
	pkgjwt "netlab-backend/pkg/jwt"
	"netlab-backend/pkg/response"
)

const (
	// ContextKeyUserID 存储已认证用户的 ID。
	ContextKeyUserID = "user_id"
	// ContextKeyUserRole 存储已认证用户的主角色 ID。
	ContextKeyUserRole = "user_role"
)

// Authorizer 是路由授权所需的最小接口，具体实现由 RBAC 服务提供。
type Authorizer interface {
	Allow(context.Context, string, string) bool
}

// AuthConfig 配置认证中间件的行为。
type AuthConfig struct {
	Required bool // 若为 false，当存在 token 时附加用户信息，但不中断请求
}

// Auth 创建一个 JWT 认证中间件。
// 它会校验 Bearer token，检查 Redis 会话，并将用户信息注入到 context 中。
func Auth(jwtManager *pkgjwt.Manager, tokenStore pkgjwt.SessionValidator, cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c)

		if tokenString == "" {
			if cfg.Required {
				response.Error(c, apperrors.New(apperrors.ErrCodeUnauthorized, "missing authorization header"))
				return
			}
			c.Next()
			return
		}

		claims, err := jwtManager.ParseAccessToken(tokenString)
		if err != nil {
			if cfg.Required {
				response.Error(c, apperrors.ErrTokenExpired)
				return
			}
			c.Next()
			return
		}

		// Redis 中的会话状态是认证的一部分。Redis 出错时按失败关闭（fail closed）
		// 处理，确保已吊销或被替换的会话不会漏过校验。
		if tokenStore != nil {
			active, err := tokenStore.IsSessionActive(c.Request.Context(), claims.UserID, claims.SessionID)
			if err != nil || !active {
				if !cfg.Required {
					c.Next()
					return
				}
				response.Error(c, apperrors.ErrTokenExpired)
				return
			}
		}

		// 将用户信息注入到 context 中
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserRole, claims.Role)

		c.Next()
	}
}

// RequireAuth 是要求提供有效 token 的认证中间件的简写形式。
func RequireAuth(jwtManager *pkgjwt.Manager, tokenStore pkgjwt.SessionValidator) gin.HandlerFunc {
	return Auth(jwtManager, tokenStore, AuthConfig{Required: true})
}

// OptionalAuth 是当存在 token 时附加用户信息的认证中间件的简写形式。
func OptionalAuth(jwtManager *pkgjwt.Manager, tokenStore pkgjwt.SessionValidator) gin.HandlerFunc {
	return Auth(jwtManager, tokenStore, AuthConfig{Required: false})
}

// RequirePermission 创建一个 RBAC 鉴权中间件，校验稳定的
// resource.action 权限码；无授权器或权限码为空时拒绝请求。
func RequirePermission(authorizer Authorizer, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authorizer == nil || permission == "" {
			response.Error(c, apperrors.ErrOperationDenied)
			return
		}
		if authorizer.Allow(c.Request.Context(), GetUserRole(c), permission) {
			c.Next()
			return
		}
		response.Error(c, apperrors.ErrOperationDenied)
	}
}

// GetUserID 从 context 中获取已认证的用户 ID。
func GetUserID(c *gin.Context) string {
	if id, exists := c.Get(ContextKeyUserID); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserRole 从 context 中获取已认证用户的角色。
func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get(ContextKeyUserRole); exists {
		if r, ok := role.(string); ok {
			return r
		}
	}
	return ""
}

// extractBearerToken 从 Authorization 请求头中提取 Bearer token；
// 头缺失或格式不合法时返回空字符串。
func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}
