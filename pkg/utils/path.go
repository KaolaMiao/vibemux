// Package utils provides utility functions for VibeMux.
package utils

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PathCompleter provides path completion functionality.
type PathCompleter struct {
	recentPaths []string
}

// NewPathCompleter creates a new path completer.
func NewPathCompleter(recentPaths []string) *PathCompleter {
	return &PathCompleter{
		recentPaths: recentPaths,
	}
}

// Complete returns completion suggestions for the given input.
func (c *PathCompleter) Complete(input string) []string {
	if input == "" {
		return c.getDefaultSuggestions()
	}

	// Expand ~ to home directory
	expanded := expandHome(input)

	// Get directory and prefix
	dir := filepath.Dir(expanded)
	prefix := filepath.Base(expanded)

	// If input ends with /, list directory contents
	if strings.HasSuffix(input, "/") || strings.HasSuffix(input, string(filepath.Separator)) {
		dir = expanded
		prefix = ""
	}

	// Read directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Try recent paths matching the input
		return c.matchRecentPaths(input)
	}

	var suggestions []string

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless input starts with .
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}

		// Match prefix
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}

		// Build full path
		fullPath := filepath.Join(dir, name)

		// Convert back to user-friendly format
		if strings.HasPrefix(input, "~") {
			home, _ := os.UserHomeDir()
			fullPath = "~" + strings.TrimPrefix(fullPath, home)
		}

		// Add trailing slash for directories
		if entry.IsDir() {
			fullPath += "/"
		}

		suggestions = append(suggestions, fullPath)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(suggestions, func(i, j int) bool {
		iDir := strings.HasSuffix(suggestions[i], "/")
		jDir := strings.HasSuffix(suggestions[j], "/")
		if iDir != jDir {
			return iDir
		}
		return suggestions[i] < suggestions[j]
	})

	// Limit results
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	return suggestions
}

// getDefaultSuggestions returns default path suggestions.
func (c *PathCompleter) getDefaultSuggestions() []string {
	home, _ := os.UserHomeDir()
	suggestions := []string{
		"~/",
		"~/Projects/",
		"~/Code/",
		"~/Documents/",
		"~/Desktop/",
		"/",
	}

	// Add recent paths
	for i, p := range c.recentPaths {
		if i >= 5 {
			break
		}
		// Convert to user-friendly format
		if strings.HasPrefix(p, home) {
			p = "~" + strings.TrimPrefix(p, home)
		}
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		suggestions = append([]string{p}, suggestions...)
	}

	return dedupe(suggestions)
}

// matchRecentPaths returns recent paths matching the prefix.
func (c *PathCompleter) matchRecentPaths(prefix string) []string {
	var matches []string
	expanded := expandHome(prefix)

	for _, p := range c.recentPaths {
		if strings.HasPrefix(p, expanded) || strings.HasPrefix(p, prefix) {
			home, _ := os.UserHomeDir()
			display := p
			if strings.HasPrefix(display, home) {
				display = "~" + strings.TrimPrefix(display, home)
			}
			matches = append(matches, display)
		}
	}

	return matches
}

// expandHome expands ~ to the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

// ExpandPath expands ~ and normalizes the path.
func ExpandPath(path string) string {
	expanded := expandHome(path)
	return filepath.Clean(expanded)
}

// dedupe removes duplicates from a string slice.
func dedupe(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// IsValidProjectPath checks if a path is a valid project directory.
func IsValidProjectPath(path string) bool {
	expanded := ExpandPath(path)
	info, err := os.Stat(expanded)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetProjectName extracts a project name from a path.
func GetProjectName(path string) string {
	expanded := ExpandPath(path)
	return filepath.Base(expanded)
}

// ListDirectories lists subdirectories in a directory.
func ListDirectories(dir string) ([]string, error) {
	expanded := ExpandPath(dir)
	entries, err := os.ReadDir(expanded)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs, nil
}
