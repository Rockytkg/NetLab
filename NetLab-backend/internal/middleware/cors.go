package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"netlab-backend/config"
)

// CORS 返回一个根据应用配置构建的 CORS 中间件。
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		allowedOrigins[origin] = true
	}

	allowMethods := strings.Join(cfg.AllowedMethods, ",")
	allowHeaders := strings.Join(cfg.AllowedHeaders, ",")

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 仅当 origin 存在且被允许时才设置 CORS 响应头
		if origin == "" {
			c.Next()
			return
		}

		originAllowed := false
		if _, ok := allowedOrigins["*"]; ok {
			originAllowed = true
			c.Header("Access-Control-Allow-Origin", "*")
		} else if _, ok := allowedOrigins[origin]; ok {
			originAllowed = true
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}

		if !originAllowed {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Methods", allowMethods)
		c.Header("Access-Control-Allow-Headers", allowHeaders)
		c.Header("Access-Control-Expose-Headers", "X-Request-Id")
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
