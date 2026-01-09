// Package terminal provides the terminal UI component.
package terminal

import (
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hinshun/vt10x"
	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/ui/styles"
)

const (
	attrReverse = 1 << iota
	attrUnderline
	attrBold
	attrGfx
	attrItalic
	attrBlink
	attrWrap
)

type ptyResponder struct {
	mu sync.RWMutex
	w  io.Writer
}

func (p *ptyResponder) Write(data []byte) (int, error) {
	p.mu.RLock()
	w := p.w
	p.mu.RUnlock()
	if w == nil {
		return len(data), nil
	}
	return w.Write(data)
}

func (p *ptyResponder) SetWriter(w io.Writer) {
	p.mu.Lock()
	p.w = w
	p.mu.Unlock()
}

// Model is the terminal component.
type Model struct {
	term         vt10x.Terminal
	responder    *ptyResponder
	focused      bool
	width        int
	height       int
	innerWidth   int
	innerHeight  int
	projectID    string
	projectName  string
	status       model.SessionStatus
	scrollback   []string
	scrollTail   string
	scrollOffset int
}

// New creates a new terminal component.
func New() Model {
	responder := &ptyResponder{}
	term := vt10x.New(vt10x.WithWriter(responder))
	return Model{
		term:      term,
		responder: responder,
		status:    model.SessionStatusIdle,
	}
}

// SetSize updates the component dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.width = width
	m.height = height
    // Reserve 1 column for scrollbar
	m.innerWidth = width - 5
	m.innerHeight = height - 6
	if m.innerWidth < 1 {
		m.innerWidth = 1
	}
	if m.innerHeight < 1 {
		m.innerHeight = 1
	}
	if m.term != nil {
		m.term.Resize(m.innerWidth, m.innerHeight)
	}
	m.clampScrollOffset()
}

// SetFocused updates the focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// IsFocused returns whether the component is focused.
func (m Model) IsFocused() bool {
	return m.focused
}

// SetProject sets the current project context.
func (m *Model) SetProject(id, name string) {
	if m.projectID != id {
		m.resetTerminal()
	}
	m.projectID = id
	m.projectName = name
}

// SetStatus updates the session status.
func (m *Model) SetStatus(status model.SessionStatus) {
	m.status = status
}

// BindWriter connects the terminal emulator to a PTY writer.
func (m *Model) BindWriter(w io.Writer) {
	if m.responder == nil {
		m.responder = &ptyResponder{}
	}
	m.responder.SetWriter(w)
}

// UnbindWriter detaches the PTY writer.
func (m *Model) UnbindWriter() {
	if m.responder == nil {
		return
	}
	m.responder.SetWriter(nil)
}

// ProjectID returns the current project ID.
func (m Model) ProjectID() string {
	return m.projectID
}

// Status returns the current session status.
func (m Model) Status() model.SessionStatus {
	return m.status
}

// IsScrolled returns whether the viewport is scrolled up (not at bottom).
func (m Model) IsScrolled() bool {
    return m.scrollOffset > 0
}

// AppendOutput feeds PTY output to the terminal emulator.
func (m *Model) AppendOutput(data []byte) {
	if len(data) == 0 || m.term == nil {
		return
	}
	_, _ = m.term.Write(data)
	m.appendScrollback(data)
}

// SetContent replaces the terminal content.
func (m *Model) SetContent(content string) {
	m.resetTerminal()
	if m.term != nil && content != "" {
		_, _ = m.term.Write([]byte(content))
	}
}

// Clear clears the terminal content.
func (m *Model) Clear() {
	m.resetTerminal()
}

// Width returns the terminal screen width.
func (m Model) Width() int {
	return m.innerWidth
}

// Height returns the terminal screen height.
func (m Model) Height() int {
	return m.innerHeight
}

// PTYSize returns the size for the PTY in columns and rows.
func (m Model) PTYSize() (cols, rows int) {
	return m.innerWidth, m.innerHeight
}

// HandleKey returns false to allow input to go to the PTY.
func (m *Model) HandleKey(key string) bool {
	switch key {
	case "pgup":
		m.scrollBy(m.innerHeight)
		return true
	case "pgdown":
		m.scrollBy(-m.innerHeight)
		return true
	case "shift+up":
		m.scrollBy(1)
		return true
	case "shift+down":
		m.scrollBy(-1)
		return true
	case "home":
		m.scrollOffset = m.maxScrollOffset()
		return true
	case "end":
		m.scrollOffset = 0
		return true
    case "esc":
        // Snap to bottom on Escape if scrolled
        if m.scrollOffset > 0 {
            m.scrollOffset = 0
            return true
        }
        return false
	}
	return false
}

// View renders the terminal panel.
func (m Model) View() string {
	innerWidth := m.innerWidth
	if innerWidth < 1 {
		innerWidth = m.width - 4
		if innerWidth < 1 {
			innerWidth = 1
		}
	}

	// Build header
	icon := m.statusIcon()
	title := "Terminal"
	if m.projectName != "" {
		title = m.projectName
	}

	if m.focused {
		title = styles.PanelTitleFocused.Render(title)
	} else {
		title = styles.PanelTitle.Render(title)
	}

	// Status info
	var statusInfo string
	switch m.status {
	case model.SessionStatusRunning:
		statusInfo = lipgloss.NewStyle().Foreground(styles.StatusRunning).Render("RUNNING")
	case model.SessionStatusStopped:
		statusInfo = lipgloss.NewStyle().Foreground(styles.StatusStopped).Render("STOPPED")
	case model.SessionStatusError:
		statusInfo = lipgloss.NewStyle().Foreground(styles.StatusError).Render("ERROR")
	default:
		statusInfo = lipgloss.NewStyle().Foreground(styles.StatusIdle).Render("IDLE")
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		icon,
		" ",
		title,
		"  ",
		statusInfo,
	)

	// Content
	var content string
	if m.projectID == "" {
		content = m.renderPlaceholder("Select a project and press Enter to start", innerWidth)
	} else if m.status == model.SessionStatusIdle {
		content = m.renderPlaceholder("Press Enter to start session", innerWidth)
	} else {
		content = m.renderScreen()
	}

	// Border style
	var borderStyle lipgloss.Style
	if m.focused {
		borderStyle = styles.FocusedBorderStyle
	} else {
		borderStyle = styles.BorderStyle
	}

	// Build panel
	// Build panel
    // Combine content with scrollbar
    mainArea := lipgloss.JoinHorizontal(lipgloss.Top, content, m.renderScrollbar())

	panel := borderStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			strings.Repeat("─", innerWidth),
			mainArea,
		))

	return panel
}

func (m *Model) renderScrollbar() string {
    height := m.innerHeight
    if height < 1 {
        return ""
    }
    
    // Total lines available (history + current screen height approximately)
    // We estimate total by scrollback lines + screen height.
    lines := m.renderScrollLines()
    totalLines := len(lines)
    if totalLines < height {
        totalLines = height
    }
    
    // Viewport start position (0 = top of history)
    // m.scrollOffset is distance from BOTTOM.
    // So top index = totalLines - height - m.scrollOffset
    
    // Simplification:
    // Pct = (total - height - offset) / (total - height)  [for top of thumb]
    
    if m.scrollOffset == 0 {
        // At bottom, full bar or just empty? 
        // Typically terminals show a bar at 100%.
        // Or we can hide it if no scrollback.
        if len(lines) <= height {
             return strings.Repeat(" ", height) 
        }
    }

    // Determine thumb size and position
    // thumbHeight / height = height / totalLines
    thumbHeight := int(float64(height) * float64(height) / float64(totalLines))
    if thumbHeight < 1 {
        thumbHeight = 1
    }
    
    // Position
    // maxScroll = totalLines - height
    // currentScroll = maxScroll - m.scrollOffset
    // pos = currentScroll / maxScroll * (height - thumbHeight)
    
    maxScroll := totalLines - height
    if maxScroll < 1 { 
         return strings.Repeat("│", height)
    }
    
    currentScroll := maxScroll - m.scrollOffset
    if currentScroll < 0 { currentScroll = 0 }
    if currentScroll > maxScroll { currentScroll = maxScroll }
    
    availableTrack := height - thumbHeight
    yPos := int(float64(currentScroll) / float64(maxScroll) * float64(availableTrack))
    
    var b strings.Builder
    for i := 0; i < height; i++ {
        if i >= yPos && i < yPos+thumbHeight {
            b.WriteString("█") // Thumb
        } else {
            b.WriteString("│") // Track
        }
        if i < height-1 {
            b.WriteByte('\n')
        }
    }
    
    return lipgloss.NewStyle().Foreground(styles.StatusIdle).Render(b.String())
}

func (m *Model) renderScreen() string {
	if m.scrollOffset > 0 {
		return m.renderScrollback()
	}
	if m.term == nil || m.innerWidth < 1 || m.innerHeight < 1 {
		return ""
	}
	m.term.Lock()
	defer m.term.Unlock()

	cursor := m.term.Cursor()
	showCursor := m.focused && m.term.CursorVisible()

	var b strings.Builder
	b.Grow((m.innerWidth + 1) * m.innerHeight)

	var prev cellStyle
	hasPrev := false

	for y := 0; y < m.innerHeight; y++ {
		for x := 0; x < m.innerWidth; x++ {
			cell := m.term.Cell(x, y)
			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}

			style := cellStyleFromGlyph(cell)
			if showCursor && cursor.X == x && cursor.Y == y {
				style.reverse = true
			}

			if !hasPrev || !style.equals(prev) {
				b.WriteString(style.sgr())
				prev = style
				hasPrev = true
			}

			b.WriteRune(ch)
		}
		if y < m.innerHeight-1 {
			b.WriteByte('\n')
		}
	}

	if hasPrev {
		b.WriteString("\x1b[0m")
	}

	return lipgloss.NewStyle().
		Width(m.innerWidth).
		Height(m.innerHeight).
		Render(b.String())
}

func (m *Model) resetTerminal() {
	if m.responder == nil {
		m.responder = &ptyResponder{}
	}
	m.scrollback = nil
	m.scrollTail = ""
	m.scrollOffset = 0
	if m.innerWidth > 0 && m.innerHeight > 0 {
		m.term = vt10x.New(vt10x.WithWriter(m.responder), vt10x.WithSize(m.innerWidth, m.innerHeight))
		return
	}
	m.term = vt10x.New(vt10x.WithWriter(m.responder))
}

// statusIcon returns the status indicator icon.
func (m Model) statusIcon() string {
	var color lipgloss.Color
	switch m.status {
	case model.SessionStatusRunning:
		color = styles.StatusRunning
	case model.SessionStatusStopped:
		color = styles.StatusStopped
	case model.SessionStatusError:
		color = styles.StatusError
	default:
		color = styles.StatusIdle
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render("●")
}

// renderPlaceholder renders a centered placeholder message.
func (m Model) renderPlaceholder(msg string, width int) string {
	styled := styles.TerminalPlaceholder.Render(msg)
	height := m.innerHeight
	if height < 1 {
		height = 1
	}
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(styled)
}

func (m *Model) appendScrollback(data []byte) {
	plain := ansi.Strip(string(data))
	if plain == "" {
		return
	}
	var line strings.Builder
	line.WriteString(m.scrollTail)
	linesAdded := 0
	flushLine := func() {
		m.scrollback = append(m.scrollback, line.String())
		linesAdded++
		line.Reset()
	}

	for i := 0; i < len(plain); i++ {
		ch := plain[i]
		switch ch {
		case '\r':
			if i+1 < len(plain) && plain[i+1] == '\n' {
				flushLine()
				i++
				continue
			}
			// Carriage return: overwrite current line.
			line.Reset()
		case '\n':
			flushLine()
		default:
			line.WriteByte(ch)
		}
	}

	m.scrollTail = line.String()
	const maxScrollback = 2000
	if len(m.scrollback) > maxScrollback {
		drop := len(m.scrollback) - maxScrollback
		m.scrollback = m.scrollback[drop:]
	}
	if m.scrollOffset > 0 {
		m.scrollOffset += linesAdded
		m.clampScrollOffset()
	}
}

func (m *Model) renderScrollback() string {
	if m.innerWidth < 1 || m.innerHeight < 1 {
		return ""
	}
	lines := m.renderScrollLines()
	total := len(lines)
	if total == 0 {
		return ""
	}
	start := total - m.innerHeight - m.scrollOffset
	if start < 0 {
		start = 0
	}
	end := start + m.innerHeight
	if end > total {
		end = total
	}
	visible := lines[start:end]
	if len(visible) < m.innerHeight {
		padding := make([]string, m.innerHeight-len(visible))
		visible = append(visible, padding...)
	}
	return lipgloss.NewStyle().
		Width(m.innerWidth).
		Height(m.innerHeight).
		Render(strings.Join(visible, "\n"))
}

func (m *Model) renderScrollLines() []string {
	if m.innerWidth < 1 {
		return nil
	}
	if len(m.scrollback) == 0 && m.scrollTail == "" {
		return nil
	}
	raw := make([]string, 0, len(m.scrollback)+1)
	raw = append(raw, m.scrollback...)
	if m.scrollTail != "" {
		raw = append(raw, m.scrollTail)
	}
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if line == "" {
			lines = append(lines, "")
			continue
		}
		wrapped := ansi.Hardwrap(line, m.innerWidth, true)
		lines = append(lines, strings.Split(wrapped, "\n")...)
	}
	return lines
}

func (m *Model) maxScrollOffset() int {
	lines := m.renderScrollLines()
	if m.innerHeight <= 0 || len(lines) <= m.innerHeight {
		return 0
	}
	return len(lines) - m.innerHeight
}

func (m *Model) clampScrollOffset() {
	max := m.maxScrollOffset()
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	if m.scrollOffset > max {
		m.scrollOffset = max
	}
}

func (m *Model) scrollBy(delta int) {
	m.scrollOffset += delta
	m.clampScrollOffset()
}

type cellStyle struct {
	fg, bg    vt10x.Color
	bold      bool
	italic    bool
	underline bool
	blink     bool
	reverse   bool
}

func cellStyleFromGlyph(g vt10x.Glyph) cellStyle {
	return cellStyle{
		fg:        g.FG,
		bg:        g.BG,
		bold:      g.Mode&attrBold != 0,
		italic:    g.Mode&attrItalic != 0,
		underline: g.Mode&attrUnderline != 0,
		blink:     g.Mode&attrBlink != 0,
	}
}

func (s cellStyle) equals(other cellStyle) bool {
	return s.fg == other.fg &&
		s.bg == other.bg &&
		s.bold == other.bold &&
		s.italic == other.italic &&
		s.underline == other.underline &&
		s.blink == other.blink &&
		s.reverse == other.reverse
}

func (s cellStyle) sgr() string {
	codes := []string{"0"}

	if s.bold {
		codes = append(codes, "1")
	}
	if s.italic {
		codes = append(codes, "3")
	}
	if s.underline {
		codes = append(codes, "4")
	}
	if s.blink {
		codes = append(codes, "5")
	}
	if s.reverse {
		codes = append(codes, "7")
	}

	codes = append(codes, colorCode(true, s.fg))
	codes = append(codes, colorCode(false, s.bg))

	return "\x1b[" + strings.Join(codes, ";") + "m"
}

func colorCode(fg bool, c vt10x.Color) string {
	if fg {
		if c == vt10x.DefaultFG {
			return "39"
		}
	} else {
		if c == vt10x.DefaultBG {
			return "49"
		}
	}

	if c < 16 {
		return strconv.Itoa(ansiColorCode(fg, int(c)))
	}
	if c < 256 {
		prefix := "48;5;"
		if fg {
			prefix = "38;5;"
		}
		return prefix + strconv.Itoa(int(c))
	}
	if c < 1<<24 {
		r := (int(c) >> 16) & 0xff
		g := (int(c) >> 8) & 0xff
		b := int(c) & 0xff
		if fg {
			return "38;2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b)
		}
		return "48;2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b)
	}
	if fg {
		return "39"
	}
	return "49"
}

func ansiColorCode(fg bool, c int) int {
	if c < 8 {
		if fg {
			return 30 + c
		}
		return 40 + c
	}
	if fg {
		return 90 + (c - 8)
	}
	return 100 + (c - 8)
}
