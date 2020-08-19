package refImpl_test

import (
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/refImpl"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	var config refImpl.Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)

	got, err := refImpl.Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	defer got.Close()
}
