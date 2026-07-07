package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// GenerateRandomKey 生成一个随机的 32 字节密钥，用于会话级签名。
func GenerateRandomKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate random key: %w", err)
	}
	return key, nil
}

// GenerateRandomKeyHex 生成一个随机的 32 字节密钥，并以
// 小写十六进制字符串（64 个字符）返回。整个代码库统一使用
// 十六进制表示密钥和盐（取代了此前的 base64 编码）。
func GenerateRandomKeyHex() (string, error) {
	key, err := GenerateRandomKey()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}
