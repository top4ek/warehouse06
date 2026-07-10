package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"warehouse06/internal/domain"
)

func TestSQLiteRepository_MemoryDSN_SharesDataAcrossQueries(t *testing.T) {
	repo, err := NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	ctx := context.Background()
	require.NoError(t, repo.SaveEntriesAndAuthors(ctx, []*domain.Entry{
		{
			Path: "vector06c/767",
			Name: "8 Bit Snail",
			Type: domain.EntryTypeDirectory,
		},
	}, nil))

	entry, err := repo.GetEntryByPath(ctx, "vector06c/767")
	require.NoError(t, err)
	assert.Equal(t, "8 Bit Snail", entry.Name)
}
