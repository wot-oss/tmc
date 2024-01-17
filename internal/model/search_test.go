package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearch_FilterType(t *testing.T) {
	// FullMatch is intended to be default value (0)
	assert.True(t, FullMatch == 0)
	assert.True(t, PrefixMatch > 0)
}
