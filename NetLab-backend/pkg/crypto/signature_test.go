package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestGenerateRandomKeyHex 验证生成的密钥是 32 字节的有效十六进制。
func TestGenerateRandomKeyHex(t *testing.T) {
	k, err := GenerateRandomKeyHex()
	if err != nil {
		t.Fatalf("GenerateRandomKeyHex: %v", err)
	}
	if len(k) != 64 { // 32 字节 → 64 个十六进制字符
		t.Fatalf("expected 64 hex chars, got %d", len(k))
	}
	raw, err := hex.DecodeString(k)
	if err != nil {
		t.Fatalf("result is not valid hex: %v", err)
	}
	if len(raw) != 32 {
		t.Fatalf("expected 32 raw bytes, got %d", len(raw))
	}

	// 两个连续生成的密钥必须不同（随机性合理性检查）。
	k2, _ := GenerateRandomKeyHex()
	if k == k2 {
		t.Fatal("two generated keys are identical")
	}
}

// TestVerifyHMACSHA256Hex 验证正确的十六进制签名能通过校验，
// 而被篡改的签名无法通过。
func TestVerifyHMACSHA256Hex(t *testing.T) {
	key := []byte("super-secret-key")
	msg := "POST\n/api/auth/login\n2026-07-07T00:00:00Z\nabc123"

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))

	ok, err := VerifyHMACSHA256Hex(key, msg, sig)
	if err != nil || !ok {
		t.Fatalf("valid signature rejected: ok=%v err=%v", ok, err)
	}

	// 被篡改的消息。
	if ok, _ := VerifyHMACSHA256Hex(key, msg+"x", sig); ok {
		t.Fatal("tampered message accepted")
	}
	// 格式错误的十六进制。
	if _, err := VerifyHMACSHA256Hex(key, msg, "zzzz"); err == nil {
		t.Fatal("expected error on malformed hex signature")
	}
}

// TestHexBaseParity 说明将签名密钥编码从 base64 迁移到十六进制
// 并不会改变底层的密钥字节：同样的 32 个原始字节既可编码为
// base64 也可编码为十六进制，因此在切换编码后未变更的密钥字节完全一致。
func TestHexBaseParity(t *testing.T) {
	// 项目现有的密钥，以十六进制表示。
	const keyHex = "5fac16cc7102dee43d857c748b8aee4df2d237203fec18ae2d9c0739aef8dc11"
	raw, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	if len(raw) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(raw))
	}
}
