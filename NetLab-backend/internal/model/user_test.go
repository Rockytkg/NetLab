package model

import (
	"reflect"
	"testing"
)

func TestRecoveryCodesScan(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  RecoveryCodes
	}{
		{name: "postgres jsonb bytes", value: []byte(`["hash-a","hash-b"]`), want: RecoveryCodes{"hash-a", "hash-b"}},
		{name: "string", value: `["hash-a"]`, want: RecoveryCodes{"hash-a"}},
		{name: "null", value: nil, want: RecoveryCodes{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got RecoveryCodes
			if err := got.Scan(tt.value); err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Scan() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestRecoveryCodesValue(t *testing.T) {
	value, err := (RecoveryCodes{"hash-a", "hash-b"}).Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	if string(value.([]byte)) != `["hash-a","hash-b"]` {
		t.Fatalf("Value() = %s", value)
	}
}
