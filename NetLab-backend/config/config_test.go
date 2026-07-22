package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestParseEnvLineStripsInlineComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line  string
		key   string
		value string
		ok    bool
	}{
		{"DB_NAME=netlab # database name", "DB_NAME", "netlab", true},
		{"DB_PORT=5432                  # database port", "DB_PORT", "5432", true},
		{"DB_PASSWORD=pa#ss # comment", "DB_PASSWORD", "pa#ss", true},
		{`DB_PASSWORD="pa ss#word" # comment`, "DB_PASSWORD", "pa ss#word", true},
		{"# comment", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			key, value, ok, err := parseEnvLine(tt.line)
			if err != nil {
				t.Fatalf("parseEnvLine() error = %v", err)
			}
			if key != tt.key || value != tt.value || ok != tt.ok {
				t.Fatalf("parseEnvLine() = (%q, %q, %t), want (%q, %q, %t)", key, value, ok, tt.key, tt.value, tt.ok)
			}
		})
	}
}

func TestDatabaseConfigDSNEscapesValues(t *testing.T) {
	t.Parallel()

	dsn := (DatabaseConfig{
		Host:     "::1",
		Port:     5432,
		User:     "root user",
		Password: "pa ss#word",
		Name:     "netlab test",
		SSLMode:  "disable",
	}).DSN()

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	password, _ := u.User.Password()
	if u.Host != "[::1]:5432" || u.User.Username() != "root user" || password != "pa ss#word" || u.Path != "/netlab test" || u.Query().Get("sslmode") != "disable" {
		t.Fatalf("DSN parsed incorrectly: %#v", u)
	}
}

func TestMergeEnvFileStripsComments(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), ".env")
	contents := "DB_NAME=netlab # database name\nDB_PORT=5432 # database port\nDB_PASSWORD=pa#ss # comment\n"
	if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	v := viper.New()
	if err := mergeEnvFile(v, filename); err != nil {
		t.Fatalf("mergeEnvFile() error = %v", err)
	}
	if got, want := v.GetString("DB_NAME"), "netlab"; got != want {
		t.Fatalf("DB_NAME = %q, want %q", got, want)
	}
	if got, want := v.GetInt("DB_PORT"), 5432; got != want {
		t.Fatalf("DB_PORT = %d, want %d", got, want)
	}
	if got, want := v.GetString("DB_PASSWORD"), "pa#ss"; got != want {
		t.Fatalf("DB_PASSWORD = %q, want %q", got, want)
	}
}

func TestValidateDatabaseConfig(t *testing.T) {
	t.Parallel()

	valid := DatabaseConfig{Host: "localhost", Port: 5432, User: "netlab", Name: "netlab", MaxOpenConns: 25, MaxIdleConns: 10}
	if err := validateDatabaseConfig(valid); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	valid.Name = ""
	if err := validateDatabaseConfig(valid); err == nil {
		t.Fatal("empty DB_NAME was accepted")
	}
}
