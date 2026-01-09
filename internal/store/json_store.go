package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/lazyvibe/vibemux/internal/model"
)

var (
	// ErrNotFound is returned when an entity is not found.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when creating a duplicate entity.
	ErrAlreadyExists = errors.New("already exists")
)

// data represents the JSON file structure.
type data struct {
	Projects []model.Project `json:"projects"`
	Profiles []model.Profile `json:"profiles"`
}

// JSONStore implements Store using JSON file persistence.
type JSONStore struct {
	mu       sync.RWMutex
	path     string
	data     *data
	modified bool
}

// NewJSONStore creates a new JSON file-based store.
func NewJSONStore(configDir string) (*JSONStore, error) {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(configDir, "data.json")
	s := &JSONStore{
		path: path,
		data: &data{
			Projects: []model.Project{},
			Profiles: []model.Profile{},
		},
	}

	// Load existing data if file exists
	if _, err := os.Stat(path); err == nil {
		if err := s.load(); err != nil {
			return nil, err
		}
	} else {
		// Create default profile if new installation
		defaultProfile := model.DefaultProfile()
		s.data.Profiles = append(s.data.Profiles, *defaultProfile)
		if err := s.save(); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// load reads data from the JSON file.
func (s *JSONStore) load() error {
	content, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, s.data); err != nil {
		return err
	}
	if s.normalizeProfiles() {
		return s.save()
	}
	return nil
}

// save writes data to the JSON file.
func (s *JSONStore) save() error {
	content, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, content, 0644)
}

// Close persists any pending changes.
func (s *JSONStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.modified {
		return s.save()
	}
	return nil
}

// ---------- ProjectStore Implementation ----------

// List returns all projects sorted by LastUsed descending.
func (s *JSONStore) List(_ context.Context) ([]model.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Project, len(s.data.Projects))
	copy(result, s.data.Projects)

	// Sort by LastUsed descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastUsed > result[j].LastUsed
	})

	return result, nil
}

// Get retrieves a project by ID.
func (s *JSONStore) Get(_ context.Context, id string) (*model.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.data.Projects {
		if s.data.Projects[i].ID == id {
			p := s.data.Projects[i]
			return &p, nil
		}
	}
	return nil, ErrNotFound
}

// Create adds a new project.
func (s *JSONStore) Create(_ context.Context, p *model.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate ID
	for _, existing := range s.data.Projects {
		if existing.ID == p.ID {
			return ErrAlreadyExists
		}
	}

	// Assign default profile if not set
	if p.ProfileID == "" {
		for _, profile := range s.data.Profiles {
			if profile.IsDefault {
				p.ProfileID = profile.ID
				break
			}
		}
	}

	s.data.Projects = append(s.data.Projects, *p)
	s.modified = true
	return s.save()
}

// Update modifies an existing project.
func (s *JSONStore) Update(_ context.Context, p *model.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Projects {
		if s.data.Projects[i].ID == p.ID {
			s.data.Projects[i] = *p
			s.modified = true
			return s.save()
		}
	}
	return ErrNotFound
}

// Delete removes a project by ID.
func (s *JSONStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Projects {
		if s.data.Projects[i].ID == id {
			s.data.Projects = append(s.data.Projects[:i], s.data.Projects[i+1:]...)
			s.modified = true
			return s.save()
		}
	}
	return ErrNotFound
}

// ---------- ProfileStore Implementation ----------

// List returns all profiles.
func (s *JSONStore) ListProfiles(_ context.Context) ([]model.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Profile, len(s.data.Profiles))
	copy(result, s.data.Profiles)
	return result, nil
}

// GetProfile retrieves a profile by ID.
func (s *JSONStore) GetProfile(_ context.Context, id string) (*model.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.data.Profiles {
		if s.data.Profiles[i].ID == id {
			p := s.data.Profiles[i]
			return &p, nil
		}
	}
	return nil, ErrNotFound
}

// CreateProfile adds a new profile.
func (s *JSONStore) CreateProfile(_ context.Context, p *model.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.data.Profiles {
		if existing.ID == p.ID {
			return ErrAlreadyExists
		}
	}

	s.normalizeProfile(p)
	s.data.Profiles = append(s.data.Profiles, *p)
	s.modified = true
	return s.save()
}

// UpdateProfile modifies an existing profile.
func (s *JSONStore) UpdateProfile(_ context.Context, p *model.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.normalizeProfile(p)
	for i := range s.data.Profiles {
		if s.data.Profiles[i].ID == p.ID {
			s.data.Profiles[i] = *p
			s.modified = true
			return s.save()
		}
	}
	return ErrNotFound
}

// DeleteProfile removes a profile by ID.
func (s *JSONStore) DeleteProfile(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Prevent deletion of default profile
	for i := range s.data.Profiles {
		if s.data.Profiles[i].ID == id {
			if s.data.Profiles[i].IsDefault {
				return errors.New("cannot delete default profile")
			}
			s.data.Profiles = append(s.data.Profiles[:i], s.data.Profiles[i+1:]...)
			s.modified = true
			return s.save()
		}
	}
	return ErrNotFound
}

// GetDefault returns the default profile.
func (s *JSONStore) GetDefault(_ context.Context) (*model.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.data.Profiles {
		if s.data.Profiles[i].IsDefault {
			p := s.data.Profiles[i]
			return &p, nil
		}
	}

	// Fallback: return first profile if no default set
	if len(s.data.Profiles) > 0 {
		p := s.data.Profiles[0]
		return &p, nil
	}

	return nil, ErrNotFound
}

func (s *JSONStore) normalizeProfiles() bool {
	changed := false
	for i := range s.data.Profiles {
		if s.normalizeProfile(&s.data.Profiles[i]) {
			changed = true
		}
	}
	return changed
}

func (s *JSONStore) normalizeProfile(p *model.Profile) bool {
	if p == nil {
		return false
	}
	changed := false
	if len(p.CommandArgs) > 0 {
		combined := strings.TrimSpace(strings.Join(append([]string{p.Command}, p.CommandArgs...), " "))
		p.Command = combined
		p.CommandArgs = nil
		changed = true
	}
	if strings.TrimSpace(p.Command) == "" {
		p.Command = "claude"
		changed = true
	}
	if p.Driver != model.DriverNative {
		p.Driver = model.DriverNative
		changed = true
	}
	return changed
}
