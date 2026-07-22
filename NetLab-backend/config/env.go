package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// mergeEnvFile accepts standard inline comments, unlike Viper's dotenv decoder.
func mergeEnvFile(v *viper.Viper, filename string) error {
	contents, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", filename, err)
	}

	values := make(map[string]any)
	for number, line := range strings.Split(string(contents), "\n") {
		key, value, ok, err := parseEnvLine(line)
		if err != nil {
			return fmt.Errorf("parse %s line %d: %w", filename, number+1, err)
		}
		if ok {
			values[key] = value
		}
	}
	return v.MergeConfigMap(values)
}

func parseEnvLine(line string) (key, value string, ok bool, err error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}

	line = strings.TrimPrefix(line, "export ")
	key, raw, found := strings.Cut(line, "=")
	if !found || strings.TrimSpace(key) == "" {
		return "", "", false, fmt.Errorf("expected KEY=VALUE")
	}
	key, raw = strings.TrimSpace(key), strings.TrimSpace(raw)
	if raw == "" {
		return key, "", true, nil
	}
	if raw[0] != '\'' && raw[0] != '"' {
		if i := strings.Index(raw, " #"); i >= 0 {
			raw = raw[:i]
		}
		return key, strings.TrimSpace(raw), true, nil
	}

	quote := raw[0]
	end := 1
	for end < len(raw) && (raw[end] != quote || (quote == '"' && raw[end-1] == '\\')) {
		end++
	}
	if end == len(raw) {
		return "", "", false, fmt.Errorf("unterminated quoted value")
	}
	if rest := strings.TrimSpace(raw[end+1:]); rest != "" && !strings.HasPrefix(rest, "#") {
		return "", "", false, fmt.Errorf("unexpected text after quoted value")
	}
	if quote == '\'' {
		return key, raw[1:end], true, nil
	}
	value, err = strconv.Unquote(raw[:end+1])
	if err != nil {
		return "", "", false, fmt.Errorf("invalid quoted value: %w", err)
	}
	return key, value, true, nil
}
