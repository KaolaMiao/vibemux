package driver

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/lazyvibe/vibemux/internal/model"
)

// NativeDriver launches the claude command directly.
type NativeDriver struct {
	claudePath string // Configured path to claude executable
	codexPath  string // Configured path to codex executable (optional)
}

// NewNativeDriver creates a new NativeDriver instance.
func NewNativeDriver() *NativeDriver {
	return &NativeDriver{}
}

// NewNativeDriverWithConfig creates a NativeDriver with configured paths.
func NewNativeDriverWithConfig(claudePath, codexPath string) *NativeDriver {
	return &NativeDriver{
		claudePath: claudePath,
		codexPath:  codexPath,
	}
}

// Name returns the driver identifier.
func (d *NativeDriver) Name() string {
	return "native"
}

// BuildCommand constructs the command for native execution.
func (d *NativeDriver) BuildCommand(workDir string, profile *model.Profile) (*exec.Cmd, error) {
	if err := d.Validate(profile); err != nil {
		return nil, err
	}

	commandLine := strings.TrimSpace(profile.Command)
	if commandLine == "" {
		commandLine = "claude"
	}

	parts, err := splitCommandLine(commandLine)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, errors.New("command is empty")
	}
	command := parts[0]
	args := parts[1:]

	if command == "" || command == "claude" {
		if d.claudePath != "" {
			command = d.claudePath
		} else {
			command = "claude"
		}
	} else if command == "codex" && d.codexPath != "" {
		command = d.codexPath
	}

	cmd := exec.Command(command, args...)
	cmd.Dir = workDir

	// Merge environment variables
	// Start with current environment, then overlay profile-specific vars
	env := os.Environ()
    // Explicitly set TERM to xterm-256color if not present in profile vars (to ensure TUI apps work)
    hasTerm := false
	for k, v := range profile.EnvVars {
		env = append(env, k+"="+v)
        if k == "TERM" {
            hasTerm = true
        }
	}
    if !hasTerm {
        env = append(env, "TERM=xterm-256color")
        env = append(env, "COLORTERM=truecolor")
    }
    // Auto-inject NODE_OPTIONS to prevent Node.js heap OOM crashes
    hasNodeOptions := false
    for _, e := range env {
        if len(e) > 12 && e[:12] == "NODE_OPTIONS" {
            hasNodeOptions = true
            break
        }
    }
    if !hasNodeOptions {
        env = append(env, "NODE_OPTIONS=--max-old-space-size=4096")
    }
	cmd.Env = env

	return cmd, nil
}

// Validate checks the profile configuration.
func (d *NativeDriver) Validate(profile *model.Profile) error {
	if profile == nil {
		return errors.New("profile is nil")
	}

	commandLine := strings.TrimSpace(profile.Command)
	if commandLine == "" {
		commandLine = "claude"
	}

	parts, err := splitCommandLine(commandLine)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return errors.New("command is empty")
	}
	command := parts[0]

	if command == "" || command == "claude" {
		if d.claudePath != "" {
			command = d.claudePath
		} else {
			command = "claude"
		}
	} else if command == "codex" && d.codexPath != "" {
		command = d.codexPath
	}

	if _, resolved := resolveExecutablePath(command); !resolved {
		return errors.New("command not found: " + command)
	}

	return nil
}
