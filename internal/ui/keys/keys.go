// Package keys defines keyboard shortcuts for VibeMux TUI.
package keys

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keyboard shortcuts.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Tab      key.Binding
	ShiftTab key.Binding

	// Actions
	Enter      key.Binding
	Delete     key.Binding
	Add        key.Binding
	Profiles   key.Binding
	Help           key.Binding
	ModeToggle     key.Binding
	DispatchToggle key.Binding
	Quit           key.Binding
	Close          key.Binding

	// Terminal
	PaneLeft  key.Binding
	PaneRight key.Binding
	PaneUp    key.Binding
	PaneDown  key.Binding
	
	// Chain Mode
	AssignRoles     key.Binding
	AssignRolesFile key.Binding
	
	// Auto-Turn & Preview
	NextTurn       key.Binding
	AutoTurnToggle key.Binding
	FilePreview    key.Binding
}

// DefaultKeyMap returns the default keyboard shortcuts.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "run/select"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "delete"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add project"),
		),
		Profiles: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "profiles"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		ModeToggle: key.NewBinding(
			key.WithKeys("f12", "ctrl+e"),
			key.WithHelp("F12/Ctrl+E", "term/ctrl"),
		),
		DispatchToggle: key.NewBinding(
			key.WithKeys("alt+m"),
			key.WithHelp("Alt+m", "dispatch"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Close: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "close"),
		),
		PaneLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "pane left"),
		),
		PaneRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "pane right"),
		),
		PaneUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "pane up"),
		),
		PaneDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "pane down"),
		),
		AssignRoles: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("Ctrl+R", "assign roles"),
		),
		AssignRolesFile: key.NewBinding(
			key.WithKeys("alt+f"),
			key.WithHelp("Alt+F", "assign roles (file)"),
		),
		NextTurn: key.NewBinding(
			key.WithKeys("alt+n"),
			key.WithHelp("Alt+N", "next turn"),
		),
		AutoTurnToggle: key.NewBinding(
			key.WithKeys("alt+a"),
			key.WithHelp("Alt+A", "auto-turn on/off"),
		),
		FilePreview: key.NewBinding(
			key.WithKeys("alt+v"),
			key.WithHelp("Alt+V", "file preview"),
		),
	}
}

// ShortHelp returns short help text for the status bar.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Tab,
		k.Enter,
		k.Add,
		k.Profiles,
		k.Delete,
		k.Close,
		k.ModeToggle,
		k.Quit,
	}
}

// FullHelp returns complete help text.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab, k.ShiftTab},
		{k.Enter, k.Add, k.Delete, k.Close, k.Profiles},
		{k.ModeToggle, k.Quit, k.PaneLeft, k.PaneRight, k.PaneUp, k.PaneDown},
		{k.Help},
	}
}
