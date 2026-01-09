// Package projectlist provides the project list UI component.
package projectlist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/ui/styles"
)

// Item represents a project in the list.
type Item struct {
	Project model.Project
	Running bool
}

// Model is the project list component.
type Model struct {
	items    []Item
	cursor   int
	focused  bool
	width    int
	height   int
	offset   int // For scrolling
	profiles map[string]string
}

// New creates a new project list component.
func New() Model {
	return Model{
		items:    []Item{},
		profiles: make(map[string]string),
	}
}

// SetSize updates the component dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused updates the focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// IsFocused returns whether the component is focused.
func (m Model) IsFocused() bool {
	return m.focused
}

// SetProjects updates the project list.
func (m *Model) SetProjects(projects []model.Project, runningIDs map[string]bool) {
	m.items = make([]Item, len(projects))
	for i, p := range projects {
		running := false
		if runningIDs != nil {
			running = runningIDs[p.ID]
		}
		m.items[i] = Item{Project: p, Running: running}
	}
}

// SetProfiles updates profile name lookup for details display.
func (m *Model) SetProfiles(profiles []model.Profile) {
	if m.profiles == nil {
		m.profiles = make(map[string]string)
	}
	for k := range m.profiles {
		delete(m.profiles, k)
	}
	for _, p := range profiles {
		m.profiles[p.ID] = p.Name
	}
}

// SetRunning updates the running state for a project.
func (m *Model) SetRunning(projectID string, running bool) {
	for i := range m.items {
		if m.items[i].Project.ID == projectID {
			m.items[i].Running = running
			return
		}
	}
}

// SelectedProject returns the currently selected project.
func (m Model) SelectedProject() *model.Project {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		p := m.items[m.cursor].Project
		return &p
	}
	return nil
}

// SelectedIndex returns the index of the selected item.
func (m Model) SelectedIndex() int {
	return m.cursor
}

// ItemCount returns the number of items.
func (m Model) ItemCount() int {
	return len(m.items)
}

// CursorUp moves cursor up.
func (m *Model) CursorUp() {
	if m.cursor > 0 {
		m.cursor--
		m.ensureVisible()
	}
}

// CursorDown moves cursor down.
func (m *Model) CursorDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
		m.ensureVisible()
	}
}

// ensureVisible adjusts scroll offset to keep cursor visible.
func (m *Model) ensureVisible() {
	visibleRows := m.height - 4 // Account for border, title, and padding
	if visibleRows < 1 {
		visibleRows = 1
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visibleRows {
		m.offset = m.cursor - visibleRows + 1
	}
}

// HandleKey processes a key event.
func (m *Model) HandleKey(key string) bool {
	switch key {
	case "up", "k":
		m.CursorUp()
		return true
	case "down", "j":
		m.CursorDown()
		return true
	case "home", "g":
		m.cursor = 0
		m.offset = 0
		return true
	case "end", "G":
		m.cursor = len(m.items) - 1
		m.ensureVisible()
		return true
	}
	return false
}

// View renders the project list.
func (m Model) View() string {
	// Calculate dimensions
	innerWidth := m.width - 4 // Border + padding
	innerHeight := m.height - 4
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Header
	icon := styles.PanelTitleIcon.Render("📁")
	title := "Projects"
	if m.focused {
		title = styles.PanelTitleFocused.Render(title)
	} else {
		title = styles.PanelTitle.Render(title)
	}
	countStr := styles.ListItemDim.Render(fmt.Sprintf("(%d)", len(m.items)))
	header := icon + title + " " + countStr

	// Build list content
	var rows []string
	detailHeight := 4
	showDetails := innerHeight >= detailHeight+2
	listArea := innerHeight
	if showDetails {
		listArea = innerHeight - detailHeight - 1
		if listArea < 1 {
			listArea = innerHeight
			showDetails = false
		}
	}

	if len(m.items) == 0 {
		emptyMsg := styles.TerminalPlaceholder.Render("No projects yet")
		hint := styles.ListItemDim.Render("Press 'a' to add one")
		rows = append(rows, "", emptyMsg, hint)
	} else {
		visibleRows := listArea
		if len(m.items) > listArea {
			visibleRows = listArea - 1
			if visibleRows < 1 {
				visibleRows = 1
			}
		}

		endIdx := m.offset + visibleRows
		if endIdx > len(m.items) {
			endIdx = len(m.items)
		}

		for i := m.offset; i < endIdx; i++ {
			item := m.items[i]
			row := m.renderItem(item, i == m.cursor, innerWidth-2)
			rows = append(rows, row)
		}

		// Scroll indicator
		if len(m.items) > visibleRows {
			scrollInfo := fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.items))
			rows = append(rows, styles.ListItemDim.Render(scrollInfo))
		}
	}

	listContent := lipgloss.NewStyle().
		Width(innerWidth).
		Height(listArea).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	content := listContent
	if showDetails {
		separator := strings.Repeat("─", innerWidth)
		details := m.renderDetails(innerWidth, detailHeight)
		content = lipgloss.JoinVertical(lipgloss.Left, listContent, separator, details)
	}

	// Build panel
	var borderStyle lipgloss.Style
	if m.focused {
		borderStyle = styles.FocusedBorderStyle
	} else {
		borderStyle = styles.BorderStyle
	}

	panel := borderStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			strings.Repeat("─", innerWidth),
			content,
		))

	return panel
}

// renderItem renders a single project item.
func (m Model) renderItem(item Item, selected bool, maxWidth int) string {
	// Status dot
	var dot string
	if item.Running {
		dot = lipgloss.NewStyle().Foreground(styles.StatusRunning).Render("● ")
	} else {
		dot = lipgloss.NewStyle().Foreground(styles.StatusIdle).Render("○ ")
	}

	// Name
	name := item.Project.DisplayName()
	if len(name) > maxWidth-8 {
		name = name[:maxWidth-11] + "..."
	}

	// Build row
	var rowStyle lipgloss.Style
	if selected {
		if m.focused {
			rowStyle = lipgloss.NewStyle().
				Foreground(styles.TextCol).
				Background(styles.SurfaceCol).
				Bold(true).
				Width(maxWidth).
				Padding(0, 1)
		} else {
			rowStyle = lipgloss.NewStyle().
				Foreground(styles.TextCol).
				Background(styles.Surface1).
				Width(maxWidth).
				Padding(0, 1)
		}
		// Selected indicator
		name = "› " + name
	} else {
		rowStyle = lipgloss.NewStyle().
			Foreground(styles.Subtext1).
			Width(maxWidth).
			Padding(0, 1)
		name = "  " + name
	}

	return rowStyle.Render(dot + name)
}

func (m Model) renderDetails(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	labelStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	valueStyle := lipgloss.NewStyle().Foreground(styles.TextCol)
	titleStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Bold(true)

	lines := []string{titleStyle.Render("Details")}

	var selected *Item
	if m.cursor >= 0 && m.cursor < len(m.items) {
		selected = &m.items[m.cursor]
	}

	if selected == nil {
		lines = append(lines, labelStyle.Render("No project selected"))
	} else {
		path := selected.Project.Path
		profileName := ""
		if m.profiles != nil {
			profileName = m.profiles[selected.Project.ProfileID]
		}
		if profileName == "" {
			if selected.Project.ProfileID != "" {
				profileName = selected.Project.ProfileID
			} else {
				profileName = "default"
			}
		}
		status := "IDLE"
		if selected.Running {
			status = "RUNNING"
		}

		lines = append(lines,
			renderDetailLine(labelStyle, valueStyle, "Path: ", path, width),
			renderDetailLine(labelStyle, valueStyle, "Profile: ", profileName, width),
			renderDetailLine(labelStyle, valueStyle, "Status: ", status, width),
		)
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func renderDetailLine(labelStyle, valueStyle lipgloss.Style, label, value string, width int) string {
	if width < 1 {
		return ""
	}
	labelRendered := labelStyle.Render(label)
	avail := width - lipgloss.Width(labelRendered)
	if avail < 0 {
		avail = 0
	}
	value = styles.TruncateWithEllipsis(value, avail)
	return labelRendered + valueStyle.Render(value)
}
