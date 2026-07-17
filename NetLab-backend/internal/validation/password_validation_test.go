package validation

import (
	"testing"

	"netlab-backend/pkg/apperrors"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantWeak bool // true = 应返回 ErrWeakPassword
	}{
		{name: "too_short_7chars", password: "Aa1!aaa", wantWeak: true},
		{name: "minimum_8chars", password: "Aa1!aaaa", wantWeak: false},
		{name: "too_long_73bytes", password: string(make([]byte, 73)), wantWeak: true},
		{name: "invalid_utf8", password: string([]byte{0xff, 0xfe, 0x81}), wantWeak: true},
		{name: "missing_lowercase", password: "AA1!AAAAA", wantWeak: true},
		{name: "missing_uppercase", password: "aa1!aaaaa", wantWeak: true},
		{name: "missing_digit", password: "A!aaaaaa", wantWeak: true},
		{name: "missing_special", password: "Aa1aaaaaa", wantWeak: true},
		{name: "strong_password", password: "S3cur3P@ssw0rd!2024", wantWeak: false},
		{name: "unicode_special_character", password: "Aa1234567中", wantWeak: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantWeak {
				if err == nil {
					t.Errorf("ValidatePassword() expected weak password error, got nil")
				} else if err.Code != apperrors.ErrCodeWeakPassword {
					t.Errorf("ValidatePassword() code = %d, want %d", err.Code, apperrors.ErrCodeWeakPassword)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePassword() unexpected error: %v", err)
				}
			}
		})
	}
}
