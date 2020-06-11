package refImpl

import (
	"testing"

	testAS "github.com/go-ocf/cloud/authorization/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	asShutdown := testAS.SetUp(t)
	defer asShutdown()

	var config Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)
	config.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL

	got, err := Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
