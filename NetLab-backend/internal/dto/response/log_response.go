package response

import "time"

// LoginLogItem 是 GET /logs/logins 的单条登录日志。
type LoginLogItem struct {
	ID          uint64    `json:"id"`
	Username    string    `json:"username"`
	LoginType   string    `json:"loginType"`
	Status      string    `json:"status"`
	IP          string    `json:"ip"`
	UserAgent   string    `json:"userAgent"`
	Fingerprint string    `json:"fingerprint"`
	OS          string    `json:"os"`
	Browser     string    `json:"browser"`
	Location    string    `json:"location"`
	CreatedAt   time.Time `json:"createdAt"`
}

// LoginLogListResult 是 GET /logs/logins 的分页响应。
type LoginLogListResult struct {
	Items []LoginLogItem `json:"items"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}
