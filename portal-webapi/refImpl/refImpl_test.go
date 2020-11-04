package refImpl

import (
	"testing"

	"github.com/plgd-dev/kit/config"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	var cfg Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	got, err := Init(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
