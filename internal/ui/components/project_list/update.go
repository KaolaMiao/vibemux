package projectlist

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages for the project list.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.HandleKey(msg.String())
	}

	return m, nil
}
