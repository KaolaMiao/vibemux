// Package styles defines the visual appearance for VibeMux TUI.
// Using Catppuccin Mocha color palette for a modern, aesthetic look.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Mocha color palette
var (
	// Base colors
	Rosewater = lipgloss.Color("#F5E0DC")
	Flamingo  = lipgloss.Color("#F2CDCD")
	Pink      = lipgloss.Color("#F5C2E7")
	Mauve     = lipgloss.Color("#CBA6F7")
	Red       = lipgloss.Color("#F38BA8")
	Maroon    = lipgloss.Color("#EBA0AC")
	Peach     = lipgloss.Color("#FAB387")
	Yellow    = lipgloss.Color("#F9E2AF")
	Green     = lipgloss.Color("#A6E3A1")
	Teal      = lipgloss.Color("#94E2D5")
	Sky       = lipgloss.Color("#89DCEB")
	Sapphire  = lipgloss.Color("#74C7EC")
	Blue      = lipgloss.Color("#89B4FA")
	Lavender  = lipgloss.Color("#B4BEFE")

	// Surface colors
	Text     = lipgloss.Color("#CDD6F4")
	Subtext1 = lipgloss.Color("#BAC2DE")
	Subtext0 = lipgloss.Color("#A6ADC8")
	Overlay2 = lipgloss.Color("#9399B2")
	Overlay1 = lipgloss.Color("#7F849C")
	Overlay0 = lipgloss.Color("#6C7086")
	Surface2 = lipgloss.Color("#585B70")
	Surface1 = lipgloss.Color("#45475A")
	Surface0 = lipgloss.Color("#313244")
	Base     = lipgloss.Color("#1E1E2E")
	Mantle   = lipgloss.Color("#181825")
	Crust    = lipgloss.Color("#11111B")
)

// Semantic colors (using the palette)
var (
	Primary     = Mauve
	Secondary   = Green
	Accent      = Sapphire
	Danger      = Red
	Warning     = Peach
	Success     = Green
	Info        = Blue
	Muted       = Overlay0
	Background  = Base
	SurfaceCol  = Surface0
	TextCol     = Text
	TextMuted   = Subtext0
	Border      = Surface1
	BorderFocus = Mauve
)

// Session status colors
var (
	StatusRunning = Green
	StatusIdle    = Overlay0
	StatusStopped = Yellow
	StatusError   = Red
)

// Gradient effects (simulated with patterns)
var (
	GradientPurple = []lipgloss.Color{Mauve, Pink, Lavender}
	GradientCyan   = []lipgloss.Color{Teal, Sky, Sapphire}
	GradientWarm   = []lipgloss.Color{Peach, Yellow, Rosewater}
)

// Base styles
var (
	// BaseStyle is applied to the entire application
	BaseStyle = lipgloss.NewStyle().
			Background(Background)

	// BorderStyle for panels
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border)

	// FocusedBorderStyle for focused panels
	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(BorderFocus)

	// GlowBorder for highlighted panels
	GlowBorder = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(Sapphire)
)

// Panel styles
var (
	// PanelTitle for panel headers
	PanelTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextCol).
			Padding(0, 1)

	// PanelTitleFocused for focused panel headers
	PanelTitleFocused = lipgloss.NewStyle().
				Bold(true).
				Foreground(Primary).
				Padding(0, 1)

	// PanelTitleIcon for icon prefix
	PanelTitleIcon = lipgloss.NewStyle().
			Foreground(Accent).
			MarginRight(1)
)

// List item styles
var (
	// ListItem for normal list items
	ListItem = lipgloss.NewStyle().
			Foreground(TextCol).
			Padding(0, 1)

	// ListItemSelected for selected list items
	ListItemSelected = lipgloss.NewStyle().
				Foreground(TextCol).
				Background(SurfaceCol).
				Bold(true).
				Padding(0, 1)

	// ListItemDim for inactive/dimmed items
	ListItemDim = lipgloss.NewStyle().
			Foreground(TextMuted).
			Padding(0, 1)

	// ListItemHighlight for highlighted items
	ListItemHighlight = lipgloss.NewStyle().
				Foreground(Accent).
				Bold(true).
				Padding(0, 1)
)

// Status indicator styles
var (
	StatusIndicator = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	StatusRunningStyle = lipgloss.NewStyle().
				Foreground(StatusRunning).
				Bold(true)

	StatusIdleStyle = lipgloss.NewStyle().
			Foreground(StatusIdle)

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(StatusError).
				Bold(true)
)

// StatusBar styles
var (
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			Background(Mantle).
			Padding(0, 1)

	StatusBarKey = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	StatusBarDesc = lipgloss.NewStyle().
			Foreground(TextMuted)

	StatusBarSeparator = lipgloss.NewStyle().
				Foreground(Overlay0).
				SetString(" │ ")

	StatusBarBrand = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)
)

// Terminal styles
var (
	TerminalStyle = lipgloss.NewStyle().
			Foreground(TextCol)

	TerminalPlaceholder = lipgloss.NewStyle().
				Foreground(TextMuted).
				Italic(true)

	TerminalHeader = lipgloss.NewStyle().
			Background(Surface0).
			Foreground(TextCol).
			Bold(true).
			Padding(0, 1)
)

// Dialog styles
var (
	DialogBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2).
			Background(SurfaceCol)

	DialogTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextCol).
			MarginBottom(1)

	DialogButton = lipgloss.NewStyle().
			Foreground(TextCol).
			Background(SurfaceCol).
			Padding(0, 2).
			MarginRight(1)

	DialogButtonActive = lipgloss.NewStyle().
				Foreground(TextCol).
				Background(Primary).
				Bold(true).
				Padding(0, 2).
				MarginRight(1)
)

// Logo and branding styles
var (
	LogoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	VersionStyle = lipgloss.NewStyle().
			Foreground(Overlay0)
)

// Helper functions

// StatusColor returns the color for a session status.
func StatusColor(status string) lipgloss.Color {
	switch status {
	case "running":
		return StatusRunning
	case "stopped":
		return StatusStopped
	case "error":
		return StatusError
	default:
		return StatusIdle
	}
}

// RenderStatusDot returns a colored status indicator.
func RenderStatusDot(running bool) string {
	if running {
		return lipgloss.NewStyle().Foreground(StatusRunning).Render("●")
	}
	return lipgloss.NewStyle().Foreground(StatusIdle).Render("○")
}

// RenderStatusDotWithStatus returns a colored status dot based on status.
func RenderStatusDotWithStatus(status string) string {
	color := StatusColor(status)
	return lipgloss.NewStyle().Foreground(color).Render("●")
}

// TruncateWithEllipsis truncates a string to maxLen with ellipsis.
func TruncateWithEllipsis(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Spinner frames for animated loading
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Icons
var (
	IconProject   = "📁"
	IconProfile   = "⚙️"
	IconTerminal  = "💻"
	IconRunning   = "▶️"
	IconStopped   = "⏹️"
	IconError     = "❌"
	IconSuccess   = "✅"
	IconWarning   = "⚠️"
	IconInfo      = "ℹ️"
	IconAdd       = "➕"
	IconDelete    = "🗑️"
	IconEdit      = "✏️"
	IconClose     = "✕"
	IconDot       = "●"
	IconDotEmpty  = "○"
	IconArrowR    = "→"
	IconArrowL    = "←"
	IconStar      = "★"
	IconStarEmpty = "☆"
)

// Box drawing characters for custom borders
var (
	BoxTopLeft     = "╭"
	BoxTopRight    = "╮"
	BoxBottomLeft  = "╰"
	BoxBottomRight = "╯"
	BoxHorizontal  = "─"
	BoxVertical    = "│"
	BoxTeeLeft     = "├"
	BoxTeeRight    = "┤"
	BoxTeeTop      = "┬"
	BoxTeeBottom   = "┴"
	BoxCross       = "┼"
)

// Fancy header style with gradient effect simulation
func RenderFancyHeader(title string, width int) string {
	// Create a fancy header with decorative elements
	left := lipgloss.NewStyle().Foreground(Mauve).Render("╭─")
	right := lipgloss.NewStyle().Foreground(Mauve).Render("─╮")
	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(TextCol).
		Background(Surface0).
		Padding(0, 1).
		Render(title)

	titleWidth := lipgloss.Width(titleStyled)
	leftDecor := lipgloss.Width(left)
	rightDecor := lipgloss.Width(right)
	fillWidth := width - titleWidth - leftDecor - rightDecor

	if fillWidth < 0 {
		fillWidth = 0
	}

	leftFill := fillWidth / 2
	rightFill := fillWidth - leftFill

	leftLine := lipgloss.NewStyle().Foreground(Surface1).Render(repeatChar("─", leftFill))
	rightLine := lipgloss.NewStyle().Foreground(Surface1).Render(repeatChar("─", rightFill))

	return left + leftLine + titleStyled + rightLine + right
}

// repeatChar repeats a character n times.
func repeatChar(char string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += char
	}
	return result
}
