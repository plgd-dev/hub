package refImpl

import (
	"os"
	"testing"

	testAS "github.com/plgd-dev/cloud/authorization/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	asShutdown := testAS.SetUp(t)
	defer asShutdown()

	var config Config
	os.Setenv("OAUTH_CALLBACK", "OAUTH_CALLBACK")
	os.Setenv("EVENTS_URL", "EVENTS_URL")
	err := envconfig.Process("", &config)
	require.NoError(t, err)
	config.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL

	got, err := Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	got.Shutdown()
}
