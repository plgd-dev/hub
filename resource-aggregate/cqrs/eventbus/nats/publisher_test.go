package nats

import (
	"testing"

	"github.com/plgd-dev/kit/security/certManager"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPublisher(t *testing.T) {
	var config certManager.Config
	err := envconfig.Process("DIAL", &config)
	assert.NoError(t, err)

	dialCertManager, err := certManager.NewCertManager(config)
	require.NoError(t, err)

	tlsConfig := dialCertManager.GetClientTLSConfig()

	bus, err := NewPublisher(Config{
		URL: "nats://localhost:4222",
	}, WithTLS(tlsConfig))
	require.NoError(t, err)
	assert.NotNil(t, bus)
	defer bus.Close()
}
