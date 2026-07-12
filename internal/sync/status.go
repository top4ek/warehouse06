package sync

import (
	"sync"
	"sync/atomic"
	"time"

	"warehouse06/internal/gitdate"
)

// Status tracks sync progress and the last successful sync check.
type Status struct {
	syncing atomic.Bool

	mu            sync.RWMutex
	lastSyncedAt  time.Time
	storageCommit gitdate.Commit
	hasCommit     bool
}

func NewStatus() *Status {
	return &Status{}
}

func (s *Status) SetSyncing(v bool) {
	s.syncing.Store(v)
}

func (s *Status) Syncing() bool {
	return s.syncing.Load()
}

func (s *Status) SetSuccess(syncedAt time.Time, commit gitdate.Commit, hasCommit bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSyncedAt = syncedAt
	if hasCommit {
		s.storageCommit = commit
		s.hasCommit = true
	}
}

// LastSyncedAt returns the time of the last successful sync check, whether
// or not it resulted in a database rebuild.
func (s *Status) LastSyncedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSyncedAt
}

// StorageCommit returns the last known storage HEAD after git sync.
func (s *Status) StorageCommit() (gitdate.Commit, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.storageCommit, s.hasCommit
}
