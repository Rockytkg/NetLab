// Package contextkeys 定义了在中间件、处理器和响应辅助函数之间共享的
// Gin 上下文 key 及访问器函数，避免产生循环导入。
package contextkeys

import "github.com/gin-gonic/gin"

const (
	// Locale 是用于存储已解析请求区域设置的 Gin 上下文 key。
	Locale = "locale"
)

// GetLocale 从 Gin 上下文中获取已解析的区域设置。
// 如果未设置，则回退为 "en-US"。
func GetLocale(c *gin.Context) string {
	if locale, exists := c.Get(Locale); exists {
		if s, ok := locale.(string); ok {
			return s
		}
	}
	return "en-US"
}
