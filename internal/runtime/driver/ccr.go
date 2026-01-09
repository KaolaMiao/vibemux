package driver

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/lazyvibe/vibemux/internal/model"
)

// CCRDriver launches commands via Claude Code Runner (ccr).
type CCRDriver struct{}

// NewCCRDriver creates a new CCRDriver instance.
func NewCCRDriver() *CCRDriver {
	return &CCRDriver{}
}

// Name returns the driver identifier.
func (d *CCRDriver) Name() string {
	return "ccr"
}

// BuildCommand constructs the command for CCR execution.
func (d *CCRDriver) BuildCommand(workDir string, profile *model.Profile) (*exec.Cmd, error) {
	if err := d.Validate(profile); err != nil {
		return nil, err
	}

	commandLine := strings.TrimSpace(profile.Command)
	if commandLine == "" {
		commandLine = "ccr"
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

	cmd := exec.Command(command, args...)
	cmd.Dir = workDir

	env := os.Environ()
	for k, v := range profile.EnvVars {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	return cmd, nil
}

// Validate checks the profile configuration.
func (d *CCRDriver) Validate(profile *model.Profile) error {
	if profile == nil {
		return errors.New("profile is nil")
	}

	commandLine := strings.TrimSpace(profile.Command)
	if commandLine == "" {
		commandLine = "ccr"
	}

	parts, err := splitCommandLine(commandLine)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return errors.New("command is empty")
	}

	command := parts[0]
	if _, resolved := resolveExecutablePath(command); !resolved {
		return errors.New("command not found: " + command)
	}

	return nil
}
