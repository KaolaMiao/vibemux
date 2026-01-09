// Package profilelist provides the profile list UI component.
package profilelist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/ui/styles"
)

// Item represents a profile in the list.
type Item struct {
	Profile model.Profile
}

// Model is the profile list component.
type Model struct {
	items   []Item
	cursor  int
	focused bool
	width   int
	height  int
	offset  int
}

// New creates a new profile list component.
func New() Model {
	return Model{
		items: []Item{},
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

// SetProfiles updates the profile list.
func (m *Model) SetProfiles(profiles []model.Profile) {
	m.items = make([]Item, len(profiles))
	for i, p := range profiles {
		m.items[i] = Item{Profile: p}
	}
	if m.cursor >= len(m.items) && len(m.items) > 0 {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
}

// SelectedProfile returns the currently selected profile.
func (m Model) SelectedProfile() *model.Profile {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		p := m.items[m.cursor].Profile
		return &p
	}
	return nil
}

// HandleKey processes a key event.
func (m *Model) HandleKey(key string) bool {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}
		return true
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
			m.ensureVisible()
		}
		return true
	case "home", "g":
		m.cursor = 0
		m.offset = 0
		return true
	case "end", "G":
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
			m.ensureVisible()
		}
		return true
	}
	return false
}

// View renders the profile list.
func (m Model) View() string {
	innerWidth := m.width - 4
	innerHeight := m.height - 4

	icon := styles.PanelTitleIcon.Render(styles.IconProfile)
	title := "Profiles"
	if m.focused {
		title = styles.PanelTitleFocused.Render(title)
	} else {
		title = styles.PanelTitle.Render(title)
	}
	countStr := styles.ListItemDim.Render(fmt.Sprintf("(%d)", len(m.items)))
	header := icon + title + " " + countStr

	var rows []string
	if len(m.items) == 0 {
		emptyMsg := styles.TerminalPlaceholder.Render("No profiles yet")
		hint := styles.ListItemDim.Render("Press 'a' to add one")
		rows = append(rows, "", emptyMsg, hint)
	} else {
		visibleRows := innerHeight - 2
		if visibleRows < 1 {
			visibleRows = 1
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

		if len(m.items) > visibleRows {
			scrollInfo := fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.items))
			rows = append(rows, styles.ListItemDim.Render(scrollInfo))
		}
	}

	help := styles.ListItemDim.Render("Enter: edit - a: add - d: delete - s: default - c: settings - Esc: close")
	contentRows := append(rows, "", help)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		contentRows...,
	)

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

func (m *Model) renderItem(item Item, selected bool, maxWidth int) string {
	mark := styles.IconStarEmpty
	if item.Profile.IsDefault {
		mark = styles.IconStar
	}
	name := item.Profile.Name
	command := strings.TrimSpace(item.Profile.Command)
	if command == "" {
		command = "claude"
	}
	content := fmt.Sprintf("%s %s - %s", mark, name, command)
	content = styles.TruncateWithEllipsis(content, maxWidth)

	if selected {
		return styles.ListItemSelected.Render(content)
	}
	return styles.ListItem.Render(content)
}

func (m *Model) ensureVisible() {
	visibleRows := m.height - 6
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
