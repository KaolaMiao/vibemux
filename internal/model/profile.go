package model

import (
	"github.com/google/uuid"
)

// Profile defines a configuration set for launching AI agents.
type Profile struct {
	// ID is the unique identifier for this profile.
	ID string `json:"id"`
	// Name is the display name (e.g., "Work-Strict", "Personal-Haiku").
	Name string `json:"name"`
	// Driver specifies the launch method (native/ccr/custom).
	Driver DriverType `json:"driver"`
	// Command is the base command to execute (e.g., "claude", "codex").
	Command string `json:"command"`
	// CommandArgs are additional arguments passed to the command.
	CommandArgs []string `json:"command_args,omitempty"`
	// EnvVars are environment variables injected into the process.
	EnvVars map[string]string `json:"env_vars,omitempty"`
	// AutoApprove sets the automatic approval level.
	AutoApprove AutoApproveLevel `json:"auto_approve"`
	// Notification configures alert settings.
	Notification NotificationConfig `json:"notification"`
	// IsDefault marks this as the default profile for new projects.
	IsDefault bool `json:"is_default"`
}

// NewProfile creates a new profile with sensible defaults.
func NewProfile(name string) *Profile {
	return &Profile{
		ID:          uuid.New().String(),
		Name:        name,
		Driver:      DriverNative,
		Command:     "claude",
		CommandArgs: nil,
		EnvVars:     make(map[string]string),
		AutoApprove: AutoApproveVibe,
		Notification: NotificationConfig{
			Desktop: true,
		},
	}
}

// DefaultProfile returns a pre-configured default profile.
func DefaultProfile() *Profile {
	return &Profile{
		ID:          "default",
		Name:        "Default",
		Driver:      DriverNative,
		Command:     "claude",
		CommandArgs: nil,
		EnvVars:     make(map[string]string),
		AutoApprove: AutoApproveVibe,
		Notification: NotificationConfig{
			Desktop: true,
		},
		IsDefault: true,
	}
}

// SetEnvVar adds or updates an environment variable.
func (p *Profile) SetEnvVar(key, value string) {
	if p.EnvVars == nil {
		p.EnvVars = make(map[string]string)
	}
	p.EnvVars[key] = value
}

// GetEnvSlice returns environment variables as a slice of "KEY=VALUE" strings.
func (p *Profile) GetEnvSlice() []string {
	result := make([]string, 0, len(p.EnvVars))
	for k, v := range p.EnvVars {
		result = append(result, k+"="+v)
	}
	return result
}

// Clone creates a deep copy of the profile with a new ID and name.
func (p *Profile) Clone(newName string) *Profile {
	newEnv := make(map[string]string, len(p.EnvVars))
	for k, v := range p.EnvVars {
		newEnv[k] = v
	}
	newArgs := make([]string, len(p.CommandArgs))
	copy(newArgs, p.CommandArgs)

	return &Profile{
		ID:           uuid.New().String(),
		Name:         newName,
		Driver:       p.Driver,
		Command:      p.Command,
		CommandArgs:  newArgs,
		EnvVars:      newEnv,
		AutoApprove:  p.AutoApprove,
		Notification: p.Notification,
		IsDefault:    false,
	}
}
