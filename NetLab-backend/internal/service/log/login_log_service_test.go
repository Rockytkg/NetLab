package log

import (
	"strings"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{name: "短字符串不截断", s: "abc", max: 5, want: "abc"},
		{name: "恰好等于上限", s: "abcde", max: 5, want: "abcde"},
		{name: "超长截断", s: "abcdefgh", max: 5, want: "abcde"},
		{name: "空字符串", s: "", max: 5, want: ""},
		{name: "长 UA 截断到 512", s: strings.Repeat("a", 600), max: 512, want: strings.Repeat("a", 512)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.s, tt.max); got != tt.want {
				t.Errorf("truncate() len = %d, want %d", len(got), len(tt.want))
			}
		})
	}
}
