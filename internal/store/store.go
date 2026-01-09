// Package store provides data persistence abstractions for VibeMux.
package store

import (
	"context"

	"github.com/lazyvibe/vibemux/internal/model"
)

// ProjectStore defines the interface for project persistence.
type ProjectStore interface {
	// List returns all projects sorted by LastUsed descending.
	List(ctx context.Context) ([]model.Project, error)
	// Get retrieves a project by its ID.
	Get(ctx context.Context, id string) (*model.Project, error)
	// Create adds a new project.
	Create(ctx context.Context, p *model.Project) error
	// Update modifies an existing project.
	Update(ctx context.Context, p *model.Project) error
	// Delete removes a project by its ID.
	Delete(ctx context.Context, id string) error
}

// ProfileStore defines the interface for profile persistence.
type ProfileStore interface {
	// ListProfiles returns all profiles.
	ListProfiles(ctx context.Context) ([]model.Profile, error)
	// GetProfile retrieves a profile by its ID.
	GetProfile(ctx context.Context, id string) (*model.Profile, error)
	// CreateProfile adds a new profile.
	CreateProfile(ctx context.Context, p *model.Profile) error
	// UpdateProfile modifies an existing profile.
	UpdateProfile(ctx context.Context, p *model.Profile) error
	// DeleteProfile removes a profile by its ID.
	DeleteProfile(ctx context.Context, id string) error
	// GetDefault returns the default profile.
	GetDefault(ctx context.Context) (*model.Profile, error)
}

// Store combines all storage interfaces.
type Store interface {
	ProjectStore
	ProfileStore
	// Close releases any resources held by the store.
	Close() error
}
