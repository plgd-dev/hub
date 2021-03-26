package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	config := Config{Debug: false}
	Setup(config)

	assert.NotPanics(t, func() { Debug("test") })
	assert.NotPanics(t, func() { Info("test") })
	assert.NotPanics(t, func() { Warn("test") })
	assert.NotPanics(t, func() { Error("test") })

	if config.Debug {
		assert.Panics(t, func() { DPanic("test") })
	}
	assert.Panics(t, func() { Panic("test") })

	//assert.NotPanics(t, func() { Fatal("test") })

	config.Debug = true
	Setup(config)

	assert.NotPanics(t, func() { Debugf("test") })
	assert.NotPanics(t, func() { Infof("test") })
	assert.NotPanics(t, func() { Warnf("test") })
	assert.NotPanics(t, func() { Errorf("test") })

	if config.Debug {
		assert.Panics(t, func() { DPanicf("test") })
	}
	assert.Panics(t, func() { Panicf("test") })

}
