package nats

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/kit/security/certificateManager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPublisher(t *testing.T) {
	var config certificateManager.Config
	err := envconfig.Process("DIAL", &config)
	assert.NoError(t, err)

	dialCertManager, err := certificateManager.NewCertificateManager(config)
	require.NoError(t, err)

	tlsConfig := dialCertManager.GetClientTLSConfig()

	bus, err := NewPublisher(Config{
		URL: "nats://localhost:4222",
	}, WithTLS(tlsConfig))
	require.NoError(t, err)
	assert.NotNil(t, bus)
	defer bus.Close()
}
