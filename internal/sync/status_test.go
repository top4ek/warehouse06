package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"warehouse06/internal/gitdate"
)

func TestStatus_SetSyncing(t *testing.T) {
	s := NewStatus()
	assert.False(t, s.Syncing())

	s.SetSyncing(true)
	assert.True(t, s.Syncing())

	s.SetSyncing(false)
	assert.False(t, s.Syncing())
}

func TestStatus_SetSuccess(t *testing.T) {
	s := NewStatus()
	syncedAt := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	commit := gitdate.Commit{
		Hash:        "deadbeef",
		CommittedAt: time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC),
		Subject:     "sync snapshot",
	}

	s.SetSuccess(syncedAt, commit, true)

	require.False(t, s.LastSyncedAt().IsZero())
	assert.True(t, syncedAt.Equal(s.LastSyncedAt()))

	got, ok := s.StorageCommit()
	require.True(t, ok)
	assert.Equal(t, "deadbeef", got.Hash)
	assert.Equal(t, "sync snapshot", got.Subject)
}

func TestStatus_SetSuccess_withoutCommit(t *testing.T) {
	s := NewStatus()
	first := gitdate.Commit{Hash: "first"}
	s.SetSuccess(time.Now(), first, true)

	s.SetSuccess(time.Now(), gitdate.Commit{Hash: "ignored"}, false)

	got, ok := s.StorageCommit()
	require.True(t, ok)
	assert.Equal(t, "first", got.Hash)
}
