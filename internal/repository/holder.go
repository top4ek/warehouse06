package repository

import (
	"strings"
	"sync"
)

// Holder provides thread-safe access to the active SQLite repository.
type Holder struct {
	mu   sync.RWMutex
	repo *SQLiteRepository
	dsn  string
}

func NewHolder(repo *SQLiteRepository, dsn string) *Holder {
	return &Holder{repo: repo, dsn: dsn}
}

func (h *Holder) Get() *SQLiteRepository {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.repo
}

func (h *Holder) DSN() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.dsn
}

// Swap installs next as the active repository and returns the previous one with its DSN.
func (h *Holder) Swap(next *SQLiteRepository, nextDSN string) (*SQLiteRepository, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	old, oldDSN := h.repo, h.dsn
	h.repo, h.dsn = next, nextDSN
	return old, oldDSN
}

// PeerDSN returns the alternate on-disk path used for blue-green rebuilds.
func PeerDSN(primaryDSN, currentDSN string) string {
	if IsInMemoryDSN(primaryDSN) {
		return ":memory:"
	}
	peer := primaryDSN + ".next"
	if currentDSN == peer {
		return primaryDSN
	}
	return peer
}

// IsInMemoryDSN reports whether dsn refers to an in-memory SQLite database.
func IsInMemoryDSN(dsn string) bool {
	if dsn == ":memory:" {
		return true
	}
	return strings.Contains(dsn, "mode=memory")
}
