package middleware

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
	"netlab-backend/pkg/response"
)

const (
	// HeaderSignature 是 HMAC 签名请求头。
	HeaderSignature = "X-Signature"
	// HeaderTimestamp 是请求时间戳请求头（RFC 3339）。
	HeaderTimestamp = "X-Timestamp"
	// MaxTimestampSkew 是允许的最大时钟偏差（5 分钟）。
	MaxTimestampSkew = 5 * time.Minute
)

// SignatureConfig 配置签名验证中间件。
type SignatureConfig struct {
	// Required 指示缺失/无效的签名是否会中断请求。
	Required bool
	// SignatureKey 是预共享的 HMAC 密钥（来自 AUTH_SIGNATURE_KEY 环境变量）。
	SignatureKey []byte
	// SignatureSalt 是预共享的 HMAC 盐值（来自 AUTH_SIGNATURE_SALT 环境变量）。
	SignatureSalt string
}

// SessionKeyLookup 解析指定用户的会话签名密钥。
// 返回原始的密钥字节及可能的错误。若未找到密钥，
// 则返回 (nil, nil) —— 此时中间件将跳过验证。
type SessionKeyLookup func(ctx context.Context, userID string) ([]byte, error)

// Signature 返回一个验证 X-Signature 请求头的中间件。
// 对于公开的认证端点，签名负载为：
//
//	X-Request-Id + salt + X-Timestamp + SHA256(body JSON)
//
// 前端使用预共享密钥通过 HMAC-SHA256（十六进制编码）计算该签名。
// 时间戳会在服务端进行校验（±5 分钟窗口）以防止重放攻击。
func Signature(cfg SignatureConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		sigHeader := c.GetHeader(HeaderSignature)
		tsHeader := c.GetHeader(HeaderTimestamp)

		if sigHeader == "" {
			if cfg.Required {
				response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "missing signature header"))
				return
			}
			c.Next()
			return
		}

		// 校验时间戳以防止重放攻击
		if tsHeader == "" {
			if cfg.Required {
				response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "missing timestamp header"))
				return
			}
			c.Next()
			return
		}

		ts, err := time.Parse(time.RFC3339, tsHeader)
		if err != nil {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid timestamp format"))
			return
		}

		skew := time.Since(ts)
		if skew < 0 {
			skew = -skew
		}
		if skew > MaxTimestampSkew {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "request timestamp expired"))
			return
		}

		// 读取并恢复请求体以用于签名计算
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "failed to read request body"))
			return
		}
		c.Request.Body.Close()
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		bodyStr := string(body)
		if bodyStr == "" {
			bodyStr = "{}"
		}

		// 构建负载并验证签名
		// 负载： X-Request-Id + salt + X-Timestamp + SHA256(body)
		// 前端为预共享密钥认证发送十六进制编码的 HMAC-SHA256。
		requestID := GetRequestID(c)
		payload := crypto.BuildSignPayloadWithTimestamp(requestID, cfg.SignatureSalt, tsHeader, bodyStr)

		valid, err := crypto.VerifyHMACSHA256Hex(cfg.SignatureKey, payload, sigHeader)
		if err != nil || !valid {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid signature"))
			return
		}

		c.Next()
	}
}

// SessionSignature 返回一个中间件，用于为已认证端点验证
// 会话级别的请求签名。
//
// 签名密钥通过 lookup 函数按请求解析，该函数通常
// 从 Redis 读取（以用户 ID 为键）。若某用户没有可用的
// 签名密钥，则跳过签名验证。
//
// 签名负载格式：
//
//	METHOD\npath\nX-Timestamp\nSHA256(body)
//
// 前端使用认证期间建立的会话签名密钥
//（在 login/refresh 响应中返回）来计算该签名。
func SessionSignature(lookup SessionKeyLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 context 中获取用户 ID（由 RequireAuth 中间件设置）
		userID := GetUserID(c)
		if userID == "" {
			c.Next()
			return
		}

		// 查找会话签名密钥
		signingKey, err := lookup(c.Request.Context(), userID)
		if err != nil || signingKey == nil {
			// 无可用的会话密钥 —— 跳过签名强制校验
			c.Next()
			return
		}

		sigHeader := c.GetHeader(HeaderSignature)
		tsHeader := c.GetHeader(HeaderTimestamp)

		// 两个请求头都必须存在才能进行会话签名强制校验
		if sigHeader == "" || tsHeader == "" {
			c.Next()
			return
		}

		// 校验时间戳以防止重放
		ts, err := time.Parse(time.RFC3339, tsHeader)
		if err != nil {
			c.Next()
			return
		}
		skew := time.Since(ts)
		if skew < 0 {
			skew = -skew
		}
		if skew > MaxTimestampSkew {
			c.Next()
			return
		}

		// 读取请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}
		c.Request.Body.Close()
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		bodyStr := string(body)
		if bodyStr == "" {
			bodyStr = "{}"
		}

		payload := crypto.BuildSessionSignPayload(c.Request.Method, c.Request.URL.Path, tsHeader, bodyStr)

		valid, err := crypto.VerifyHMACSHA256Hex(signingKey, payload, sigHeader)
		if err != nil || !valid {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid session signature"))
			return
		}

		c.Next()
	}
}
