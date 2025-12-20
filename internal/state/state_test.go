package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir)

	assert.NilError(t, err)
	assert.Assert(t, mgr != nil)
	assert.Equal(t, tmpDir, mgr.stateDir)
}

func TestNewManager_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "nested", "state")

	mgr, err := NewManager(stateDir)

	assert.NilError(t, err)
	assert.Assert(t, mgr != nil)

	// Verify directory was created
	info, err := os.Stat(stateDir)
	assert.NilError(t, err)
	assert.Assert(t, info.IsDir())
}

func TestHashIP(t *testing.T) {
	hash1 := HashIP("203.0.113.42")
	hash2 := HashIP("203.0.113.42")
	hash3 := HashIP("192.168.1.1")

	// Same IP should produce same hash
	assert.Equal(t, hash1, hash2)

	// Different IP should produce different hash
	assert.Assert(t, hash1 != hash3)

	// Hash should be 64 characters (SHA256 hex)
	assert.Equal(t, 64, len(hash1))
}

func TestManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	now := time.Now().UTC().Truncate(time.Second)
	state := &State{
		IPHash:    HashIP("203.0.113.42"),
		UpdatedAt: now,
	}

	// Save
	err := mgr.Save("test-owner", "home", state)
	assert.NilError(t, err)

	// Load
	loaded, err := mgr.Load("test-owner", "home")
	assert.NilError(t, err)
	assert.Assert(t, loaded != nil)
	assert.Equal(t, state.IPHash, loaded.IPHash)
	assert.Equal(t, now.Unix(), loaded.UpdatedAt.Unix())
}

func TestManager_Load_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	loaded, err := mgr.Load("nonexistent", "location")

	assert.NilError(t, err)
	assert.Assert(t, loaded == nil)
}

func TestManager_HasIPChanged_NoState(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	changed, err := mgr.HasIPChanged("test-owner", "home", "203.0.113.42")

	assert.NilError(t, err)
	assert.Assert(t, changed, "should return true when no state exists")
}

func TestManager_HasIPChanged_SameIP(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	// Save initial state
	state := &State{
		IPHash:    HashIP("203.0.113.42"),
		UpdatedAt: time.Now().UTC(),
	}
	mgr.Save("test-owner", "home", state)

	// Check with same IP
	changed, err := mgr.HasIPChanged("test-owner", "home", "203.0.113.42")

	assert.NilError(t, err)
	assert.Assert(t, !changed, "should return false when IP is the same")
}

func TestManager_HasIPChanged_DifferentIP(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	// Save initial state
	state := &State{
		IPHash:    HashIP("203.0.113.42"),
		UpdatedAt: time.Now().UTC(),
	}
	mgr.Save("test-owner", "home", state)

	// Check with different IP
	changed, err := mgr.HasIPChanged("test-owner", "home", "192.168.1.1")

	assert.NilError(t, err)
	assert.Assert(t, changed, "should return true when IP is different")
}

func TestManager_StateFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	path := mgr.stateFilePath("test-owner", "home")

	expected := filepath.Join(tmpDir, "test-owner-home.state")
	assert.Equal(t, expected, path)
}

func TestManager_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	state := &State{
		IPHash:    HashIP("203.0.113.42"),
		UpdatedAt: time.Now().UTC(),
	}
	mgr.Save("test-owner", "home", state)

	// Check file permissions
	path := mgr.stateFilePath("test-owner", "home")
	info, err := os.Stat(path)
	assert.NilError(t, err)

	// File should be readable/writable by owner only (0600)
	perm := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0600), perm)
}
