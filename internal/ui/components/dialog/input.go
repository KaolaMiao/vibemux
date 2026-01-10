// Package dialog provides modal dialog components for VibeMux.
package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazyvibe/vibemux/pkg/utils"
)

// InputField represents a single input field in the dialog.
type InputField struct {
	Label          string
	Placeholder    string
	Value          string
	EnablePathComp bool // Enable path completion for this field
	Options        []string
}

// InputDialog is a modal dialog for text input.
type InputDialog struct {
	title             string
	inputs            []textinput.Model
	labels            []string
	pathCompEnabled   []bool // Track which fields have path completion enabled
	optionCompEnabled []bool
	options           [][]string
	focusIndex        int
	width             int
	height            int
	submitted         bool
	cancelled         bool
	styles            InputStyles

	// Path completion
	pathCompleter   *utils.PathCompleter
	suggestions     []string
	suggestionIndex int
	showSuggestions bool
}

// InputStyles defines the visual appearance of the dialog.
type InputStyles struct {
	Overlay      lipgloss.Style
	Box          lipgloss.Style
	Title        lipgloss.Style
	Label        lipgloss.Style
	LabelFocused lipgloss.Style
	Input        lipgloss.Style
	InputFocused lipgloss.Style
	Button       lipgloss.Style
	ButtonActive lipgloss.Style
	Help         lipgloss.Style
}

// DefaultInputStyles returns beautifully styled dialog styles.
func DefaultInputStyles() InputStyles {
	purple := lipgloss.Color("#7C3AED")
	cyan := lipgloss.Color("#06B6D4")
	pink := lipgloss.Color("#EC4899")
	surface := lipgloss.Color("#1E1E2E")
	surfaceLight := lipgloss.Color("#313244")
	text := lipgloss.Color("#CDD6F4")
	textMuted := lipgloss.Color("#6C7086")

	return InputStyles{
		Overlay: lipgloss.NewStyle().
			Background(lipgloss.Color("#00000088")),

		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Background(surface).
			Padding(1, 2),

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
			Foreground(pink).
			Bold(true).
			MarginBottom(0),

		Input: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(surfaceLight).
			Padding(0, 1).
			MarginBottom(1),

		InputFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(0, 1).
			MarginBottom(1),

		Button: lipgloss.NewStyle().
			Foreground(textMuted).
			Background(surfaceLight).
			Padding(0, 2).
			MarginRight(1),

		ButtonActive: lipgloss.NewStyle().
			Foreground(text).
			Background(purple).
			Bold(true).
			Padding(0, 2).
			MarginRight(1),

		Help: lipgloss.NewStyle().
			Foreground(textMuted).
			MarginTop(1),
	}
}

// NewInputDialog creates a new input dialog.
func NewInputDialog(title string, fields []InputField) InputDialog {
	inputs := make([]textinput.Model, len(fields))
	labels := make([]string, len(fields))
	pathCompEnabled := make([]bool, len(fields))
	optionCompEnabled := make([]bool, len(fields))
	options := make([][]string, len(fields))

	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.Placeholder
		ti.SetValue(f.Value)
		ti.CharLimit = 256
		ti.Width = 40

		if i == 0 {
			ti.Focus()
		}

		inputs[i] = ti
		labels[i] = f.Label
		pathCompEnabled[i] = f.EnablePathComp
		if len(f.Options) > 0 {
			optionCompEnabled[i] = true
			options[i] = append([]string{}, f.Options...)
		}
	}

	return InputDialog{
		title:             title,
		inputs:            inputs,
		labels:            labels,
		pathCompEnabled:   pathCompEnabled,
		optionCompEnabled: optionCompEnabled,
		options:           options,
		styles:            DefaultInputStyles(),
		pathCompleter:     utils.NewPathCompleter(nil),
	}
}

// SetSize updates the dialog dimensions.
func (d *InputDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Update handles input dialog messages.
func (d InputDialog) Update(msg tea.Msg) (InputDialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// If path completion is enabled and we have suggestions, cycle through them
			if d.isSuggestionEnabled() && d.showSuggestions && len(d.suggestions) > 0 {
				d.suggestionIndex = (d.suggestionIndex + 1) % len(d.suggestions)
				d.inputs[d.focusIndex].SetValue(d.suggestions[d.suggestionIndex])
				d.inputs[d.focusIndex].CursorEnd()
				return d, nil
			}
			// Otherwise, move to next field
			d.focusIndex++
			if d.focusIndex >= len(d.inputs) {
				d.focusIndex = 0
			}
			d.showSuggestions = false
			d.suggestions = nil
			return d, d.updateFocus()

		case "shift+tab":
			// If showing suggestions, cycle backwards
			if d.isSuggestionEnabled() && d.showSuggestions && len(d.suggestions) > 0 {
				d.suggestionIndex--
				if d.suggestionIndex < 0 {
					d.suggestionIndex = len(d.suggestions) - 1
				}
				d.inputs[d.focusIndex].SetValue(d.suggestions[d.suggestionIndex])
				d.inputs[d.focusIndex].CursorEnd()
				return d, nil
			}
			// Otherwise, move to previous field
			d.focusIndex--
			if d.focusIndex < 0 {
				d.focusIndex = len(d.inputs) - 1
			}
			d.showSuggestions = false
			d.suggestions = nil
			return d, d.updateFocus()

		case "down":
			// Move to next field
			d.focusIndex++
			if d.focusIndex >= len(d.inputs) {
				d.focusIndex = 0
			}
			d.showSuggestions = false
			d.suggestions = nil
			return d, d.updateFocus()

		case "up":
			// Move to previous field
			d.focusIndex--
			if d.focusIndex < 0 {
				d.focusIndex = len(d.inputs) - 1
			}
			d.showSuggestions = false
			d.suggestions = nil
			return d, d.updateFocus()

		case "enter":
			d.submitted = true
			return d, nil

		case "esc":
			if d.showSuggestions {
				// First Esc hides suggestions
				d.showSuggestions = false
				d.suggestions = nil
				return d, nil
			}
			d.cancelled = true
			return d, nil

		case "ctrl+space":
			// Trigger path completion manually
			if d.isSuggestionEnabled() {
				d.updateSuggestions()
				d.showSuggestions = len(d.suggestions) > 0
			}
			return d, nil
		}
	}

	// Update focused input
	var cmd tea.Cmd
	d.inputs[d.focusIndex], cmd = d.inputs[d.focusIndex].Update(msg)

	// Auto-trigger completion if enabled
	if d.isSuggestionEnabled() {
		d.updateSuggestions()
		d.showSuggestions = len(d.suggestions) > 0
	}

	return d, cmd
}

// updateSuggestions refreshes the path completion suggestions.
func (d *InputDialog) updateSuggestions() {
	input := d.inputs[d.focusIndex].Value()
	if d.pathCompEnabled[d.focusIndex] {
		d.suggestions = d.pathCompleter.Complete(input)
	} else if d.optionCompEnabled[d.focusIndex] {
		d.suggestions = d.matchOptions(input)
	} else {
		d.suggestions = nil
	}
	d.suggestionIndex = 0
}

// updateFocus sets focus to the correct input.
func (d *InputDialog) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(d.inputs))
	for i := range d.inputs {
		if i == d.focusIndex {
			cmds[i] = d.inputs[i].Focus()
		} else {
			d.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

// View renders the dialog.
func (d InputDialog) View() string {
	var b strings.Builder

	// Title
	b.WriteString(d.styles.Title.Render("✨ " + d.title))
	b.WriteString("\n\n")

	// Input fields
	for i, input := range d.inputs {
		labelStyle := d.styles.Label
		inputStyle := d.styles.Input
		if i == d.focusIndex {
			labelStyle = d.styles.LabelFocused
			inputStyle = d.styles.InputFocused
		}

		b.WriteString(labelStyle.Render(d.labels[i]))
		b.WriteString("\n")
		b.WriteString(inputStyle.Render(input.View()))
		b.WriteString("\n")

		// Show suggestions for completion fields
		if i == d.focusIndex && d.isSuggestionEnabled() && d.showSuggestions && len(d.suggestions) > 0 {
			suggestionStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6C7086")).
				PaddingLeft(2)
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).
				Bold(true).
				PaddingLeft(2)

			// Show max 5 suggestions
			maxShow := 5
			if len(d.suggestions) < maxShow {
				maxShow = len(d.suggestions)
			}

			for j := 0; j < maxShow; j++ {
				if j == d.suggestionIndex {
					b.WriteString(selectedStyle.Render("→ " + d.suggestions[j]))
				} else {
					b.WriteString(suggestionStyle.Render("  " + d.suggestions[j]))
				}
				b.WriteString("\n")
			}
			if len(d.suggestions) > maxShow {
				b.WriteString(suggestionStyle.Render("  ..."))
				b.WriteString("\n")
			}
		}
	}

	// Help text
	helpText := "Enter: Confirm • Esc: Cancel"
	if d.isSuggestionEnabled() {
		helpText = "Tab: Cycle suggestions • Enter: Confirm • Esc: Cancel"
	}
	b.WriteString(d.styles.Help.Render(helpText))

	// Wrap in box
	content := d.styles.Box.Render(b.String())

	// Center in screen
	if d.width > 0 && d.height > 0 {
		boxWidth := lipgloss.Width(content)
		boxHeight := lipgloss.Height(content)
		padX := (d.width - boxWidth) / 2
		padY := (d.height - boxHeight) / 2

		if padX < 0 {
			padX = 0
		}
		if padY < 0 {
			padY = 0
		}

		content = lipgloss.NewStyle().
			MarginLeft(padX).
			MarginTop(padY).
			Render(content)
	}

	return content
}

// IsSubmitted returns true if the user submitted the dialog.
func (d InputDialog) IsSubmitted() bool {
	return d.submitted
}

// IsCancelled returns true if the user cancelled the dialog.
func (d InputDialog) IsCancelled() bool {
	return d.cancelled
}

// Values returns all input values.
func (d InputDialog) Values() []string {
	values := make([]string, len(d.inputs))
	for i, input := range d.inputs {
		values[i] = input.Value()
	}
	return values
}

// Value returns the value of the input at the given index.
func (d InputDialog) Value(index int) string {
	if index < 0 || index >= len(d.inputs) {
		return ""
	}
	return d.inputs[index].Value()
}

// Reset resets the dialog state.
func (d *InputDialog) Reset() {
	d.submitted = false
	d.cancelled = false
	d.focusIndex = 0
	d.suggestions = nil
	d.showSuggestions = false
	for i := range d.inputs {
		d.inputs[i].SetValue("")
		if i == 0 {
			d.inputs[i].Focus()
		} else {
			d.inputs[i].Blur()
		}
	}
}

// SetFieldOptions logic helper
func (d *InputDialog) SetFieldOptions(index int, options []string) {
	if index < 0 || index >= len(d.inputs) {
		return
	}
	if len(options) == 0 {
		d.optionCompEnabled[index] = false
		d.options[index] = nil
		return
	}
	d.optionCompEnabled[index] = true
	d.options[index] = append([]string{}, options...)
}

func (d *InputDialog) isSuggestionEnabled() bool {
	if d.focusIndex < 0 || d.focusIndex >= len(d.inputs) {
		return false
	}
	return d.pathCompEnabled[d.focusIndex] || d.optionCompEnabled[d.focusIndex]
}

func (d *InputDialog) matchOptions(input string) []string {
	opts := d.options[d.focusIndex]
	if len(opts) == 0 {
		return nil
	}
	if input == "" {
		return opts
	}
	lower := strings.ToLower(input)
	matches := make([]string, 0, len(opts))
	for _, opt := range opts {
		if strings.HasPrefix(strings.ToLower(opt), lower) {
			matches = append(matches, opt)
		}
	}
	if len(matches) == 0 {
		for _, opt := range opts {
			if strings.Contains(strings.ToLower(opt), lower) {
				matches = append(matches, opt)
			}
		}
	}
	if len(matches) > 10 {
		matches = matches[:10]
	}
	return matches
}
