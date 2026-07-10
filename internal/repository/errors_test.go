package repository

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNotFound(t *testing.T) {
	assert.True(t, IsNotFound(sql.ErrNoRows))
	assert.False(t, IsNotFound(errors.New("other")))
	assert.False(t, IsNotFound(nil))
}
