package model

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a managed project directory.
type Project struct {
	// ID is the unique identifier for this project.
	ID string `json:"id"`
	// Name is the display name for the project.
	Name string `json:"name"`
	// Path is the absolute filesystem path to the project.
	Path string `json:"path"`
	// ProfileID references the configuration profile to use.
	ProfileID string `json:"profile_id"`
	// LastUsed is the Unix timestamp of the last session.
	LastUsed int64 `json:"last_used"`
	// CreatedAt is when the project was added.
	CreatedAt int64 `json:"created_at"`
}

// NewProject creates a new project with a generated UUID.
func NewProject(name, path string) *Project {
	now := time.Now().Unix()
	return &Project{
		ID:        uuid.New().String(),
		Name:      name,
		Path:      path,
		CreatedAt: now,
		LastUsed:  now,
	}
}

// Touch updates the LastUsed timestamp to now.
func (p *Project) Touch() {
	p.LastUsed = time.Now().Unix()
}

// SetProfile binds a profile to this project.
func (p *Project) SetProfile(profileID string) {
	p.ProfileID = profileID
}

// DisplayName returns the name to display in the UI.
// Falls back to path basename if name is empty.
func (p *Project) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	// Extract basename from path
	for i := len(p.Path) - 1; i >= 0; i-- {
		if p.Path[i] == '/' {
			return p.Path[i+1:]
		}
	}
	return p.Path
}
