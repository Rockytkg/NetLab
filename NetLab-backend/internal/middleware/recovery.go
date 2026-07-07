package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"netlab-backend/pkg/response"
)

// Recovery 返回一个使用 zap 记录日志的自定义 panic 恢复中间件。
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())

				logger.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("stack", stack),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("request_id", GetRequestID(c)),
				)

				// 不要将堆栈信息泄露给客户端
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.ApiResponse{
					Code:    http.StatusInternalServerError,
					Message: fmt.Sprintf("internal server error"),
				})
			}
		}()

		c.Next()
	}
}
