package repository

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveStartupDSN_memory(t *testing.T) {
	assert.Equal(t, ":memory:", ResolveStartupDSN(":memory:"))
}

func TestResolveStartupDSN_primaryOnly(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "warehouse.db")
	require.NoError(t, os.WriteFile(primary, []byte("primary"), 0o644))

	assert.Equal(t, primary, ResolveStartupDSN(primary))
}

func TestResolveStartupDSN_peerOnly(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "warehouse.db")
	peer := primary + ".next"
	require.NoError(t, os.WriteFile(peer, []byte("peer"), 0o644))

	assert.Equal(t, peer, ResolveStartupDSN(primary))
}

func TestResolveStartupDSN_prefersNewerPeer(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "warehouse.db")
	peer := primary + ".next"

	require.NoError(t, os.WriteFile(primary, []byte("old"), 0o644))
	require.NoError(t, os.WriteFile(peer, []byte("newer-peer-data"), 0o644))

	now := time.Now()
	require.NoError(t, os.Chtimes(primary, now.Add(-time.Hour), now.Add(-time.Hour)))
	require.NoError(t, os.Chtimes(peer, now, now))

	assert.Equal(t, peer, ResolveStartupDSN(primary))
}

func TestResolveStartupDSN_prefersLargerWhenSameModTime(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "warehouse.db")
	peer := primary + ".next"

	require.NoError(t, os.WriteFile(primary, []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(peer, []byte("much-larger"), 0o644))

	ts := time.Now()
	require.NoError(t, os.Chtimes(primary, ts, ts))
	require.NoError(t, os.Chtimes(peer, ts, ts))

	assert.Equal(t, peer, ResolveStartupDSN(primary))
}
