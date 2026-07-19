// Package radiusd 提供 RADIUS 认证计费核心运行时。
//
// 本包移植自 github.com/talkincode/toughradius（MIT License）的 radiusd 核心，
// 已适配 NetLab 的配置、模型与依赖注入体系。
package radiusd

// NA 是空值占位符，用于会话/记账记录中无法获取的字符串字段。
const NA = "N/A"

// IfEmptyStr 在 s 为空时返回 def，否则返回 s。
func IfEmptyStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
