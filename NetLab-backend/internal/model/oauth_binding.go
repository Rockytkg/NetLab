package model

import "time"

// OAuthProviderInfo 是返回给前端的 OAuth 绑定信息。
type OAuthProviderInfo struct {
	Provider  string    `json:"provider"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
