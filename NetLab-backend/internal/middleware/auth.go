package middleware

import (
	"strings"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"

	"netlab-backend/pkg/apperrors"
	pkgjwt "netlab-backend/pkg/jwt"
	"netlab-backend/pkg/response"
)

const (
	// ContextKeyUserID 存储已认证用户的 ID。
	ContextKeyUserID = "user_id"
	// ContextKeyUsername 存储已认证用户的用户名。
	ContextKeyUsername = "username"
	// ContextKeyUserRole 存储已认证用户的角色。
	ContextKeyUserRole = "user_role"
	// ContextKeySessionID 存储当前登录会话 ID。
	ContextKeySessionID = "session_id"
	// ContextKeyResource 存储当前请求对应的 RBAC 资源名。
	ContextKeyResource = "rbac_resource"
	// ContextKeyAction 存储当前请求对应的 RBAC 操作。
	ContextKeyAction = "rbac_action"
)

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

		// Redis-backed session state is part of authentication. Fail closed on
		// Redis errors so revoked or replaced sessions cannot slip through.
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
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyUserRole, claims.Role)
		c.Set(ContextKeySessionID, claims.SessionID)

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

// Authorize uses Casbin to enforce an explicit RBAC resource/action decision.
func Authorize(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if enforcer == nil {
			response.Error(c, apperrors.ErrOperationDenied)
			return
		}

		role := GetUserRole(c)
		resource := GetResource(c)
		action := GetAction(c)
		if resource == "" || action == "" {
			response.Error(c, apperrors.ErrOperationDenied)
			return
		}

		allowed, err := enforcer.Enforce(role, resource, action)
		if err == nil && allowed {
			c.Next()
			return
		}
		response.Error(c, apperrors.ErrOperationDenied)
	}
}

// RequireRBAC 标注并立即执行当前路由的 RBAC 权限检查。
// 该中间件必须同时完成标注和校验，避免全局中间件早于路由中间件执行。
func RequireRBAC(enforcer *casbin.Enforcer, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ContextKeyResource, resource)
		c.Set(ContextKeyAction, action)
		Authorize(enforcer)(c)
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

// GetUsername 从 context 中获取已认证的用户名。
func GetUsername(c *gin.Context) string {
	if name, exists := c.Get(ContextKeyUsername); exists {
		if s, ok := name.(string); ok {
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

// GetSessionID 从 context 中获取当前登录会话 ID。
func GetSessionID(c *gin.Context) string {
	if sid, exists := c.Get(ContextKeySessionID); exists {
		if s, ok := sid.(string); ok {
			return s
		}
	}
	return ""
}

// GetResource 从 context 中获取当前请求的 RBAC 资源名。
func GetResource(c *gin.Context) string {
	if v, exists := c.Get(ContextKeyResource); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetAction 从 context 中获取当前请求的 RBAC 操作。
func GetAction(c *gin.Context) string {
	if v, exists := c.Get(ContextKeyAction); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

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
