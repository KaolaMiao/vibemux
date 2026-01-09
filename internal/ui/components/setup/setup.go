// Package setup provides the first-run setup wizard.
package setup

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazyvibe/vibemux/internal/app"
	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/store"
	"github.com/lazyvibe/vibemux/internal/ui/components/dialog"
	"github.com/lazyvibe/vibemux/internal/ui/styles"
	"github.com/lazyvibe/vibemux/pkg/utils"
)

// Step represents a setup wizard step.
type Step int

const (
	StepWelcome Step = iota
	StepDetectClaude
	StepConfigureClaude
	StepProfileIntro
	StepConfigureProfile
	StepAddAnotherProfile
	StepComplete
)

// Model is the setup wizard model.
type Model struct {
	step               Step
	config             *app.Config
	configDir          string
	claudeInput        textinput.Model
	detectedPath       string
	error              string
	width              int
	height             int
	store              *store.JSONStore
	storeErr           string
	profileDialog      dialog.InputDialog
	profilesConfigured int
}

// New creates a new setup wizard.
func New(configDir string, config *app.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "/path/to/claude"
	ti.CharLimit = 256
	ti.Width = 50

	var storeErr string
	s, err := store.NewJSONStore(configDir)
	if err != nil {
		storeErr = err.Error()
	}

	return Model{
		step:        StepWelcome,
		config:      config,
		configDir:   configDir,
		claudeInput: ti,
		store:       s,
		storeErr:    storeErr,
	}
}

// Init initializes the setup wizard.
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize sets the wizard dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.profileDialog.SetSize(width, height)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.profileDialog.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		if m.step == StepConfigureProfile {
			var cmd tea.Cmd
			m.profileDialog, cmd = m.profileDialog.Update(msg)
			if m.profileDialog.IsSubmitted() {
				if err := m.saveProfileFromDialog(); err != nil {
					m.error = err.Error()
					return m, nil
				}
				m.error = ""
				m.step = StepAddAnotherProfile
				return m, nil
			}
			if m.profileDialog.IsCancelled() {
				m.step = StepProfileIntro
				return m, nil
			}
			return m, cmd
		}

		switch msg.String() {
		case "enter":
			return m.handleEnter()

		case "esc":
			if m.step == StepConfigureClaude {
				m.step = StepDetectClaude
				return m, nil
			}
			if m.step == StepProfileIntro || m.step == StepAddAnotherProfile {
				m.step = StepComplete
				return m, nil
			}
		case "a":
			if m.step == StepAddAnotherProfile {
				m.step = StepConfigureProfile
				m.initProfileDialog()
				return m, nil
			}
		}
	}

	// Update text input if on configure step
	if m.step == StepConfigureClaude {
		var cmd tea.Cmd
		m.claudeInput, cmd = m.claudeInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleEnter processes the enter key for each step.
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepWelcome:
		m.step = StepDetectClaude
		// Auto-detect claude
		m.detectedPath = app.DetectClaudePath()
		return m, nil

	case StepDetectClaude:
		if m.detectedPath != "" {
			// Use detected path
			m.config.ClaudePath = m.detectedPath
			m.config.Initialized = true
			if err := app.SaveConfig(m.configDir, m.config); err != nil {
				m.error = err.Error()
				return m, nil
			}
			if m.storeErr != "" {
				m.step = StepComplete
			} else {
				m.step = StepProfileIntro
			}
		} else {
			// Go to manual configuration
			m.step = StepConfigureClaude
			m.claudeInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case StepConfigureClaude:
		path := strings.TrimSpace(m.claudeInput.Value())
		if path == "" {
			m.error = "Please enter a path to the claude executable"
			return m, nil
		}

		// Expand ~ if present
		if strings.HasPrefix(path, "~") {
			path = strings.Replace(path, "~", "", 1)
			// This will be expanded by the config
		}

		if !app.ValidateClaudePath(path) {
			m.error = "Invalid path or file is not executable"
			return m, nil
		}

		m.config.ClaudePath = path
		m.config.Initialized = true
		if err := app.SaveConfig(m.configDir, m.config); err != nil {
			m.error = err.Error()
			return m, nil
		}
		if m.storeErr != "" {
			m.step = StepComplete
		} else {
			m.step = StepProfileIntro
		}
		return m, nil

	case StepProfileIntro:
		m.step = StepConfigureProfile
		m.initProfileDialog()
		return m, nil

	case StepAddAnotherProfile:
		m.step = StepComplete
		return m, nil

	case StepComplete:
		return m, tea.Quit
	}

	return m, nil
}

func (m *Model) initProfileDialog() {
	m.profileDialog = dialog.NewInputDialog("Create Profile", []dialog.InputField{
		{Label: "Profile Name", Placeholder: "My Profile"},
		{Label: "Command", Placeholder: "claude, codex, or ccr code"},
		{Label: "Env Vars", Placeholder: "KEY=VALUE, KEY2=VALUE2"},
	})
	m.profileDialog.SetSize(m.width, m.height)
}

func (m *Model) saveProfileFromDialog() error {
	if m.store == nil {
		return errors.New("profile store unavailable")
	}
	values := m.profileDialog.Values()
	if len(values) < 3 {
		return errors.New("profile form is incomplete")
	}

	name := strings.TrimSpace(values[0])
	command := strings.TrimSpace(values[1])
	envInput := strings.TrimSpace(values[2])

	if name == "" {
		return errors.New("profile name is required")
	}

	if command == "" {
		command = defaultProfileCommand()
	}

	envVars, err := utils.ParseEnvVars(envInput)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if m.profilesConfigured == 0 {
		profiles, err := m.store.ListProfiles(ctx)
		if err == nil && len(profiles) == 1 && profiles[0].IsDefault {
			p := profiles[0]
			p.Name = name
			p.Command = command
			p.EnvVars = envVars
			p.IsDefault = true
			p.Driver = model.DriverNative
			p.CommandArgs = nil
			if err := m.store.UpdateProfile(ctx, &p); err != nil {
				return err
			}
			m.profilesConfigured++
			return nil
		}
	}

	profile := model.NewProfile(name)
	profile.Command = command
	profile.EnvVars = envVars
	profile.Driver = model.DriverNative
	profile.CommandArgs = nil
	if m.profilesConfigured == 0 {
		profile.IsDefault = true
	}

	if err := m.store.CreateProfile(ctx, profile); err != nil {
		return err
	}
	if profile.IsDefault {
		if err := m.setDefaultProfile(ctx, profile.ID); err != nil {
			return err
		}
	}

	m.profilesConfigured++
	return nil
}

func defaultProfileCommand() string {
	return "claude"
}

func (m *Model) setDefaultProfile(ctx context.Context, id string) error {
	profiles, err := m.store.ListProfiles(ctx)
	if err != nil {
		return err
	}
	for i := range profiles {
		shouldBeDefault := profiles[i].ID == id
		if profiles[i].IsDefault != shouldBeDefault {
			profiles[i].IsDefault = shouldBeDefault
			if err := m.store.UpdateProfile(ctx, &profiles[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsComplete returns true if setup is complete.
func (m Model) IsComplete() bool {
	return m.step == StepComplete
}

// Config returns the configured config.
func (m Model) Config() *app.Config {
	return m.config
}

// View renders the setup wizard.
func (m Model) View() string {
	switch m.step {
	case StepWelcome:
		return m.viewWelcome()
	case StepDetectClaude:
		return m.viewDetect()
	case StepConfigureClaude:
		return m.viewConfigure()
	case StepProfileIntro:
		return m.viewProfileIntro()
	case StepConfigureProfile:
		return m.profileDialog.View()
	case StepAddAnotherProfile:
		return m.viewProfileAddAnother()
	case StepComplete:
		return m.viewComplete()
	}
	return ""
}

func (m Model) viewWelcome() string {
	logo := `
 ██╗   ██╗██╗██████╗ ███████╗███╗   ███╗██╗   ██╗██╗  ██╗
 ██║   ██║██║██╔══██╗██╔════╝████╗ ████║██║   ██║╚██╗██╔╝
 ██║   ██║██║██████╔╝█████╗  ██╔████╔██║██║   ██║ ╚███╔╝
 ╚██╗ ██╔╝██║██╔══██╗██╔══╝  ██║╚██╔╝██║██║   ██║ ██╔██╗
  ╚████╔╝ ██║██████╔╝███████╗██║ ╚═╝ ██║╚██████╔╝██╔╝ ██╗
   ╚═══╝  ╚═╝╚═════╝ ╚══════╝╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═╝`

	styledLogo := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render(logo)

	title := lipgloss.NewStyle().
		Foreground(styles.Accent).
		Bold(true).
		Render("Welcome to VibeMux!")

	subtitle := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Render("AI Agent Orchestration Terminal")

	desc := lipgloss.NewStyle().
		Foreground(styles.Text).
		Width(60).
		Align(lipgloss.Center).
		Render("VibeMux helps you manage multiple Claude Code instances in a beautiful terminal interface.")

	hint := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true).
		Render("Press Enter to continue...")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		styledLogo,
		"",
		title,
		subtitle,
		"",
		desc,
		"",
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m Model) viewDetect() string {
	title := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render("🔍 Detecting Claude Installation")

	var statusContent string

	if m.detectedPath != "" {
		checkmark := lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Bold(true).
			Render("✓")

		pathStyle := lipgloss.NewStyle().
			Foreground(styles.Accent).
			Render(m.detectedPath)

		statusContent = lipgloss.JoinVertical(
			lipgloss.Center,
			checkmark+" Found Claude at:",
			"",
			pathStyle,
			"",
			lipgloss.NewStyle().Foreground(styles.TextMuted).Render("Press Enter to use this path"),
		)
	} else {
		warning := lipgloss.NewStyle().
			Foreground(styles.Warning).
			Bold(true).
			Render("⚠")

		statusContent = lipgloss.JoinVertical(
			lipgloss.Center,
			warning+" Claude not found in common locations",
			"",
			lipgloss.NewStyle().Foreground(styles.TextMuted).Render("Press Enter to configure manually"),
		)
	}

	hint := lipgloss.NewStyle().
		Foreground(styles.Overlay0).
		Render("Tip: Install Claude Code with: npm install -g @anthropic-ai/claude-code")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		"",
		statusContent,
		"",
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m Model) viewConfigure() string {
	title := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render("⚙️  Configure Claude Path")

	desc := lipgloss.NewStyle().
		Foreground(styles.Text).
		Width(60).
		Align(lipgloss.Center).
		Render("Enter the full path to your Claude executable:")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(0, 1).
		Render(m.claudeInput.View())

	var errorMsg string
	if m.error != "" {
		errorMsg = lipgloss.NewStyle().
			Foreground(styles.Danger).
			Bold(true).
			Render("❌ " + m.error)
	}

	examples := lipgloss.NewStyle().
		Foreground(styles.Overlay0).
		Render("Examples:\n" +
			"  • /usr/local/bin/claude\n" +
			"  • ~/.npm-global/bin/claude\n" +
			"  • /opt/homebrew/bin/claude")

	hint := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Render("Press Enter to confirm • Esc to go back")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		desc,
		"",
		inputBox,
		"",
		errorMsg,
		"",
		examples,
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m Model) viewProfileIntro() string {
	title := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render("⚙️  Configure Profiles")

	desc := lipgloss.NewStyle().
		Foreground(styles.Text).
		Width(70).
		Align(lipgloss.Center).
		Render("Profiles define how VibeMux launches agents using a command and env vars.")

	hint := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Render("Press Enter to create your first profile • Esc to skip")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		desc,
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m Model) viewProfileAddAnother() string {
	title := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true).
		Render("✅ Profile saved")

	countInfo := lipgloss.NewStyle().
		Foreground(styles.Accent).
		Render(fmt.Sprintf("Profiles configured: %d", m.profilesConfigured))

	hint := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Render("Press 'a' to add another • Enter to finish")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		countInfo,
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m Model) viewComplete() string {
	checkmark := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true).
		Render("✓")

	title := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true).
		Render("Setup Complete!")

	pathInfo := fmt.Sprintf("Claude path: %s", m.config.ClaudePath)
	pathStyle := lipgloss.NewStyle().
		Foreground(styles.Accent).
		Render(pathInfo)

	profileInfo := ""
	if m.profilesConfigured > 0 {
		profileInfo = lipgloss.NewStyle().
			Foreground(styles.TextMuted).
			Render(fmt.Sprintf("Profiles configured: %d", m.profilesConfigured))
	}

	hint := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render("Press Enter to start VibeMux...")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		checkmark,
		"",
		title,
		"",
		pathStyle,
		"",
		profileInfo,
		"",
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}
