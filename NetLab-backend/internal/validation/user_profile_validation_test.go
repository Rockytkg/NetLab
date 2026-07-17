package validation

import "testing"

func TestNormalizeNickname(t *testing.T) {
	if got, err := NormalizeNickname("  Alice "); err != nil || got != "Alice" {
		t.Fatalf("NormalizeNickname() = %q, %v", got, err)
	}
	if _, err := NormalizeNickname(""); err == nil {
		t.Fatal("expected empty nickname to fail")
	}
}

func TestNormalizePhone(t *testing.T) {
	for _, phone := range []string{"13800000000", "19912345678"} {
		if got, err := NormalizePhone(phone); err != nil || got != phone {
			t.Fatalf("NormalizePhone(%q) = %q, %v", phone, got, err)
		}
	}
	for _, phone := range []string{"1380000000", "12800000000", "1380000000a"} {
		if _, err := NormalizePhone(phone); err == nil {
			t.Fatalf("expected invalid phone %q to fail", phone)
		}
	}
}
