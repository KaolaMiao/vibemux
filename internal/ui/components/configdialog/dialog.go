// Package configdialog provides a split-layout configuration dialog.
package configdialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputType defines the type of input field.
type InputType int

const (
	InputText InputType = iota
	InputTextArea
)

// Field represents a single input field in the dialog.
type Field struct {
	Label       string
	Placeholder string
	Value       string
	Type        InputType
	Column      int // 0 = Left (Global), 1 = Right (Terminals)
	Header      string // Optional Section Header
	
	// Grid Positioning for Right Column (Column 1)
	GridRow int
	GridCol int
}

// WrappedInput separates the common interface.
type WrappedInput interface {
	Update(tea.Msg) (WrappedInput, tea.Cmd)
	View() string
	Focus() (WrappedInput, tea.Cmd)
	Blur() WrappedInput
	SetValue(string) WrappedInput
	Value() string
}

// TextInputWrapper wraps textinput.Model
type TextInputWrapper struct {
	model textinput.Model
}

func (w TextInputWrapper) Update(msg tea.Msg) (WrappedInput, tea.Cmd) {
	var cmd tea.Cmd
	w.model, cmd = w.model.Update(msg)
	return w, cmd
}
func (w TextInputWrapper) View() string { return w.model.View() }
func (w TextInputWrapper) Focus() (WrappedInput, tea.Cmd) { 
	cmd := w.model.Focus()
	return w, cmd
}
func (w TextInputWrapper) Blur() WrappedInput { 
	w.model.Blur()
	return w
}
func (w TextInputWrapper) SetValue(s string) WrappedInput { 
	w.model.SetValue(s)
	return w
}
func (w TextInputWrapper) Value() string { return w.model.Value() }

// TextAreaWrapper wraps textarea.Model
type TextAreaWrapper struct {
	model textarea.Model
}

func (w TextAreaWrapper) Update(msg tea.Msg) (WrappedInput, tea.Cmd) {
	var cmd tea.Cmd
	w.model, cmd = w.model.Update(msg)
	return w, cmd
}
func (w TextAreaWrapper) View() string { return w.model.View() }
func (w TextAreaWrapper) Focus() (WrappedInput, tea.Cmd) { 
	cmd := w.model.Focus()
	return w, cmd
}
func (w TextAreaWrapper) Blur() WrappedInput { 
	w.model.Blur()
	return w
}
func (w TextAreaWrapper) SetValue(s string) WrappedInput { 
	w.model.SetValue(s) 
	return w
}
func (w TextAreaWrapper) Value() string { return w.model.Value() }


// Model is the specialized configuration dialog.
type Model struct {
	title      string
	inputs     []WrappedInput
	fields     []Field // Meta data
	
	focusIndex int
	width      int
	height     int
	submitted  bool
	cancelled  bool
	styles     Styles
}

// Styles defines the visual appearance.
type Styles struct {
	Overlay      lipgloss.Style
	Box          lipgloss.Style
	Title        lipgloss.Style
	Label        lipgloss.Style
	LabelFocused lipgloss.Style
	Header       lipgloss.Style
	Input        lipgloss.Style
	InputFocused lipgloss.Style
	Help         lipgloss.Style
	
	// Grid Cell Styles
	Cell        lipgloss.Style
	CellFocused lipgloss.Style
}

func DefaultStyles() Styles {
	purple := lipgloss.Color("#7C3AED")
	cyan := lipgloss.Color("#06B6D4")
	surface := lipgloss.Color("#1E1E2E")
	surfaceLight := lipgloss.Color("#313244")
	textMuted := lipgloss.Color("#6C7086")

	return Styles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Background(surface).
			Padding(1, 1), // Tighter padding

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan).
			Background(surface).
			Padding(0, 1).
			MarginBottom(1),

		Label: lipgloss.NewStyle().
			Foreground(textMuted).
			MarginBottom(0),

		LabelFocused: lipgloss.NewStyle().
			Foreground(purple).
			Bold(true).
			MarginBottom(0),
			
		Header: lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true).
			MarginBottom(0), // Removed top margin to fit better

		Input: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(surfaceLight).
			Padding(0, 1).
			MarginBottom(0), // Tighter

		InputFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(0, 1).
			MarginBottom(0),

		Help: lipgloss.NewStyle().
			Foreground(textMuted).
			MarginTop(1),
			
		Cell: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(surfaceLight).
			Padding(0, 1).
			MarginRight(1).
			MarginBottom(1),
			
		CellFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(purple).
			Padding(0, 1).
			MarginRight(1).
			MarginBottom(1),
	}
}

// New creates a new config dialog.
func New(title string, fields []Field) Model {
	inputs := make([]WrappedInput, len(fields))

	for i, f := range fields {
		if f.Type == InputTextArea {
			ta := textarea.New()
			ta.Placeholder = f.Placeholder
			// Use wrappers for initialization too to be consistent, but setting safely here
			
			// Layout Config
			if f.Column == 1 {
				// Compact size for grid
				ta.SetWidth(33) // Narrower for grid
				ta.SetHeight(4) // Shorter
			} else {
				ta.SetWidth(35)
				ta.SetHeight(5)
			}
			ta.CharLimit = 0
			ta.ShowLineNumbers = false
			
			inputs[i] = TextAreaWrapper{model: ta}
			inputs[i] = inputs[i].SetValue(f.Value)
			
			if i == 0 {
				var cmd tea.Cmd
				inputs[i], cmd = inputs[i].Focus()
				// We drop the cmd here as it's usually just blink, but ideally we'd return it. 
				// Since we can't return cmd from New, we rely on Update to tick?
				// Actually textinput blink needs the command. 
				// But we are in New(). The model loop will pick it up? 
				// Often we need to return the Init cmd.
				// For now let's just hold the state.
				_ = cmd 
			}
		} else {
			ti := textinput.New()
			ti.Placeholder = f.Placeholder
			ti.CharLimit = 256
			ti.Width = 35 
			if f.Column == 1 {
				ti.Width = 25 // Compact inputs
			}

			inputs[i] = TextInputWrapper{model: ti}
			inputs[i] = inputs[i].SetValue(f.Value)
			
			if i == 0 {
				var cmd tea.Cmd
				inputs[i], cmd = inputs[i].Focus()
				_ = cmd
			}
		}
	}

	return Model{
		title:      title,
		inputs:     inputs,
		fields:     fields,
		styles:     DefaultStyles(),
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			return m, m.updateFocus()

		case "shift+tab":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			return m, m.updateFocus()
			
		// Explicit submission keys
		// Note: TextAreas consume Enter, so Ctrl+Enter is safer
		case "ctrl+s", "ctrl+enter":
			m.submitted = true
			return m, nil
			
		case "esc":
			m.cancelled = true
			return m, nil
		}
		
		// Fallback for Ctrl+Enter raw codes if needed
		// (Assuming basic string matching works, handled by bubbletea)
	}

	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	return m, cmd
}

func (m *Model) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex {
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Focus()
			cmds[i] = cmd
		} else {
			m.inputs[i] = m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m Model) View() string {
	var leftB strings.Builder
	
	// Collect Grid Cells for Right Panel
	type GridCell struct {
		Row, Col int
		Content  string
		Focused  bool
	}
	var gridCells []GridCell

	maxRow := 0
	maxCol := 0

	for i, f := range m.fields {
		// Header (Only show if present)
		var header string
		if f.Header != "" {
			header = m.styles.Header.Render(f.Header) + "\n"
		}
		
		// Style selection
		labelStyle := m.styles.Label
		inputStyle := m.styles.Input
		isFocused := i == m.focusIndex
		if isFocused {
			labelStyle = m.styles.LabelFocused
			inputStyle = m.styles.InputFocused
		}
		
		// Render Input Block
		block := header + 
			labelStyle.Render(f.Label) + "\n" +
			inputStyle.Render(m.inputs[i].View()) + "\n"

		if f.Column == 0 {
			leftB.WriteString(block)
		} else {
			// For Right Column, group by (Row, Col)
			// We append the block to an existing cell or create new
			found := false
			for idx := range gridCells {
				if gridCells[idx].Row == f.GridRow && gridCells[idx].Col == f.GridCol {
					gridCells[idx].Content += block
					if isFocused { gridCells[idx].Focused = true }
					found = true
					break
				}
			}
			if !found {
				gridCells = append(gridCells, GridCell{
					Row: f.GridRow, 
					Col: f.GridCol, 
					Content: block,
					Focused: isFocused,
				})
				if f.GridRow > maxRow { maxRow = f.GridRow }
				if f.GridCol > maxCol { maxCol = f.GridCol }
			}
		}
	}

	// Render Left Column
	leftCol := lipgloss.NewStyle().
		Width(40).
		PaddingRight(1).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("240")).
		Render(leftB.String())
	
	// Render Right Grid
	var gridRows []string
	for r := 0; r <= maxRow; r++ {
		var rowCells []string
		for c := 0; c <= maxCol; c++ {
			// Find cell content
			content := ""
			focused := false
			for _, cell := range gridCells {
				if cell.Row == r && cell.Col == c {
					content = cell.Content
					focused = cell.Focused
					break
				}
			}
			
			// Style the cell container
			style := m.styles.Cell
			if focused { style = m.styles.CellFocused }
			
			// If empty, render placeholder? Or just empty string
			if content != "" {
				rowCells = append(rowCells, style.Render(content))
			} else {
				// Empty placeholder to maintain grid alignment if needed
				// rowCells = append(rowCells, style.Render("Also"))
			}
		}
		gridRows = append(gridRows, lipgloss.JoinHorizontal(lipgloss.Top, rowCells...))
	}
	
	rightCol := lipgloss.NewStyle().
		PaddingLeft(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, gridRows...))

	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	title := m.styles.Title.Render("✨ " + m.title)
	help := m.styles.Help.Render("Tab: Next • Ctrl+S/Ctrl+Enter: Confirm • Esc: Cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, columns, "\n", help)
	box := m.styles.Box.Render(content)

	if m.width > 0 && m.height > 0 {
		box = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	return box
}

func (m Model) IsSubmitted() bool { return m.submitted }
func (m Model) IsCancelled() bool { return m.cancelled }
func (m Model) Values() []string {
	values := make([]string, len(m.inputs))
	for i, input := range m.inputs {
		values[i] = input.Value()
	}
	return values
}
