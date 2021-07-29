package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoExpiration(t *testing.T) {
	s, ok := ValidUntil(time.Time{})
	assert.Equal(t, int64(0), s)
	assert.True(t, ok)
}

func TestExpired(t *testing.T) {
	_, ok := ValidUntil(time.Now().Add(-time.Minute))
	assert.False(t, ok)
}

func TestExpiresIn(t *testing.T) {
	_, ok := ValidUntil(time.Now().Add(time.Minute))
	assert.True(t, ok)
}
