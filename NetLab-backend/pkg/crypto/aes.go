package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// encryptedPrefix 标记一个字符串是经过 AES-GCM 加密的密文。
// 存储在数据库中的密文形如 "enc:v1:<base64(nonce|ciphertext)>"，
// 版本号便于将来在不破坏既有数据的前提下演进加密方案。
const encryptedPrefix = "enc:v1:"

// ErrDecryptFailed 表示密文无法被解密（密钥错误或数据损坏）。
var ErrDecryptFailed = errors.New("decrypt failed")

// AESCipher 使用 AES-256-GCM 对敏感配置值进行认证加密。
//
// 主密钥通过 SHA-256 从任意长度的字符串派生为固定的 32 字节，
// 因此调用方无需关心原始密钥的长度。GCM 同时提供机密性与完整性，
// 每次加密都会生成随机 nonce 并前置到密文中。
type AESCipher struct {
	gcm cipher.AEAD
}

// NewAESCipher 根据主密钥字符串创建一个 AESCipher。
// 密钥经 SHA-256 派生为 32 字节（AES-256）。
func NewAESCipher(masterKey string) (*AESCipher, error) {
	if masterKey == "" {
		return nil, errors.New("master key must not be empty")
	}

	sum := sha256.Sum256([]byte(masterKey))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	return &AESCipher{gcm: gcm}, nil
}

// Encrypt 对明文进行 AES-GCM 加密，返回带 "enc:v1:" 前缀的可存储字符串。
// 空字符串直接原样返回——空密钥无需加密，也便于前端识别“未设置”。
func (c *AESCipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密由 Encrypt 生成的密文（形如 "enc:v1:<base64>"）。
// 空字符串直接返回空；其它格式不合法的输入返回 ErrDecryptFailed。
func (c *AESCipher) Decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}

	if !strings.HasPrefix(stored, encryptedPrefix) {
		return "", fmt.Errorf("%w: missing %q prefix", ErrDecryptFailed, encryptedPrefix)
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encryptedPrefix))
	if err != nil {
		return "", fmt.Errorf("%w: base64: %v", ErrDecryptFailed, err)
	}

	nonceSize := c.gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrDecryptFailed)
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptFailed, err)
	}

	return string(plaintext), nil
}
