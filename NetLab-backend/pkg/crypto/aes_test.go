package crypto

import (
	"strings"
	"testing"
)

func TestAESCipher_RoundTrip(t *testing.T) {
	c, err := NewAESCipher("test-master-key")
	if err != nil {
		t.Fatalf("NewAESCipher: %v", err)
	}

	cases := []string{"", "hunter2", "a very long secret with spaces and 符号 🎉"}
	for _, plain := range cases {
		enc, err := c.Encrypt(plain)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plain, err)
		}
		if plain == "" {
			if enc != "" {
				t.Errorf("Encrypt(\"\") = %q, want empty", enc)
			}
			continue
		}
		if !strings.HasPrefix(enc, "enc:v1:") {
			t.Errorf("Encrypt(%q) = %q, expected enc:v1: prefix", plain, enc)
		}
		dec, err := c.Decrypt(enc)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if dec != plain {
			t.Errorf("round trip = %q, want %q", dec, plain)
		}
	}
}

func TestAESCipher_DecryptRejectsNonCiphertext(t *testing.T) {
	c, _ := NewAESCipher("k")
	// 没有 enc:v1: 前缀的值不再被视为明文，应返回错误。
	if _, err := c.Decrypt("legacy-plaintext"); err == nil {
		t.Error("expected error for value without enc:v1: prefix, got nil")
	}
}

func TestAESCipher_DecryptWrongKeyFails(t *testing.T) {
	c1, _ := NewAESCipher("key-one")
	c2, _ := NewAESCipher("key-two")

	enc, _ := c1.Encrypt("secret")
	if _, err := c2.Decrypt(enc); err == nil {
		t.Error("expected decrypt with wrong key to fail, got nil error")
	}
}

func TestAESCipher_NonceIsRandom(t *testing.T) {
	c, _ := NewAESCipher("k")
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Error("expected distinct ciphertexts for repeated plaintext (random nonce)")
	}
}
