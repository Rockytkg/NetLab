package sysconfig

import "net/url"

// ProviderMeta 描述一个受支持的 OAuth 提供商的静态元数据
// （显示名称、图标、品牌色、授权端点与 scope）。凭据本身是可配置的，
// 但这些元数据是内置的，不随部署变化。
type ProviderMeta struct {
	ID        string
	Name      string
	Icon      string
	Color     string
	AuthBase  string // 授权端点基础 URL
	Scope     string // scope 参数值（已 URL 编码）
	UsesAppID bool   // 微信使用 appid 而非 client_id 参数
}

// providerRegistry 是受支持 OAuth 提供商的规范列表（保持稳定顺序）。
var providerRegistry = []ProviderMeta{
	{ID: "github", Name: "GitHub", Icon: "github", Color: "#24292f",
		AuthBase: "https://github.com/login/oauth/authorize", Scope: "read:user+user:email"},
	{ID: "google", Name: "Google", Icon: "google", Color: "#4285f4",
		AuthBase: "https://accounts.google.com/o/oauth2/v2/auth", Scope: "openid+profile+email"},
	{ID: "linuxdo", Name: "LinuxDO", Icon: "linuxdo", Color: "#4a90d9",
		AuthBase: "https://connect.linux.do/oauth2/authorize", Scope: "openid+profile+email"},
	{ID: "wechat", Name: "WeChat", Icon: "wechat", Color: "#07c160",
		AuthBase: "https://open.weixin.qq.com/connect/qrconnect", Scope: "snsapi_login", UsesAppID: true},
	{ID: "qq", Name: "QQ", Icon: "qq", Color: "#12b7f5",
		AuthBase: "https://graph.qq.com/oauth2.0/authorize", Scope: "get_user_info"},
}

// providerMeta 按 ID 查找提供商元数据。
func providerMeta(id string) (ProviderMeta, bool) {
	for _, m := range providerRegistry {
		if m.ID == id {
			return m, true
		}
	}
	return ProviderMeta{}, false
}

// ProviderIDs 返回所有受支持提供商 ID 的有序列表。
func ProviderIDs() []string {
	ids := make([]string, len(providerRegistry))
	for i, m := range providerRegistry {
		ids[i] = m.ID
	}
	return ids
}

// ProviderName 返回提供商的显示名称，未知 ID 时回退为其 ID。
func ProviderName(id string) string {
	if m, ok := providerMeta(id); ok {
		return m.Name
	}
	return id
}

// buildAuthURL 根据提供商元数据与已配置凭据构建授权 URL。
// 用户提供的 client_id 与 redirect_uri 经 URL 转义，避免特殊字符
// 破坏查询字符串结构。scope 是内置常量，保持原样传递。
func buildAuthURL(meta ProviderMeta, cfg ProviderSettings) string {
	if cfg.ClientID == "" {
		return ""
	}
	clientID := url.QueryEscape(cfg.ClientID)
	redirect := ""
	if cfg.RedirectURL != "" {
		redirect = "&redirect_uri=" + url.QueryEscape(cfg.RedirectURL)
	}

	// 微信采用特殊的 URL 格式（appid + #wechat_redirect 锚点）。
	if meta.ID == "wechat" {
		return meta.AuthBase + "?appid=" + clientID +
			"&response_type=code&scope=" + meta.Scope + redirect + "#wechat_redirect"
	}

	// GitHub 授权 URL 省略 response_type（默认为 code）。
	responseType := "&response_type=code"
	if meta.ID == "github" {
		responseType = ""
	}
	return meta.AuthBase + "?client_id=" + clientID +
		responseType + "&scope=" + meta.Scope + redirect
}
