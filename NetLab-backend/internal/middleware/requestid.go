package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// HeaderRequestID 是 request ID 的请求头 key。
	HeaderRequestID = "X-Request-Id"
	// ContextKeyRequestID 是 request ID 的 context key。
	ContextKeyRequestID = "request_id"
)

// RequestID 注入或传递 X-Request-Id 请求头。
// 若客户端发送了 X-Request-Id，则复用它；否则生成一个新的 UUID。
// request ID 会同时设置到响应头和 Gin context 中。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(ContextKeyRequestID, requestID)
		c.Header(HeaderRequestID, requestID)
		c.Next()
	}
}

// GetRequestID 从 Gin context 中获取 request ID。
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(ContextKeyRequestID); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}
