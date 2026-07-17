package crypto

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("72_bytes_ok", func(t *testing.T) {
		pwd := strings.Repeat("a", 72)
		hash, err := HashPassword(pwd)
		if err != nil {
			t.Fatalf("HashPassword(72 bytes) failed: %v", err)
		}
		if !VerifyPassword(hash, pwd) {
			t.Error("VerifyPassword failed for 72-byte password")
		}
	})

	t.Run("73_bytes_fails", func(t *testing.T) {
		pwd := strings.Repeat("a", 73)
		_, err := HashPassword(pwd)
		if err != bcrypt.ErrPasswordTooLong {
			t.Errorf("HashPassword(73 bytes) err = %v, want ErrPasswordTooLong", err)
		}
	})

	t.Run("multibyte_utf8_exact_72_bytes", func(t *testing.T) {
		// 每个中文字符 3 字节，24 个中文字 = 72 字节
		pwd := strings.Repeat("中", 24)
		if len(pwd) != 72 {
			t.Fatalf("expected 72 bytes, got %d", len(pwd))
		}
		hash, err := HashPassword(pwd)
		if err != nil {
			t.Fatalf("HashPassword(multibyte 72 bytes) failed: %v", err)
		}
		if !VerifyPassword(hash, pwd) {
			t.Error("VerifyPassword failed for multibyte 72-byte password")
		}
	})

	t.Run("multibyte_utf8_25_chars_75_bytes_fails", func(t *testing.T) {
		pwd := strings.Repeat("中", 25)
		if len(pwd) != 75 {
			t.Fatalf("expected 75 bytes, got %d", len(pwd))
		}
		_, err := HashPassword(pwd)
		if err != bcrypt.ErrPasswordTooLong {
			t.Errorf("HashPassword(75 bytes) err = %v, want ErrPasswordTooLong", err)
		}
	})

	t.Run("verify_compatibility", func(t *testing.T) {
		// 验证既有 bcrypt 哈希的兼容性
		pwd := "test-password-123"
		hash, err := HashPassword(pwd)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}
		if !VerifyPassword(hash, pwd) {
			t.Error("VerifyPassword failed for known password")
		}
		// 错误密码应返回 false（不 panic）
		if VerifyPassword(hash, "wrong-password") {
			t.Error("VerifyPassword returned true for wrong password")
		}
	})
}
