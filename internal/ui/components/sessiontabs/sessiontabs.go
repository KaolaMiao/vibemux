// Package sessiontabs provides the session tab bar component for multi-instance display.
package sessiontabs

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazyvibe/vibemux/internal/model"
)

// Tab represents a single session tab.
type Tab struct {
	ID       string
	Name     string
	Status   model.SessionStatus
	HasNew   bool // Has new unread output
	IsActive bool
}

// Model is the session tabs component.
type Model struct {
	tabs        []Tab
	activeIndex int
	offset      int
	width       int
	focused     bool
	styles      TabStyles
}

// TabStyles defines the visual appearance of tabs.
type TabStyles struct {
	Container     lipgloss.Style
	Tab           lipgloss.Style
	TabActive     lipgloss.Style
	TabHasNew     lipgloss.Style
	StatusDot     lipgloss.Style
	StatusRunning lipgloss.Color
	StatusIdle    lipgloss.Color
	StatusStopped lipgloss.Color
	StatusError   lipgloss.Color
	CloseBtn      lipgloss.Style
	CloseBtnHover lipgloss.Style
}

// DefaultTabStyles returns beautiful tab styles.
func DefaultTabStyles() TabStyles {
	purple := lipgloss.Color("#7C3AED")
	cyan := lipgloss.Color("#06B6D4")
	pink := lipgloss.Color("#EC4899")
	green := lipgloss.Color("#10B981")
	amber := lipgloss.Color("#F59E0B")
	red := lipgloss.Color("#EF4444")
	surface := lipgloss.Color("#1E1E2E")
	surfaceLight := lipgloss.Color("#313244")
	text := lipgloss.Color("#CDD6F4")
	textMuted := lipgloss.Color("#6C7086")

	return TabStyles{
		Container: lipgloss.NewStyle().
			Background(surface).
			Padding(0, 1),

		Tab: lipgloss.NewStyle().
			Foreground(textMuted).
			Background(surfaceLight).
			Padding(0, 2).
			MarginRight(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(surfaceLight),

		TabActive: lipgloss.NewStyle().
			Foreground(text).
			Background(surface).
			Bold(true).
			Padding(0, 2).
			MarginRight(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple),

		TabHasNew: lipgloss.NewStyle().
			Foreground(cyan).
			Background(surfaceLight).
			Padding(0, 2).
			MarginRight(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(pink),

		StatusDot: lipgloss.NewStyle().
			Bold(true),

		StatusRunning: green,
		StatusIdle:    textMuted,
		StatusStopped: amber,
		StatusError:   red,

		CloseBtn: lipgloss.NewStyle().
			Foreground(textMuted).
			MarginLeft(1),

		CloseBtnHover: lipgloss.NewStyle().
			Foreground(red).
			Bold(true).
			MarginLeft(1),
	}
}

// New creates a new session tabs component.
func New() Model {
	return Model{
		tabs:   []Tab{},
		offset: 0,
		styles: DefaultTabStyles(),
	}
}

// SetWidth sets the component width.
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetFocused sets the focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// AddTab adds a new tab.
func (m *Model) AddTab(id, name string, status model.SessionStatus) {
	// Check if tab already exists
	for i, t := range m.tabs {
		if t.ID == id {
			m.tabs[i].Status = status
			return
		}
	}

	m.tabs = append(m.tabs, Tab{
		ID:     id,
		Name:   name,
		Status: status,
	})
}

// RemoveTab removes a tab by ID.
func (m *Model) RemoveTab(id string) {
	for i, t := range m.tabs {
		if t.ID == id {
			m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
			if m.activeIndex >= len(m.tabs) && m.activeIndex > 0 {
				m.activeIndex--
			}
			if m.offset >= len(m.tabs) && m.offset > 0 {
				m.offset--
			}
			return
		}
	}
}

// SetActiveTab sets the active tab by ID.
func (m *Model) SetActiveTab(id string) {
	for i, t := range m.tabs {
		if t.ID == id {
			m.activeIndex = i
			m.tabs[i].HasNew = false
			return
		}
	}
}

// SetTabStatus updates a tab's status.
func (m *Model) SetTabStatus(id string, status model.SessionStatus) {
	for i, t := range m.tabs {
		if t.ID == id {
			m.tabs[i].Status = status
			return
		}
	}
}

// MarkTabHasNew marks a tab as having new output.
func (m *Model) MarkTabHasNew(id string) {
	for i, t := range m.tabs {
		if t.ID == id && i != m.activeIndex {
			m.tabs[i].HasNew = true
			return
		}
	}
}

// ActiveTab returns the currently active tab.
func (m Model) ActiveTab() *Tab {
	if m.activeIndex >= 0 && m.activeIndex < len(m.tabs) {
		t := m.tabs[m.activeIndex]
		return &t
	}
	return nil
}

// ActiveID returns the ID of the active tab.
func (m Model) ActiveID() string {
	if t := m.ActiveTab(); t != nil {
		return t.ID
	}
	return ""
}

// TabCount returns the number of tabs.
func (m Model) TabCount() int {
	return len(m.tabs)
}

// NextTab switches to the next tab.
func (m *Model) NextTab() {
	if len(m.tabs) == 0 {
		return
	}
	m.activeIndex = (m.activeIndex + 1) % len(m.tabs)
	m.tabs[m.activeIndex].HasNew = false
}

// PrevTab switches to the previous tab.
func (m *Model) PrevTab() {
	if len(m.tabs) == 0 {
		return
	}
	m.activeIndex--
	if m.activeIndex < 0 {
		m.activeIndex = len(m.tabs) - 1
	}
	m.tabs[m.activeIndex].HasNew = false
}

// View renders the session tabs.
func (m *Model) View() string {
	if len(m.tabs) == 0 {
		return ""
	}

	rendered := make([]string, 0, len(m.tabs))
	widths := make([]int, 0, len(m.tabs))

	for i, t := range m.tabs {
		// Status dot
		var dotColor lipgloss.Color
		switch t.Status {
		case model.SessionStatusRunning:
			dotColor = m.styles.StatusRunning
		case model.SessionStatusStopped:
			dotColor = m.styles.StatusStopped
		case model.SessionStatusError:
			dotColor = m.styles.StatusError
		default:
			dotColor = m.styles.StatusIdle
		}
		dot := m.styles.StatusDot.Foreground(dotColor).Render("●")

		// Tab name (truncate if needed)
		name := t.Name
		if len(name) > 12 {
			name = name[:10] + "…"
		}

		// Index indicator
		indexStr := fmt.Sprintf("%d:", i+1)

		// Build tab content
		content := fmt.Sprintf("%s %s %s", indexStr, dot, name)

		// Select style
		var tabStyle lipgloss.Style
		if i == m.activeIndex {
			tabStyle = m.styles.TabActive
		} else if t.HasNew {
			tabStyle = m.styles.TabHasNew
		} else {
			tabStyle = m.styles.Tab
		}

		tab := tabStyle.Render(content)
		rendered = append(rendered, tab)
		widths = append(widths, lipgloss.Width(tab))
	}

	start, end := m.visibleRange(widths)
	if start < 0 || end <= start {
		return m.styles.Container.Width(m.width).Render("")
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, rendered[start:end]...)
	return m.styles.Container.Width(m.width).Render(row)
}

// Tabs returns all tabs.
func (m Model) Tabs() []Tab {
	return m.tabs
}

// HasTabs returns true if there are any tabs.
func (m Model) HasTabs() bool {
	return len(m.tabs) > 0
}

func (m *Model) visibleRange(widths []int) (int, int) {
	if len(widths) == 0 {
		return 0, 0
	}
	if m.width <= 0 {
		return 0, 0
	}

	total := 0
	for _, w := range widths {
		total += w
	}
	if total <= m.width {
		m.offset = 0
		return 0, len(widths)
	}

	start := m.offset
	if start < 0 {
		start = 0
	}
	if start >= len(widths) {
		start = len(widths) - 1
	}

	end := m.fitFrom(start, widths)

	if m.activeIndex < start {
		start = m.activeIndex
		end = m.fitFrom(start, widths)
	} else if m.activeIndex >= end {
		start = m.shiftLeftToFit(m.activeIndex, widths)
		end = m.fitFrom(start, widths)
	}

	if start < 0 {
		start = 0
	}
	if end > len(widths) {
		end = len(widths)
	}
	m.offset = start
	return start, end
}

func (m *Model) fitFrom(start int, widths []int) int {
	if start < 0 {
		return 0
	}
	used := 0
	end := start
	for end < len(widths) && used+widths[end] <= m.width {
		used += widths[end]
		end++
	}
	return end
}

func (m *Model) shiftLeftToFit(active int, widths []int) int {
	if active < 0 {
		return 0
	}
	used := 0
	start := active
	for start >= 0 && used+widths[start] <= m.width {
		used += widths[start]
		start--
	}
	return start + 1
}
