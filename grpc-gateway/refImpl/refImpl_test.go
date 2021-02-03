package refImpl_test

import (
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/refImpl"
	"github.com/plgd-dev/cloud/grpc-gateway/service"
	"github.com/plgd-dev/kit/config"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	var cfg service.Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	got, err := refImpl.Init(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	defer got.Close()
}
