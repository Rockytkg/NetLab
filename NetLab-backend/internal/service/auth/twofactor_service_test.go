package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestNormalizeTwoFactorCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"123456", "123456"},
		{"  123456  ", "123456"},
		{"12345", ""},
		{"1234567", ""},
		{"abcdef", ""},
		{"", ""},
		{"12a456", ""},
	}
	for _, c := range cases {
		if got := normalizeTwoFactorCode(c.in); got != c.want {
			t.Errorf("normalizeTwoFactorCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTOTPRoundTrip(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "NetLab", AccountName: "user@example.com"})
	if err != nil {
		t.Fatalf("generate totp key: %v", err)
	}
	code, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	if normalizeTwoFactorCode(code) == "" {
		t.Fatalf("generated code not normalized: %q", code)
	}
	if !totp.Validate(code, key.Secret()) {
		t.Fatal("totp.Validate failed for freshly generated code")
	}
}

func TestNormalizeRecoveryCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ABCD-EFGH-IJKL-MNPQ", "ABCDEFGHIJKLMNPQ"},
		{"abcd-efgh-ijkl-mnpq", "ABCDEFGHIJKLMNPQ"},
		{"  ABCD EFGH IJKL MNPQ  ", "ABCDEFGHIJKLMNPQ"},
		{"ABCDEFGHIJKLMNOP", "ABCDEFGHIJKLMNOP"},
		{"ABCD-EFGH-IJKL", ""},       // 太短
		{"ABCD-EFGH-IJKL-MNPQR", ""}, // 太长
		{"ABCD-EFGH-IJKL-MN!Q", ""},  // 非法字符
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeRecoveryCode(c.in); got != c.want {
			t.Errorf("normalizeRecoveryCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGenerateRecoveryCodes(t *testing.T) {
	codes, hashes := generateRecoveryCodes(10)
	if len(codes) != 10 || len(hashes) != 10 {
		t.Fatalf("expected 10 codes and hashes, got %d / %d", len(codes), len(hashes))
	}
	seen := make(map[string]bool, len(codes))
	for i, c := range codes {
		// 明文格式应为 XXXX-XXXX-XXXX-XXXX（19 字符，含 3 个连字符）
		if len(c) != 19 {
			t.Errorf("code %d has wrong length %d: %q", i, len(c), c)
		}
		normalized := normalizeRecoveryCode(c)
		if normalized == "" {
			t.Fatalf("generated code does not normalize: %q", c)
		}
		// 哈希应与规范化明文一致
		if sha256Hex(normalized) != hashes[i] {
			t.Errorf("hash mismatch for code %q", c)
		}
		if seen[c] {
			t.Fatalf("duplicate recovery code generated: %q", c)
		}
		seen[c] = true
	}
}

func TestFormatRecoveryCode(t *testing.T) {
	if got := formatRecoveryCode("ABCDEFGHIJKLMNOP"); got != "ABCD-EFGH-IJKL-MNOP" {
		t.Errorf("formatRecoveryCode = %q, want %q", got, "ABCD-EFGH-IJKL-MNOP")
	}
}
