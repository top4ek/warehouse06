package repository

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHolder_GetAndSwap(t *testing.T) {
	primary, err := NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = primary.Close() })

	holder := NewHolder(primary, ":memory:")
	assert.Equal(t, primary, holder.Get())

	secondary, err := NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)

	old, oldDSN := holder.Swap(secondary, ":memory:")
	assert.Equal(t, primary, old)
	assert.Equal(t, ":memory:", oldDSN)
	assert.Equal(t, secondary, holder.Get())

	_ = old.Close()
	_ = secondary.Close()
}

func TestHolder_ConcurrentGet(t *testing.T) {
	repo, err := NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	holder := NewHolder(repo, ":memory:")

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NotNil(t, holder.Get())
		}()
	}
	wg.Wait()
}

func TestPeerDSN(t *testing.T) {
	assert.Equal(t, "warehouse.db.next", PeerDSN("warehouse.db", "warehouse.db"))
	assert.Equal(t, "warehouse.db", PeerDSN("warehouse.db", "warehouse.db.next"))
	assert.Equal(t, ":memory:", PeerDSN(":memory:", ":memory:"))
}
