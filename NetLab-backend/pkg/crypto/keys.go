package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
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

// GenerateRandomBase64URL 返回紧凑的 URL 安全随机标识符。
func GenerateRandomBase64URL(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("invalid random size: %d", size)
	}
	key := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("generate random id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

// GenerateNumericCode 返回定长零填充的加密安全随机数字码。
func GenerateNumericCode(digits int) (string, error) {
	if digits <= 0 || digits > 18 {
		return "", fmt.Errorf("invalid code length: %d", digits)
	}
	max := big.NewInt(1)
	for i := 0; i < digits; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate numeric code: %w", err)
	}
	return fmt.Sprintf("%0*d", digits, n), nil
}
