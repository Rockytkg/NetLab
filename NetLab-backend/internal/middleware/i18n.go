package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"netlab-backend/internal/contextkeys"
	"netlab-backend/pkg/i18n"
)

const (
	// HeaderAcceptLanguage 是用于语言协商的标准 HTTP 请求头。
	HeaderAcceptLanguage = "Accept-Language"
	// HeaderUserLanguage 是一个自定义请求头，用于携带用户明确指定的
	// 语言偏好，其优先级高于 Accept-Language。
	HeaderUserLanguage = "X-User-Language"
)

// SupportedLocales 是服务器接受的 locale 集合。
var SupportedLocales = map[string]bool{
	"zh-CN": true,
	"en-US": true,
}

// I18N 从请求头中解析用户的 locale，并将其存入 Gin context。
// 解析优先级：
//
//  1. X-User-Language —— 用户明确指定的偏好（最高优先级）
//  2. Accept-Language —— 浏览器/操作系统的语言协商
//  3. 默认值 —— "en-US"
func I18N() gin.HandlerFunc {
	return func(c *gin.Context) {
		userLang := c.GetHeader(HeaderUserLanguage)
		acceptLang := c.GetHeader(HeaderAcceptLanguage)
		locale := resolveLocale(userLang, acceptLang)
		c.Set(contextkeys.Locale, locale)
		c.Next()
	}
}

// resolveLocale 应用优先级链：userLang > acceptLang > 默认值。
func resolveLocale(userLang, acceptLang string) string {
	// 优先级 1：用户明确指定的偏好（X-User-Language）
	if userLang != "" {
		if l := matchLocale(userLang); l != "" {
			return l
		}
	}

	// 优先级 2：浏览器/操作系统的语言（Accept-Language）
	if acceptLang != "" {
		if l := matchLocale(acceptLang); l != "" {
			return l
		}
	}

	// 优先级 3：默认回退值
	return i18n.DefaultLocale
}

// matchLocale 尝试将原始的请求头值匹配到受支持的 locale。
// 它首先尝试精确匹配，然后回退到前缀匹配
// （例如 "zh" → "zh-CN"，"en" → "en-US"）。
func matchLocale(header string) string {
	// 提取第一个语言标签（位于逗号、分号或空格之前）
	locale := extractPrimaryTag(header)
	if locale == "" {
		return ""
	}

	// 精确匹配
	if SupportedLocales[locale] {
		return locale
	}

	// 前缀匹配： "zh" → "zh-CN"，"en" → "en-US"
	if len(locale) >= 2 {
		prefix := strings.ToLower(locale[:2])
		for supported := range SupportedLocales {
			if strings.HasPrefix(strings.ToLower(supported), prefix) {
				return supported
			}
		}
	}

	return ""
}

// extractPrimaryTag 从 Accept-Language 请求头值中返回第一个语言标签，
// 在逗号、分号或空白字符处停止。
func extractPrimaryTag(header string) string {
	for i, ch := range header {
		if ch == ',' || ch == ';' || ch == ' ' {
			return header[:i]
		}
	}
	return header
}
