package auth

import (
	"encoding/hex"

	"netlab-backend/config"
)

// CryptoService 管理公开认证接口所使用的预共享签名 key/salt。
//
// 注意：此处不涉及请求/响应体加密。公开认证接口发送明文请求体
//（机密性由 HTTPS/TLS 保证），仅通过下方预共享 key 校验的
// HMAC 签名进行保护。
//
// 签名 key 以十六进制编码字符串（AUTH_SIGNATURE_KEY）配置，
// 必须与前端的 VITE_AUTH_SIGNATURE_KEY 保持一致。
type CryptoService struct {
	signatureKey  []byte
	signatureSalt string
}

// NewCryptoService 根据配置创建一个新的 CryptoService。
func NewCryptoService(cfg config.AuthConfig) (*CryptoService, error) {
	// 解码十六进制编码的签名 key。
	signatureKey, err := hex.DecodeString(cfg.SignatureKey)
	if err != nil {
		return nil, err
	}

	return &CryptoService{
		signatureKey:  signatureKey,
		signatureSalt: cfg.SignatureSalt,
	}, nil
}

// SignatureKey 返回预共享签名 key。
func (s *CryptoService) SignatureKey() []byte {
	return s.signatureKey
}

// SignatureSalt 返回预共享签名 salt。
func (s *CryptoService) SignatureSalt() string {
	return s.signatureSalt
}
