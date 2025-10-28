package session

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// Archive represents an archived session
type Archive struct {
	OriginalName string    `json:"original_name"`
	OriginalID   string    `json:"original_id"`
	WorkingDir   string    `json:"working_dir"`
	Description  string    `json:"description,omitempty"`
	ArchivedAt   time.Time `json:"archived_at"`
	ArchivePath  string    `json:"archive_path"`
	FileSizeBytes int64    `json:"file_size_bytes"`
	MessageCount int       `json:"message_count"`
}

const (
	// archiveDir is the directory where archived sessions are stored
	archiveDir = "/archives"
	// archiveIndexFile is the path to the archive index
	archiveIndexFile = "/archives/index.json"
)

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

// Delete deletes a session (archives it first if it has a session ID)
func (m *Manager) Delete(nameOrID string) error {
	// Archive before deleting (unlock during archive operation)
	archive, archiveErr := m.Archive(nameOrID)
	if archiveErr != nil {
		// Log warning but continue with deletion
		fmt.Printf("Warning: Failed to archive session before deletion: %v\n", archiveErr)
	} else if archive != nil {
		fmt.Printf("Session archived: %s (size: %d bytes, %d messages)\n",
			archive.ArchivePath, archive.FileSizeBytes, archive.MessageCount)
	}

	// Now delete from sessions map
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

// GetSessionSize returns the size and message count of a session's JSONL file
func (m *Manager) GetSessionSize(nameOrID string) (sizeBytes int64, messageCount int, err error) {
	session, err := m.Get(nameOrID)
	if err != nil {
		return 0, 0, err
	}

	if session.ID == "" {
		return 0, 0, nil // New session with no file yet
	}

	sessionFilePath, err := findClaudeSessionFile(session.WorkingDir, session.ID)
	if err != nil {
		return 0, 0, nil // File doesn't exist yet
	}

	fileInfo, err := os.Stat(sessionFilePath)
	if err != nil {
		return 0, 0, err
	}

	messageCount, _ = countMessagesInJSONL(sessionFilePath)

	return fileInfo.Size(), messageCount, nil
}

// Archive archives a session before deletion
// Returns the archive metadata or error
func (m *Manager) Archive(nameOrID string) (*Archive, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.get(nameOrID)
	if err != nil {
		return nil, err
	}

	// Ensure archive directory exists
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Find the Claude session file
	// Claude stores sessions in /home/node/.claude/projects/{normalized-path}/{session-id}.jsonl
	sessionFilePath, err := findClaudeSessionFile(session.WorkingDir, session.ID)
	if err != nil {
		// Session file not found - still archive metadata but note it
		return m.archiveMetadataOnly(session)
	}

	// Get file stats
	fileInfo, err := os.Stat(sessionFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat session file: %w", err)
	}

	// Count messages in session file
	messageCount, err := countMessagesInJSONL(sessionFilePath)
	if err != nil {
		messageCount = 0 // Non-fatal error
	}

	// Create archive filename: {name}_{timestamp}_{session-id}.jsonl
	timestamp := time.Now().Format("20060102_150405")
	archiveFilename := fmt.Sprintf("%s_%s_%s.jsonl", session.Name, timestamp, session.ID)
	archivePath := filepath.Join(archiveDir, archiveFilename)

	// Copy the session file to archive
	if err := copyFile(sessionFilePath, archivePath); err != nil {
		return nil, fmt.Errorf("failed to copy session file: %w", err)
	}

	// Create archive metadata
	archive := &Archive{
		OriginalName:  session.Name,
		OriginalID:    session.ID,
		WorkingDir:    session.WorkingDir,
		Description:   session.Description,
		ArchivedAt:    time.Now(),
		ArchivePath:   archivePath,
		FileSizeBytes: fileInfo.Size(),
		MessageCount:  messageCount,
	}

	// Add to archive index
	if err := m.addToArchiveIndex(archive); err != nil {
		return nil, fmt.Errorf("failed to update archive index: %w", err)
	}

	return archive, nil
}

// archiveMetadataOnly creates an archive entry when the session file doesn't exist
func (m *Manager) archiveMetadataOnly(session *Session) (*Archive, error) {
	archive := &Archive{
		OriginalName:  session.Name,
		OriginalID:    session.ID,
		WorkingDir:    session.WorkingDir,
		Description:   session.Description,
		ArchivedAt:    time.Now(),
		ArchivePath:   "", // No file archived
		FileSizeBytes: 0,
		MessageCount:  0,
	}

	if err := m.addToArchiveIndex(archive); err != nil {
		return nil, fmt.Errorf("failed to update archive index: %w", err)
	}

	return archive, nil
}

// ListArchives returns all archived sessions
func (m *Manager) ListArchives() ([]*Archive, error) {
	archives, err := m.loadArchiveIndex()
	if err != nil {
		if os.IsNotExist(err) {
			return []*Archive{}, nil
		}
		return nil, err
	}
	return archives, nil
}

// GetArchive returns archive metadata by name or ID
func (m *Manager) GetArchive(nameOrID string) (*Archive, error) {
	archives, err := m.ListArchives()
	if err != nil {
		return nil, err
	}

	for _, archive := range archives {
		if archive.OriginalName == nameOrID || archive.OriginalID == nameOrID {
			return archive, nil
		}
	}

	return nil, fmt.Errorf("archive not found: %s", nameOrID)
}

// DeleteArchive permanently deletes an archived session
func (m *Manager) DeleteArchive(nameOrID string) error {
	archive, err := m.GetArchive(nameOrID)
	if err != nil {
		return err
	}

	// Delete the archive file if it exists
	if archive.ArchivePath != "" {
		if err := os.Remove(archive.ArchivePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete archive file: %w", err)
		}
	}

	// Remove from index
	return m.removeFromArchiveIndex(archive)
}

// Helper: loadArchiveIndex loads the archive index
func (m *Manager) loadArchiveIndex() ([]*Archive, error) {
	data, err := os.ReadFile(archiveIndexFile)
	if err != nil {
		return nil, err
	}

	var archives []*Archive
	if err := json.Unmarshal(data, &archives); err != nil {
		return nil, err
	}

	return archives, nil
}

// Helper: saveArchiveIndex saves the archive index
func (m *Manager) saveArchiveIndex(archives []*Archive) error {
	data, err := json.MarshalIndent(archives, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(archiveIndexFile, data, 0644)
}

// Helper: addToArchiveIndex adds an archive to the index
func (m *Manager) addToArchiveIndex(archive *Archive) error {
	archives, err := m.loadArchiveIndex()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	archives = append(archives, archive)
	return m.saveArchiveIndex(archives)
}

// Helper: removeFromArchiveIndex removes an archive from the index
func (m *Manager) removeFromArchiveIndex(archiveToRemove *Archive) error {
	archives, err := m.loadArchiveIndex()
	if err != nil {
		return err
	}

	filtered := make([]*Archive, 0, len(archives))
	for _, archive := range archives {
		if archive.OriginalID != archiveToRemove.OriginalID || archive.ArchivedAt != archiveToRemove.ArchivedAt {
			filtered = append(filtered, archive)
		}
	}

	return m.saveArchiveIndex(filtered)
}

// Helper: findClaudeSessionFile finds the Claude session JSONL file
func findClaudeSessionFile(workingDir, sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session ID is empty")
	}

	// Claude normalizes the path by replacing / with -
	// e.g., /workspace/vestnik -> -workspace-vestnik
	normalizedPath := strings.ReplaceAll(workingDir, "/", "-")

	// Session file path pattern
	sessionFilePath := filepath.Join("/home/node/.claude/projects", normalizedPath, sessionID+".jsonl")

	// Check if file exists
	if _, err := os.Stat(sessionFilePath); err != nil {
		return "", fmt.Errorf("session file not found at %s: %w", sessionFilePath, err)
	}

	return sessionFilePath, nil
}

// Helper: copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// Helper: countMessagesInJSONL counts the number of lines in a JSONL file
func countMessagesInJSONL(filePath string) (int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	return count, nil
}
