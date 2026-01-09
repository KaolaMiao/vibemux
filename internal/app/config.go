// Package app provides application-level configuration and initialization.
package app

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	// ClaudePath is the full path to the claude executable.
	ClaudePath string `json:"claude_path"`
	// CodexPath is the full path to the codex executable (optional).
	CodexPath string `json:"codex_path,omitempty"`
	// DefaultShell is the default shell to use.
	DefaultShell string `json:"default_shell"`
	// Initialized indicates if the first-run setup has been completed.
	Initialized bool `json:"initialized"`
	// Theme is the color theme (future use).
	Theme string `json:"theme"`
	// RecentPaths stores recently used project paths for completion.
	RecentPaths []string `json:"recent_paths,omitempty"`
	// GridRows is the number of terminal rows in the grid layout.
	GridRows int `json:"grid_rows,omitempty"`
	// GridCols is the number of terminal columns in the grid layout.
	GridCols int `json:"grid_cols,omitempty"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "cmd.exe"
		} else {
			shell = "/bin/sh"
		}
	}

	return &Config{
		DefaultShell: shell,
		Theme:        "catppuccin-mocha",
		RecentPaths:  []string{},
		GridRows:     2,
		GridCols:     2,
	}
}

// ConfigPath returns the path to the config file.
func ConfigPath(configDir string) string {
	return filepath.Join(configDir, "config.json")
}

// LoadConfig loads the configuration from disk.
func LoadConfig(configDir string) (*Config, error) {
	path := ConfigPath(configDir)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig saves the configuration to disk.
func SaveConfig(configDir string, config *Config) error {
	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(configDir), data, 0644)
}

// DetectClaudePath attempts to find the claude executable.
func DetectClaudePath() string {
	// Common installation paths to check
	candidates := []string{
		"claude", // In PATH
	}

	// Add platform-specific paths
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		candidates = append(candidates,
			"/opt/homebrew/bin/claude",
			"/usr/local/bin/claude",
			filepath.Join(home, ".local/bin/claude"),
			filepath.Join(home, ".npm-global/bin/claude"),
			"/opt/homebrew/opt/claude/bin/claude",
		)
		// Check for npm global installations
		npmPrefix, err := exec.Command("npm", "config", "get", "prefix").Output()
		if err == nil {
			npmBin := filepath.Join(strings.TrimSpace(string(npmPrefix)), "bin", "claude")
			candidates = append(candidates, npmBin)
		}
	} else if runtime.GOOS == "linux" {
		home, _ := os.UserHomeDir()
		candidates = append(candidates,
			"/usr/local/bin/claude",
			"/usr/bin/claude",
			filepath.Join(home, ".local/bin/claude"),
			filepath.Join(home, ".npm-global/bin/claude"),
		)
	} else if runtime.GOOS == "windows" {
		home, _ := os.UserHomeDir()
		candidates = append(candidates,
			filepath.Join(home, "AppData", "Roaming", "npm", "claude.cmd"),
			filepath.Join(home, "AppData", "Local", "Programs", "claude", "claude.exe"),
		)
	}

	// Try to find claude in PATH first
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}

	// Check each candidate path
	for _, path := range candidates {
		if path == "claude" {
			continue // Already checked via LookPath
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// DetectCodexPath attempts to find the codex executable.
func DetectCodexPath() string {
	// Try PATH first
	if path, err := exec.LookPath("codex"); err == nil {
		return path
	}

	// Platform-specific paths
	candidates := []string{}

	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		candidates = append(candidates,
			"/opt/homebrew/bin/codex",
			"/usr/local/bin/codex",
			filepath.Join(home, ".local/bin/codex"),
		)
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// ValidateClaudePath checks if the given path is a valid claude executable.
func ValidateClaudePath(path string) bool {
	if path == "" {
		return false
	}

	// Check if file exists and is executable
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// On Unix, check executable permission
	if runtime.GOOS != "windows" {
		return info.Mode()&0111 != 0
	}

	return true
}

// AddRecentPath adds a path to the recent paths list.
func (c *Config) AddRecentPath(path string) {
	// Normalize path
	path = filepath.Clean(path)

	// Remove if already exists
	paths := make([]string, 0, len(c.RecentPaths))
	for _, p := range c.RecentPaths {
		if p != path {
			paths = append(paths, p)
		}
	}

	// Add to front
	c.RecentPaths = append([]string{path}, paths...)

	// Keep only last 20
	if len(c.RecentPaths) > 20 {
		c.RecentPaths = c.RecentPaths[:20]
	}
}

// GetRecentPaths returns recent paths matching the given prefix.
func (c *Config) GetRecentPaths(prefix string) []string {
	if prefix == "" {
		return c.RecentPaths
	}

	var matches []string
	for _, p := range c.RecentPaths {
		if strings.HasPrefix(p, prefix) {
			matches = append(matches, p)
		}
	}
	return matches
}
