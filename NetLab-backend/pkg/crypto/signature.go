package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// VerifyHMACSHA256Hex 校验以十六进制编码的 HMAC-SHA256 签名。
//
// 预共享密钥认证路径对签名和密钥使用十六进制编码。
func VerifyHMACSHA256Hex(key []byte, message, signatureHex string) (bool, error) {
	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	expected := mac.Sum(nil)

	return hmac.Equal(expected, sig), nil
}

// BuildSignPayloadWithTimestamp 按照前端约定创建预共享密钥
// 签名负载：
//
//	{requestId}{salt}{timestamp}{bodyHash}
//
// 其中 bodyHash 是（明文）请求体 JSON 的 SHA-256 十六进制值。
func BuildSignPayloadWithTimestamp(requestID, salt, timestamp, bodyJSON string) string {
	bodyHash := SHA256Hex(bodyJSON)
	return requestID + salt + timestamp + bodyHash
}
