package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword 创建密码的 bcrypt 哈希。
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword 将 bcrypt 哈希与密码进行比对。
func VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SHA256Hex 返回输入的 SHA-256 哈希，以十六进制字符串表示。
func SHA256Hex(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

// SHA256Base64URL 返回紧凑的 URL 安全无填充 SHA-256 摘要。
func SHA256Base64URL(input string) string {
	h := sha256.Sum256([]byte(input))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
