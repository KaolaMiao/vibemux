package runtime

import (
	"context"
	"errors"
	"os"
    "path/filepath"
    "fmt"
	"sync"

	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/runtime/driver"
)

// Engine manages PTY sessions for multiple projects.
type Engine interface {
	// CreateSession creates and starts a new session for a project.
	CreateSession(ctx context.Context, project *model.Project, profile *model.Profile, rows, cols int) (Session, error)
	// GetSession retrieves an existing session by project ID.
	GetSession(projectID string) (Session, bool)
	// ListSessions returns all active sessions.
	ListSessions() []Session
	// CloseSession stops and removes a session.
	CloseSession(projectID string) error
	// CloseAll stops and removes all sessions.
	CloseAll() error
}

// DefaultEngine is the default implementation of Engine.
type DefaultEngine struct {
	mu       sync.RWMutex
	sessions map[string]*PTYSession
	registry *driver.Registry
}

// NewEngine creates a new runtime engine.
func NewEngine() *DefaultEngine {
	return NewEngineWithConfig(driver.Config{})
}

// NewEngineWithConfig creates a new runtime engine with configuration.
func NewEngineWithConfig(cfg driver.Config) *DefaultEngine {
	return &DefaultEngine{
		sessions: make(map[string]*PTYSession),
		registry: driver.NewRegistryWithConfig(cfg),
	}
}

// CreateSession creates and starts a new PTY session.
func (e *DefaultEngine) CreateSession(ctx context.Context, project *model.Project, profile *model.Profile, rows, cols int) (Session, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if session already exists
	if existing, ok := e.sessions[project.ID]; ok {
		if existing.Status() == model.SessionStatusRunning {
			return existing, nil
		}
		// Remove stopped session to create new one
		delete(e.sessions, project.ID)
	}

	// Use native driver for all profiles; command line is user-defined.
	d, ok := e.registry.Get(model.DriverNative)
	if !ok {
		return nil, errors.New("driver not found: native")
	}

	if project == nil {
		return nil, errors.New("project is nil")
	}
	if info, err := os.Stat(project.Path); err != nil || !info.IsDir() {
		return nil, errors.New("project path not found: " + project.Path)
	}

	// Inject CLAUDE_CONFIG_DIR for isolation if not present
    // We isolate by Project ID to ensure multiple projects don't conflict
    sessionConfigDir := filepath.Join(os.Getenv("USERPROFILE"), ".config", "vibemux", "sessions", project.ID)
    if err := os.MkdirAll(sessionConfigDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create session config dir: %w", err)
    }
    
    // Copy the existing EnvVars to avoid mutating the original profile
    if profile.EnvVars == nil {
        profile.EnvVars = make(map[string]string)
    } else {
        // Deep copy needed if we reuse profile pointer, but profile here comes from caller. 
        // To be safe, we can just modify the map if the caller (UI) doesn't reuse it for other sessions.
        // Or we can create a temporary profile clone.
        newEnv := make(map[string]string)
        for k, v := range profile.EnvVars {
            newEnv[k] = v
        }
        profile.EnvVars = newEnv
    }
    
    // Only set if not already set by user
    if _, ok := profile.EnvVars["CLAUDE_CONFIG_DIR"]; !ok {
        profile.EnvVars["CLAUDE_CONFIG_DIR"] = sessionConfigDir
    }

	// Build command
	cmd, err := d.BuildCommand(project.Path, profile)
	if err != nil {
		return nil, err
	}

	// Create session
	session := NewPTYSession(project.ID, cmd)
    if rows > 0 && cols > 0 {
        session.SetInitialSize(rows, cols)
    }

	// Start session
	if err := session.Start(ctx); err != nil {
		return nil, err
	}

	// Store session
	e.sessions[project.ID] = session

	return session, nil
}

// GetSession retrieves an existing session.
func (e *DefaultEngine) GetSession(projectID string) (Session, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	session, ok := e.sessions[projectID]
	if !ok {
		return nil, false
	}
	return session, true
}

// ListSessions returns all sessions.
func (e *DefaultEngine) ListSessions() []Session {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]Session, 0, len(e.sessions))
	for _, s := range e.sessions {
		result = append(result, s)
	}
	return result
}

// CloseSession stops and removes a session.
func (e *DefaultEngine) CloseSession(projectID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	session, ok := e.sessions[projectID]
	if !ok {
		return nil
	}

	if err := session.Stop(); err != nil {
		return err
	}

	delete(e.sessions, projectID)
	return nil
}

// CloseAll stops and removes all sessions.
func (e *DefaultEngine) CloseAll() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var lastErr error
	for id, session := range e.sessions {
		if err := session.Stop(); err != nil {
			lastErr = err
		}
		delete(e.sessions, id)
	}
	return lastErr
}

// GetSessionStatus returns the status of a session without the full session.
func (e *DefaultEngine) GetSessionStatus(projectID string) model.SessionStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	session, ok := e.sessions[projectID]
	if !ok {
		return model.SessionStatusIdle
	}
	return session.Status()
}
