package middleware

import (
	"strings"
	"time"

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
	// ContextKeyUserRoles 存储已认证用户的角色。
	ContextKeyUserRoles = "user_roles"
	// ContextKeyJTI 存储当前 token 的 JWT ID。
	ContextKeyJTI = "jti"
	// ContextKeyTokenExp 存储 token 的过期时间。
	ContextKeyTokenExp = "token_exp"
)

// AuthConfig 配置认证中间件的行为。
type AuthConfig struct {
	Required bool // 若为 false，当存在 token 时附加用户信息，但不中断请求
}

// Auth 创建一个 JWT 认证中间件。
// 它会校验 Bearer token，检查 Redis 黑名单，并将用户信息注入到 context 中。
func Auth(jwtManager *pkgjwt.Manager, blacklist pkgjwt.BlacklistManager, cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c)

		if tokenString == "" {
			if cfg.Required {
				response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "missing authorization header"))
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

		// 检查黑名单
		if blacklist != nil {
			blacklisted, err := blacklist.IsBlacklisted(c.Request.Context(), claims.ID)
			if err != nil {
				// 发生 Redis 错误时，放行请求（黑名单检查采用失败放行策略）
				// 在生产环境中，你可能希望采用失败拒绝策略
			} else if blacklisted {
				response.Error(c, apperrors.ErrTokenExpired)
				return
			}
		}

		// 将用户信息注入到 context 中
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyUserRoles, claims.Roles)
		c.Set(ContextKeyJTI, claims.ID)
		if claims.ExpiresAt != nil {
			c.Set(ContextKeyTokenExp, claims.ExpiresAt.Time)
		}

		c.Next()
	}
}

// RequireAuth 是要求提供有效 token 的认证中间件的简写形式。
func RequireAuth(jwtManager *pkgjwt.Manager, blacklist pkgjwt.BlacklistManager) gin.HandlerFunc {
	return Auth(jwtManager, blacklist, AuthConfig{Required: true})
}

// OptionalAuth 是当存在 token 时附加用户信息的认证中间件的简写形式。
func OptionalAuth(jwtManager *pkgjwt.Manager, blacklist pkgjwt.BlacklistManager) gin.HandlerFunc {
	return Auth(jwtManager, blacklist, AuthConfig{Required: false})
}

// RequireRoles 创建一个中间件，检查已认证用户是否至少拥有所需角色之一。
// 必须在 RequireAuth 之后使用。
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get(ContextKeyUserRoles)
		if !exists {
			response.Error(c, apperrors.ErrOperationDenied)
			return
		}

		allowedRoles := make(map[string]bool, len(roles))
		for _, r := range roles {
			allowedRoles[r] = true
		}

		roleList, ok := userRoles.([]string)
		if !ok {
			response.Error(c, apperrors.ErrOperationDenied)
			return
		}

		for _, r := range roleList {
			if allowedRoles[r] {
				c.Next()
				return
			}
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

// GetUsername 从 context 中获取已认证的用户名。
func GetUsername(c *gin.Context) string {
	if name, exists := c.Get(ContextKeyUsername); exists {
		if s, ok := name.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserRoles 从 context 中获取已认证用户的角色。
func GetUserRoles(c *gin.Context) []string {
	if roles, exists := c.Get(ContextKeyUserRoles); exists {
		if r, ok := roles.([]string); ok {
			return r
		}
	}
	return nil
}

// GetJTI 从 context 中获取 JWT ID。
func GetJTI(c *gin.Context) string {
	if jti, exists := c.Get(ContextKeyJTI); exists {
		if s, ok := jti.(string); ok {
			return s
		}
	}
	return ""
}

// GetTokenExp 从 context 中获取 token 的过期时间。
func GetTokenExp(c *gin.Context) time.Time {
	if exp, exists := c.Get(ContextKeyTokenExp); exists {
		if t, ok := exp.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
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
