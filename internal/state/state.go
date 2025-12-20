package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultStateDir = ".config/ddns-client"
	FilePermission  = 0600
	DirPermission   = 0700
)

// State represents the persisted state for a location.
type State struct {
	IPHash    string    `json:"ip_hash"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Manager handles state file operations.
type Manager struct {
	stateDir string
}

// NewManager creates a new state manager.
// If stateDir is empty, defaults to ~/.config/ddns-client/
func NewManager(stateDir string) (*Manager, error) {
	if stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		stateDir = filepath.Join(homeDir, DefaultStateDir)
	}

	// Ensure directory exists with proper permissions
	if err := os.MkdirAll(stateDir, DirPermission); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	return &Manager{stateDir: stateDir}, nil
}

// stateFilePath returns the path for a given owner/location.
func (m *Manager) stateFilePath(owner, location string) string {
	filename := fmt.Sprintf("%s-%s.state", owner, location)
	return filepath.Join(m.stateDir, filename)
}

// HashIP returns the SHA256 hash of an IP address.
func HashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}

// Load reads the state for a given owner/location.
// Returns nil, nil if the state file doesn't exist.
func (m *Manager) Load(owner, location string) (*State, error) {
	path := m.stateFilePath(owner, location)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// Save writes the state for a given owner/location.
func (m *Manager) Save(owner, location string, state *State) error {
	path := m.stateFilePath(owner, location)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, FilePermission); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// HasIPChanged checks if the IP has changed since the last save.
// Returns true if the state doesn't exist or the hash differs.
func (m *Manager) HasIPChanged(owner, location, currentIP string) (bool, error) {
	state, err := m.Load(owner, location)
	if err != nil {
		return false, err
	}

	if state == nil {
		return true, nil // No previous state means "changed"
	}

	currentHash := HashIP(currentIP)
	return state.IPHash != currentHash, nil
}
