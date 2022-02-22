package log

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	const testStr = "test"

	config := MakeDefaultConfig()
	Setup(config)

	assert.NotPanics(t, func() { Debug(testStr) })
	assert.NotPanics(t, func() { Info(testStr) })
	assert.NotPanics(t, func() { Warn(testStr) })
	assert.NotPanics(t, func() { Error(testStr) })

	assert.NotPanics(t, func() { Debugf(testStr) })
	assert.NotPanics(t, func() { Infof(testStr) })
	assert.NotPanics(t, func() { Warnf(testStr) })
	assert.NotPanics(t, func() { Errorf(testStr) })
	var timesStr = []string{"rfc3339nano", "rfc3339", "iso8601", "millis", "nanos", ""}
	for _, str := range timesStr {
		var v TimeEncoderWrapper
		assert.NoError(t, v.UnmarshalText([]byte(str)))
		text, err := v.MarshalText()
		assert.NoError(t, err)
		assert.Equal(t, str, string(text))
	}

	assert.Equal(t, float32(1000), DurationToMilliseconds(time.Second))
	assert.Error(t, LogAndReturnError(fmt.Errorf(testStr)))

	cfg := MakeDefaultConfig()
	assert.NoError(t, cfg.Validate())
}
