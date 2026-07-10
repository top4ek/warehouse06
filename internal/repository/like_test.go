package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeLike(t *testing.T) {
	assert.Equal(t, `vec\%tor`, escapeLike("vec%tor"))
	assert.Equal(t, `a\_b`, escapeLike("a_b"))
	assert.Equal(t, `a\\b`, escapeLike(`a\b`))
}
