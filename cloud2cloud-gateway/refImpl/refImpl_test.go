package refImpl

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	oauthTest "github.com/plgd-dev/cloud/oauth-server/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	authShutdown := oauthTest.New(t, oauthTest.MakeConfig(t))
	defer authShutdown()

	var config Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)
	config.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	config.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	config.Service.OAuth.Audience = testCfg.OAUTH_MANAGER_AUDIENCE

	got, err := Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
