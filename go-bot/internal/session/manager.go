package session

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Session represents a Claude Code session with its metadata
type Session struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	WorkingDir  string    `json:"working_dir"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	Description string    `json:"description,omitempty"`
}

// Manager manages multiple Claude sessions
type Manager struct {
	sessions      map[string]*Session
	currentID     string
	storePath     string
	mu            sync.RWMutex
}

// NewManager creates a new session manager
func NewManager(storePath string) (*Manager, error) {
	m := &Manager{
		sessions:  make(map[string]*Session),
		storePath: storePath,
	}

	// Load existing sessions from disk
	if err := m.load(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load sessions: %w", err)
		}
	}

	return m, nil
}

// Create creates a new session
func (m *Manager) Create(name, description, workingDir string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	session := &Session{
		ID:          "", // Will be set by Claude SDK
		Name:        name,
		WorkingDir:  workingDir,
		CreatedAt:   now,
		LastUsedAt:  now,
		Description: description,
	}

	// Store by name for now, will update with ID when available
	m.sessions[name] = session
	m.currentID = name

	if err := m.save(); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// List returns all sessions
func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// Get returns a session by name or ID
func (m *Manager) Get(nameOrID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, ok := m.sessions[nameOrID]; ok {
		return session, nil
	}

	// Try to find by ID
	for _, session := range m.sessions {
		if session.ID == nameOrID {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", nameOrID)
}

// Switch switches to a different session
func (m *Manager) Switch(nameOrID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.get(nameOrID)
	if err != nil {
		return nil, err
	}

	m.currentID = nameOrID
	session.LastUsedAt = time.Now()

	if err := m.save(); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// Current returns the current session
func (m *Manager) Current() *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentID == "" {
		return nil
	}

	return m.sessions[m.currentID]
}

// UpdateSessionID updates the session ID (called after Claude SDK assigns one)
func (m *Manager) UpdateSessionID(name, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[name]
	if !ok {
		return fmt.Errorf("session not found: %s", name)
	}

	session.ID = id
	return m.save()
}

// UpdateWorkingDir updates the working directory for the current session
func (m *Manager) UpdateWorkingDir(workingDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentID == "" {
		return fmt.Errorf("no current session")
	}

	session := m.sessions[m.currentID]
	session.WorkingDir = workingDir
	session.LastUsedAt = time.Now()

	return m.save()
}

// Delete deletes a session
func (m *Manager) Delete(nameOrID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.get(nameOrID)
	if err != nil {
		return err
	}

	// Find the key to delete
	var keyToDelete string
	for key, s := range m.sessions {
		if s == session {
			keyToDelete = key
			break
		}
	}

	if keyToDelete != "" {
		delete(m.sessions, keyToDelete)
	}

	// If this was the current session, clear it
	if m.currentID == nameOrID || m.currentID == keyToDelete {
		m.currentID = ""
	}

	return m.save()
}

// get (internal, no lock) returns a session by name or ID
func (m *Manager) get(nameOrID string) (*Session, error) {
	if session, ok := m.sessions[nameOrID]; ok {
		return session, nil
	}

	// Try to find by ID
	for _, session := range m.sessions {
		if session.ID == nameOrID {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", nameOrID)
}

// save persists sessions to disk
func (m *Manager) save() error {
	data, err := json.MarshalIndent(struct {
		Sessions  map[string]*Session `json:"sessions"`
		CurrentID string              `json:"current_id"`
	}{
		Sessions:  m.sessions,
		CurrentID: m.currentID,
	}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.storePath, data, 0644)
}

// load loads sessions from disk
func (m *Manager) load() error {
	data, err := os.ReadFile(m.storePath)
	if err != nil {
		return err
	}

	var stored struct {
		Sessions  map[string]*Session `json:"sessions"`
		CurrentID string              `json:"current_id"`
	}

	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	m.sessions = stored.Sessions
	m.currentID = stored.CurrentID

	return nil
}
