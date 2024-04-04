package log

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"testing"
	"time"

	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	const testStr = "test"

	config := MakeDefaultConfig()
	Setup(config)

	require.NotPanics(t, func() { Debug(testStr) })
	require.NotPanics(t, func() { Info(testStr) })
	require.NotPanics(t, func() {
		Info(testStr, pkgX509.NewError([][]*x509.Certificate{{&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "certName",
			},
		}}}, errors.New(" x509")))
	})
	require.NotPanics(t, func() { Warn(testStr) })
	require.NotPanics(t, func() { Error(testStr) })

	require.NotPanics(t, func() { Debugf(testStr) })
	require.NotPanics(t, func() { Infof(testStr) })
	require.NotPanics(t, func() { Warnf(testStr) })
	require.NotPanics(t, func() { Errorf(testStr) })
	timesStr := []string{"rfc3339nano", "rfc3339", "iso8601", "millis", "nanos", ""}
	for _, str := range timesStr {
		var v TimeEncoderWrapper
		require.NoError(t, v.UnmarshalText([]byte(str)))
		text, err := v.MarshalText()
		require.NoError(t, err)
		require.Equal(t, str, string(text))
	}

	require.InEpsilon(t, float32(1000), DurationToMilliseconds(time.Second), 0.1)
	require.Error(t, LogAndReturnError(errors.New(testStr)))

	cfg := MakeDefaultConfig()
	require.NoError(t, cfg.Validate())
}
