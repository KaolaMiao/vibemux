// Package statusbar provides the status bar UI component.
package statusbar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazyvibe/vibemux/internal/ui/keys"
	"github.com/lazyvibe/vibemux/internal/ui/styles"
)

// Model is the status bar component.
type Model struct {
	width        int
	message      string
	isError      bool
	keyMap       keys.KeyMap
	sessionCount int
	modeLabel    string
}

// New creates a new status bar component.
func New() Model {
	return Model{
		keyMap: keys.DefaultKeyMap(),
	}
}

// SetWidth updates the status bar width.
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetMessage sets a temporary message.
func (m *Model) SetMessage(msg string, isError bool) {
	m.message = msg
	m.isError = isError
}

// ClearMessage clears the temporary message.
func (m *Model) ClearMessage() {
	m.message = ""
	m.isError = false
}

// SetSessionCount updates the active session count.
func (m *Model) SetSessionCount(count int) {
	m.sessionCount = count
}

// SetModeLabel updates the current input mode label.
func (m *Model) SetModeLabel(label string) {
	m.modeLabel = strings.ToUpper(strings.TrimSpace(label))
}

// View renders the status bar.
func (m Model) View() string {
	// Brand
	brand := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render(" VibeMux ")

	modeLabel := m.modeLabel
	if modeLabel == "" {
		modeLabel = "CTRL"
	}
	modeBadge := lipgloss.NewStyle().
		Foreground(styles.Base).
		Background(styles.Accent).
		Bold(true).
		Padding(0, 1).
		Render(modeLabel)

	// Build help text
	helpItems := []string{}
	if modeLabel == "TERM" {
		helpItems = append(helpItems, m.renderKey("F12", "control"))
	} else {
		helpItems = append(helpItems,
			m.renderKey("F12", "term"),
			m.renderKey("Tab", "switch"),
			m.renderKey("Enter", "run/term"),
			m.renderKey("a", "add"),
			m.renderKey("p", "profiles"),
			m.renderKey("d", "delete"),
			m.renderKey("x", "close"),
			m.renderKey("←/→", "pane"),
			m.renderKey("↑/↓", "pane"),
			m.renderKey("Shift+↑/↓", "page"),
			m.renderKey("q", "quit"),
		)
	}
	help := strings.Join(helpItems, " ")

	// Session count indicator
	sessionInfo := ""
	if m.sessionCount > 0 {
		sessionInfo = lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Render(fmt.Sprintf(" ● %d sessions ", m.sessionCount))
	}

	// Message area
	var msgArea string
	if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		if m.isError {
			msgStyle = lipgloss.NewStyle().Foreground(styles.Danger).Bold(true)
		}
		msgArea = msgStyle.Render(" " + m.message + " ")
	}

	// Calculate spacing
	leftContent := brand + modeBadge + sessionInfo
	rightContent := help
	middleContent := msgArea

	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	middleWidth := lipgloss.Width(middleContent)

	// Calculate padding
	totalUsed := leftWidth + rightWidth + middleWidth
	padding := m.width - totalUsed
	if padding < 0 {
		padding = 0
	}

	leftPad := padding / 2
	rightPad := padding - leftPad

	// Build status bar
	content := leftContent +
		strings.Repeat(" ", leftPad) +
		middleContent +
		strings.Repeat(" ", rightPad) +
		rightContent

	return lipgloss.NewStyle().
		Background(styles.Mantle).
		Foreground(styles.TextMuted).
		Width(m.width).
		Render(content)
}

// renderKey renders a key binding hint.
func (m Model) renderKey(key, desc string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Accent).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(styles.Overlay0)
	return keyStyle.Render(key) + descStyle.Render(":"+desc)
}
