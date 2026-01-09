package utils

import (
	"errors"
	"sort"
	"strings"
)

// ParseEnvVars parses KEY=VALUE pairs separated by commas, semicolons, or newlines.
func ParseEnvVars(input string) (map[string]string, error) {
	result := make(map[string]string)
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return result, nil
	}

	parts := splitEnvInput(trimmed)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, errors.New("invalid env var: " + part)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, errors.New("invalid env var: " + part)
		}
		result[key] = strings.TrimSpace(value)
	}

	return result, nil
}

// FormatEnvVars formats env vars as a comma-separated list of KEY=VALUE entries.
func FormatEnvVars(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+env[k])
	}
	return strings.Join(parts, ", ")
}

func splitEnvInput(input string) []string {
	return strings.FieldsFunc(input, func(r rune) bool {
		switch r {
		case ',', ';', '\n':
			return true
		default:
			return false
		}
	})
}
