package utils

import (
	"errors"
	"strings"
)

// SplitCommandLine splits a command line into arguments, honoring simple quotes.
func SplitCommandLine(input string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, errors.New("unfinished escape sequence in command")
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote in command")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}

// ParseArgs parses a whitespace/quote-aware argument string.
func ParseArgs(input string) ([]string, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	return SplitCommandLine(input)
}
