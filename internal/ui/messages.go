// Package ui provides the terminal user interface for VibeMux.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvibe/vibemux/internal/model"
)

// ---------- Projects Messages ----------

// ProjectsLoadedMsg is sent when projects are loaded from store.
type ProjectsLoadedMsg struct {
	Projects []model.Project
	Err      error
}

// ProjectSelectedMsg is sent when a project is selected.
type ProjectSelectedMsg struct {
	ProjectID string
}

// ProjectCreatedMsg is sent when a new project is created.
type ProjectCreatedMsg struct {
	Project model.Project
}

// ProjectDeletedMsg is sent when a project is deleted.
type ProjectDeletedMsg struct {
	ProjectID string
}

// ---------- Profile Messages ----------

// ProfilesLoadedMsg is sent when profiles are loaded from store.
type ProfilesLoadedMsg struct {
	Profiles []model.Profile
	Err      error
}

// ProfileSelectedMsg is sent when a profile is selected.
type ProfileSelectedMsg struct {
	ProfileID string
}

// ProfileSavedMsg is sent when a profile is created or updated.
type ProfileSavedMsg struct {
	Profile model.Profile
	IsNew   bool
}

// ProfileDeletedMsg is sent when a profile is deleted.
type ProfileDeletedMsg struct {
	ProfileID string
}

// ---------- Session Messages ----------

// SessionStartedMsg is sent when a PTY session starts.
type SessionStartedMsg struct {
	ProjectID string
}

// SessionStoppedMsg is sent when a PTY session stops.
type SessionStoppedMsg struct {
	ProjectID string
	Err       error
}

// SessionOutputMsg carries PTY output data.
type SessionOutputMsg struct {
	ProjectID string
	Data      []byte
}

// SessionStatusMsg reports session status changes.
type SessionStatusMsg struct {
	ProjectID string
	Status    model.SessionStatus
}

// ---------- UI Messages ----------

// FocusChangedMsg is sent when focus changes between panes.
type FocusChangedMsg struct {
	Focus FocusArea
}

// ErrorMsg is sent when an error occurs.
type ErrorMsg struct {
	Err error
}

// ---------- Input Messages ----------

// InputSubmittedMsg is sent when text input is submitted.
type InputSubmittedMsg struct {
	Value string
}

// InputCancelledMsg is sent when input dialog is cancelled.
type InputCancelledMsg struct{}

// ---------- Command Functions ----------

// LoadProjects returns a command to load projects from store.
func LoadProjects(loader func() ([]model.Project, error)) tea.Cmd {
	return func() tea.Msg {
		projects, err := loader()
		return ProjectsLoadedMsg{Projects: projects, Err: err}
	}
}

// LoadProfiles returns a command to load profiles from store.
func LoadProfiles(loader func() ([]model.Profile, error)) tea.Cmd {
	return func() tea.Msg {
		profiles, err := loader()
		return ProfilesLoadedMsg{Profiles: profiles, Err: err}
	}
}

// WaitForOutput returns a command that waits for session output.
// It implements "Dynamic Catch-up" by opportunistic batching:
// If the channel has more data ready, it bundles it into a single message
// to reduce Bubble Tea render cycles (which is the main bottleneck).
func WaitForOutput(outputCh <-chan []byte, projectID string) tea.Cmd {
	return func() tea.Msg {
		// 1. Block wait for the first chunk (latency priority)
		data, ok := <-outputCh
		if !ok {
			return SessionStoppedMsg{ProjectID: projectID}
		}

		// 2. Batching loop (throughput priority)
		// If the consumer (UI) is slower than producer (PTY), the channel will fill up.
		// We drain up to maxBatchSize to render fewer frames with more content.
		const maxBatchSize = 32 * 1024 // 32KB batch limit
		
		for len(data) < maxBatchSize {
			select {
			case next, ok := <-outputCh:
				if !ok {
					// Channel closed, return what we have. 
					// The next call to WaitForOutput (if any) or the logic above handles close state.
					return SessionOutputMsg{ProjectID: projectID, Data: data}
				}
				data = append(data, next...)
			default:
				// No more data immediately available, send current batch
				return SessionOutputMsg{ProjectID: projectID, Data: data}
			}
		}

		return SessionOutputMsg{ProjectID: projectID, Data: data}
	}
}
