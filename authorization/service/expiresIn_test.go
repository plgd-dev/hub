package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoExpiration(t *testing.T) {
	s, ok := ExpiresIn(time.Time{})
	assert.Equal(t, int64(-1), s)
	assert.True(t, ok)
}

func TestExpired(t *testing.T) {
	_, ok := ExpiresIn(time.Now().Add(-time.Minute))
	assert.False(t, ok)
}

func TestExpiresIn(t *testing.T) {
	s, ok := ExpiresIn(time.Now().Add(time.Minute))
	assert.True(t, 55 < s && s <= 60)
	assert.True(t, ok)
}
