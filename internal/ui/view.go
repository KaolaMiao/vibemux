package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazyvibe/vibemux/internal/ui/styles"
)

// View renders the entire application.
func (a App) View() string {
	if a.quitting {
		// Fancy goodbye message
		bye := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.Primary).
			Render("👋 Goodbye from VibeMux!")
		return lipgloss.NewStyle().
			Width(a.width).
			Height(a.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(bye)
	}

	if !a.ready {
		// Loading screen
		loading := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.Accent).
			Render("⚡ Loading VibeMux...")
		return lipgloss.NewStyle().
			Width(a.width).
			Height(a.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(loading)
	}

	if a.windowTooSmall() {
		msg := fmt.Sprintf("窗口太小，请至少 %dx%d（当前 %dx%d）", minAppWidth, minAppHeight, a.width, a.height)
		notice := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.Accent).
			Render(msg)
		return lipgloss.NewStyle().
			Width(a.width).
			Height(a.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(notice)
	}

	// Calculate layout
	leftWidth := a.width * 25 / 100
	if leftWidth < 20 {
		leftWidth = 20
	}
	if leftWidth > 40 {
		leftWidth = 40
	}
	rightWidth := a.width - leftWidth

	// Left panel: Project list
	leftPanel := a.projectList.View()

	// Right panel: Terminal grid
	rightPanel := a.renderTerminalGrid(rightWidth, a.height-1)

	// Combine panels
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		rightPanel,
	)

	// Status bar
	statusBar := a.statusBar.View()

	// Combine everything
	fullView := lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusBar,
	)

	// Overlay dialog if open
	if a.dialogMode != DialogNone {
		return a.renderWithDialog(fullView)
	}

	return fullView
}

// renderTerminalPlaceholder renders a placeholder terminal panel.
func (a App) renderTerminalPlaceholder(width int) string {
	height := a.height - 1

	msg := styles.TerminalPlaceholder.Render("Select a project to start a session")

	content := lipgloss.NewStyle().
		Width(width-4).
		Height(height-4).
		Align(lipgloss.Center, lipgloss.Center).
		Render(msg)

	return styles.BorderStyle.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

func (a App) renderTerminalGrid(width, height int) string {
	rowsCount, colsCount := a.gridActiveDims()
	if rowsCount == 0 || colsCount == 0 {
		return a.renderEmptyTerminalArea(width, height)
	}

	_, _, _, colWidths, rowHeights := a.gridLayout()
	ids := a.gridOrder()
	cellIndex := 0
	rows := make([]string, 0, rowsCount)

	for r := 0; r < rowsCount; r++ {
		cols := make([]string, 0, colsCount)
		cellHeight := 0
		if r < len(rowHeights) {
			cellHeight = rowHeights[r]
		}
		for c := 0; c < colsCount; c++ {
			cellWidth := 0
			if c < len(colWidths) {
				cellWidth = colWidths[c]
			}
			focused := a.focus == FocusTerminal && cellIndex == a.activePane
			if cellIndex < len(ids) {
				if inst, ok := a.terminals[ids[cellIndex]]; ok {
					inst.Terminal.SetFocused(focused)
					cols = append(cols, inst.Terminal.View())
				} else {
					cols = append(cols, a.renderEmptyPane(cellWidth, cellHeight, focused))
				}
			} else {
				cols = append(cols, a.renderEmptyPane(cellWidth, cellHeight, focused))
			}
			cellIndex++
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cols...))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (a App) renderEmptyPane(width, height int, focused bool) string {
	if width < 2 {
		width = 2
	}
	if height < 2 {
		height = 2
	}
	msg := styles.TerminalPlaceholder.Render("Empty pane")
	innerWidth := width - 4
	innerHeight := height - 4
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}
	content := lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render(msg)

	border := styles.BorderStyle
	if focused {
		border = styles.FocusedBorderStyle
	}

	return border.
		Width(width - 2).
		Height(height - 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, content))
}

// renderEmptyTerminalArea renders the terminal area when no sessions exist.
func (a App) renderEmptyTerminalArea(width, height int) string {
	// ASCII art logo
	logo := `
 ██╗   ██╗██╗██████╗ ███████╗███╗   ███╗██╗   ██╗██╗  ██╗
 ██║   ██║██║██╔══██╗██╔════╝████╗ ████║██║   ██║╚██╗██╔╝
 ██║   ██║██║██████╔╝█████╗  ██╔████╔██║██║   ██║ ╚███╔╝
 ╚██╗ ██╔╝██║██╔══██╗██╔══╝  ██║╚██╔╝██║██║   ██║ ██╔██╗
  ╚████╔╝ ██║██████╔╝███████╗██║ ╚═╝ ██║╚██████╔╝██╔╝ ██╗
   ╚═══╝  ╚═╝╚═════╝ ╚══════╝╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═╝`

	// Style the logo with gradient effect
	styledLogo := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render(logo)

	subtitle := lipgloss.NewStyle().
		Foreground(styles.Accent).
		Italic(true).
		Render("AI Agent Orchestration Terminal")

	hint1 := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Render("Select a project and press Enter to start a session")

	hint2 := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Render("or press 'a' to add a new project")

	version := lipgloss.NewStyle().
		Foreground(styles.Overlay0).
		Render("v0.1.0")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		styledLogo,
		"",
		subtitle,
		"",
		"",
		hint1,
		hint2,
		"",
		version,
	)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Background(styles.Background).
		Render(content)
}

// renderWithDialog overlays a dialog on top of the main view.
func (a App) renderWithDialog(_ string) string {
	// Render dialog
	var dialogView string
	switch a.dialogMode {
	case DialogAddProject:
		dialogView = a.addDialog.View()
	case DialogManageProfiles:
		width, height := a.profileManagerSize()
		a.profileList.SetSize(width, height)
		dialogView = a.profileList.View()
	case DialogEditProfile:
		dialogView = a.profileDialog.View()
	case DialogSettings:
		dialogView = a.settingsDialog.View()
	case DialogCommand:
		dialogView = a.commandDialog.View()
	}

	// Overlay dialog in center
	return lipgloss.Place(
		a.width,
		a.height,
		lipgloss.Center,
		lipgloss.Center,
		dialogView,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#00000000")),
	)
}
