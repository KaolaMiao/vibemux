// Package chaindialog provides a dialog component for viewing Chain Context.
package chaindialog

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazyvibe/vibemux/internal/runtime"
)

// Model is the Chain Preview dialog component.
type Model struct {
	context      *runtime.ChainContext
	width        int
	height       int
	scrollOffset int
	closed       bool
	cleared      bool
}

// ChainDialogStyles defines the visual appearance.
type ChainDialogStyles struct {
	Box           lipgloss.Style
	Title         lipgloss.Style
	TaskLabel     lipgloss.Style
	EntryHeader   lipgloss.Style
	EntryContent  lipgloss.Style
	Timestamp     lipgloss.Style
	Help          lipgloss.Style
	Scrollbar     lipgloss.Style
	EmptyMessage  lipgloss.Style
}

// DefaultStyles returns the default styles for the dialog.
func DefaultStyles() ChainDialogStyles {
	purple := lipgloss.Color("#7C3AED")
	cyan := lipgloss.Color("#06B6D4")
	pink := lipgloss.Color("#EC4899")
	surface := lipgloss.Color("#1E1E2E")
	text := lipgloss.Color("#CDD6F4")
	textMuted := lipgloss.Color("#6C7086")
	green := lipgloss.Color("#A6E3A1")

	return ChainDialogStyles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Background(surface).
			Padding(1, 2),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan).
			Background(surface).
			Padding(0, 1),

		TaskLabel: lipgloss.NewStyle().
			Foreground(pink).
			Bold(true),

		EntryHeader: lipgloss.NewStyle().
			Foreground(green).
			Bold(true),

		EntryContent: lipgloss.NewStyle().
			Foreground(text).
			PaddingLeft(2),

		Timestamp: lipgloss.NewStyle().
			Foreground(textMuted).
			Italic(true),

		Help: lipgloss.NewStyle().
			Foreground(textMuted).
			MarginTop(1),

		Scrollbar: lipgloss.NewStyle().
			Foreground(textMuted),

		EmptyMessage: lipgloss.NewStyle().
			Foreground(textMuted).
			Italic(true).
			Align(lipgloss.Center),
	}
}

// New creates a new Chain Preview dialog.
func New(ctx *runtime.ChainContext) Model {
	return Model{
		context: ctx,
	}
}

// SetContext updates the chain context being displayed.
func (m *Model) SetContext(ctx *runtime.ChainContext) {
	m.context = ctx
	m.scrollOffset = 0
}

// SetSize updates the dialog dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles input for the dialog.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.closed = true
			return m, nil

		case "c", "C":
			// Clear chain context
			m.cleared = true
			m.closed = true
			return m, nil

		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
			return m, nil

		case "down", "j":
			m.scrollOffset++
			m.clampScroll()
			return m, nil

		case "pgup":
			m.scrollOffset -= 5
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return m, nil

		case "pgdown":
			m.scrollOffset += 5
			m.clampScroll()
			return m, nil

		case "home":
			m.scrollOffset = 0
			return m, nil

		case "end":
			m.scrollOffset = m.maxScroll()
			return m, nil
		}
	}
	return m, nil
}

// View renders the dialog.
func (m Model) View() string {
	styles := DefaultStyles()

	// Calculate inner dimensions
	innerWidth := m.width - 10
	if innerWidth < 30 {
		innerWidth = 30
	}
	innerHeight := m.height - 12
	if innerHeight < 5 {
		innerHeight = 5
	}

	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("📋 Chain Context Preview"))
	b.WriteString("\n\n")

	// Task description
	if m.context != nil && m.context.Task != "" {
		b.WriteString(styles.TaskLabel.Render("Task: "))
		b.WriteString(truncateStr(m.context.Task, innerWidth-10))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", innerWidth))
		b.WriteString("\n\n")
	}

	// Entries
	if m.context == nil || len(m.context.Chain) == 0 {
		emptyMsg := styles.EmptyMessage.
			Width(innerWidth).
			Height(innerHeight - 4).
			Render("No chain entries yet.\n\nUse Ctrl+S in Chain Mode to save context.")
		b.WriteString(emptyMsg)
	} else {
		// Build all entry lines
		var entryLines []string
		for i, entry := range m.context.Chain {
			// Header: Agent name + timestamp
			header := fmt.Sprintf("%d. [%s] %s",
				i+1,
				entry.Agent,
				entry.Timestamp.Format(time.TimeOnly))
			entryLines = append(entryLines, styles.EntryHeader.Render(header))

			// Content: truncated conclusion
			content := truncateStr(entry.Conclusion, innerWidth*2)
			content = strings.ReplaceAll(content, "\n", " ")
			wrapped := wrapText(content, innerWidth-4)
			for _, line := range strings.Split(wrapped, "\n") {
				entryLines = append(entryLines, styles.EntryContent.Render(line))
			}
			entryLines = append(entryLines, "") // Empty line between entries
		}

		// Apply scroll
		start := m.scrollOffset
		if start > len(entryLines) {
			start = len(entryLines)
		}
		end := start + innerHeight - 4
		if end > len(entryLines) {
			end = len(entryLines)
		}
		visible := entryLines[start:end]

		for _, line := range visible {
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(entryLines) > innerHeight-4 {
			progress := float64(m.scrollOffset) / float64(m.maxScroll())
			if progress > 1 {
				progress = 1
			}
			indicator := fmt.Sprintf("[%.0f%%]", progress*100)
			b.WriteString(styles.Scrollbar.Render(indicator))
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", innerWidth))
	b.WriteString("\n")
	entryCount := 0
	if m.context != nil {
		entryCount = len(m.context.Chain)
	}
	footer := fmt.Sprintf("Entries: %d", entryCount)
	b.WriteString(styles.Timestamp.Render(footer))
	b.WriteString("\n\n")

	// Help
	b.WriteString(styles.Help.Render("[C] Clear  [↑/↓] Scroll  [Esc] Close"))

	// Wrap in box
	content := styles.Box.Width(innerWidth + 4).Render(b.String())

	// Center in screen
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			content,
		)
	}
	return content
}

// IsClosed returns true if the dialog was closed.
func (m Model) IsClosed() bool {
	return m.closed
}

// IsCleared returns true if the user requested to clear the chain.
func (m Model) IsCleared() bool {
	return m.cleared
}

// Reset resets the dialog state.
func (m *Model) Reset() {
	m.closed = false
	m.cleared = false
	m.scrollOffset = 0
}

// clampScroll ensures scroll offset is within bounds.
func (m *Model) clampScroll() {
	max := m.maxScroll()
	if m.scrollOffset > max {
		m.scrollOffset = max
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// maxScroll returns the maximum scroll offset.
func (m Model) maxScroll() int {
	if m.context == nil {
		return 0
	}
	// Estimate total lines
	totalLines := len(m.context.Chain) * 4 // rough estimate
	viewHeight := m.height - 16
	if viewHeight < 1 {
		viewHeight = 1
	}
	max := totalLines - viewHeight
	if max < 0 {
		return 0
	}
	return max
}

// Helper functions

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func wrapText(s string, width int) string {
	if width < 1 {
		return s
	}
	var result strings.Builder
	line := ""
	for _, word := range strings.Fields(s) {
		if len(line)+len(word)+1 > width {
			if line != "" {
				result.WriteString(line)
				result.WriteString("\n")
			}
			line = word
		} else {
			if line != "" {
				line += " "
			}
			line += word
		}
	}
	if line != "" {
		result.WriteString(line)
	}
	return result.String()
}
