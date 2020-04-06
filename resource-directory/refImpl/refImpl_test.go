package refImpl

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	var config Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)

	got, err := Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
