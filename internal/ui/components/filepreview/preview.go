package filepreview

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TickMsg time.Time

type Model struct {
	viewport viewport.Model
	filePath string
	content  string
	lastMod  time.Time
	width    int
	height   int
	active   bool
}

func New() Model {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	return Model{
		viewport: vp,
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w - 4
	m.viewport.Height = h - 4
}

func (m *Model) SetFile(path string) {
	m.filePath = path
	m.active = true
	m.lastMod = time.Time{} // Reset to force refresh
	m.content = "" // Clear cached content
	m.refreshFile()
}

func (m *Model) Deactivate() {
	m.active = false
}

func (m *Model) refreshFile() {
	if m.filePath == "" {
		m.viewport.SetContent("No file path specified.")
		return
	}

	info, err := os.Stat(m.filePath)
	if err != nil {
		m.viewport.SetContent(fmt.Sprintf("Waiting for file: %s\n\n(File will appear once agents start writing)", m.filePath))
		return
	}

	// Only update if modified
	if !info.ModTime().After(m.lastMod) && m.content != "" {
		return
	}

	content, err := os.ReadFile(m.filePath)
	if err != nil {
		m.viewport.SetContent("Error reading file: " + err.Error())
		return
	}

	if len(content) == 0 {
		m.viewport.SetContent("(File is empty - waiting for content)")
		return
	}

	m.content = string(content)
	m.lastMod = info.ModTime()
	
	// Preserve scroll position logic? 
	// Usually for a log/live view, we want to follow tail if we were at bottom.
	atBottom := m.viewport.AtBottom()
	
	m.viewport.SetContent(m.content)
	
	if atBottom {
		m.viewport.GotoBottom()
	}
}

func (m Model) Init() tea.Cmd {
	return m.tick()
}

func (m Model) tick() tea.Cmd {
	if !m.active {
		return nil
	}
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TickMsg:
		if m.active {
			m.refreshFile()
			cmds = append(cmds, m.tick())
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.viewport.LineUp(3)
		case "down", "j":
			m.viewport.LineDown(3)
		case "pgup":
			m.viewport.HalfViewUp()
		case "pgdown":
			m.viewport.HalfViewDown()
		case "home", "g":
			m.viewport.GotoTop()
		case "end", "G":
			m.viewport.GotoBottom()
		case "esc", "q":
			// Handled by parent
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.active {
		return ""
	}
	
	// Debug status line
	status := fmt.Sprintf("Path: %s | Content: %d bytes", m.filePath, len(m.content))
	statusLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render(status)
	
	title := fmt.Sprintf(" Live Preview: %s ", m.filePath)
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62")).
		Render(title)
	
	// Display viewport content directly with white text
	viewContent := m.viewport.View()
	if viewContent == "" {
		// Fallback: Show a message if viewport is empty
		viewContent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true).
			Render("(No content to display)")
	}
	
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Foreground(lipgloss.Color("255")).
		Width(m.width - 4).
		Height(m.height - 6).
		Padding(1, 2).
		Render(viewContent)
		
	return lipgloss.JoinVertical(lipgloss.Center, header, statusLine, contentBox)
}

func (m Model) IsActive() bool {
	return m.active
}
