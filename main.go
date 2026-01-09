// VibeMux - AI Agent Orchestration Terminal
// A TUI for managing multiple Claude Code or Codex instances.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvibe/vibemux/internal/app"
	"github.com/lazyvibe/vibemux/internal/runtime"
	"github.com/lazyvibe/vibemux/internal/runtime/driver"
	"github.com/lazyvibe/vibemux/internal/store"
	"github.com/lazyvibe/vibemux/internal/ui"
	"github.com/lazyvibe/vibemux/internal/ui/components/setup"
)

const (
	appName    = "VibeMux"
	appVersion = "0.1.0"
)

func main() {
	// Get config directory
	configDir, err := getConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting config directory: %v\n", err)
		os.Exit(1)
	}

	// Load application configuration
	config, err := app.LoadConfig(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check if first-run setup is needed
	if !config.Initialized {
		if err := runSetupWizard(configDir, config); err != nil {
			fmt.Fprintf(os.Stderr, "Error running setup wizard: %v\n", err)
			os.Exit(1)
		}
		// Reload config after setup
		config, err = app.LoadConfig(configDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reloading config: %v\n", err)
			os.Exit(1)
		}
	}

	// Initialize store
	s, err := store.NewJSONStore(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing store: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	// Initialize runtime engine with configuration
	driverCfg := driver.Config{
		ClaudePath: config.ClaudePath,
		CodexPath:  config.CodexPath,
	}
	engine := runtime.NewEngineWithConfig(driverCfg)
	defer engine.CloseAll()

	// Create application
	application := ui.New(s, engine, config, configDir)

	// Run the TUI
	p := tea.NewProgram(
		application,
		tea.WithAltScreen(),       // Use alternate screen buffer
        tea.WithMouseCellMotion(), // Enable mouse support
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

// runSetupWizard runs the first-run setup wizard.
func runSetupWizard(configDir string, config *app.Config) error {
	wizard := setup.New(configDir, config)

	p := tea.NewProgram(
		wizard,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if setup was completed
	if m, ok := finalModel.(setup.Model); ok {
		if !m.IsComplete() {
			// User quit without completing setup
			os.Exit(0)
		}
	}

	return nil
}

// getConfigDir returns the VibeMux configuration directory.
func getConfigDir() (string, error) {
	// Use XDG_CONFIG_HOME if available, otherwise default to ~/.config
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}

	return filepath.Join(configHome, "vibemux"), nil
}
