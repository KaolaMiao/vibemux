// Package driver provides abstractions for launching AI agent processes.
package driver

import (
	"os/exec"

	"github.com/lazyvibe/vibemux/internal/model"
)

// Driver defines the interface for building process commands.
type Driver interface {
	// Name returns the driver identifier.
	Name() string
	// BuildCommand constructs the exec.Cmd for the given profile and working directory.
	BuildCommand(workDir string, profile *model.Profile) (*exec.Cmd, error)
	// Validate checks if the profile configuration is valid for this driver.
	Validate(profile *model.Profile) error
}

// Config holds driver configuration.
type Config struct {
	ClaudePath string
	CodexPath  string
}

// Registry holds all available drivers.
type Registry struct {
	drivers map[model.DriverType]Driver
	config  Config
}

// NewRegistry creates a driver registry with built-in drivers.
func NewRegistry() *Registry {
	return NewRegistryWithConfig(Config{})
}

// NewRegistryWithConfig creates a driver registry with configuration.
func NewRegistryWithConfig(cfg Config) *Registry {
	r := &Registry{
		drivers: make(map[model.DriverType]Driver),
		config:  cfg,
	}

	// Register built-in drivers with configuration
	r.Register(NewNativeDriverWithConfig(cfg.ClaudePath, cfg.CodexPath))
	r.Register(NewCCRDriver())

	return r
}

// Register adds a driver to the registry.
func (r *Registry) Register(d Driver) {
	switch d.Name() {
	case "native":
		r.drivers[model.DriverNative] = d
	case "ccr":
		r.drivers[model.DriverCCR] = d
	case "custom":
		r.drivers[model.DriverCustom] = d
	}
}

// Get retrieves a driver by type.
func (r *Registry) Get(t model.DriverType) (Driver, bool) {
	d, ok := r.drivers[t]
	return d, ok
}
