package refImpl_test

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/cloud/resource-directory/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	authShutdown := oauthTest.New(t, oauthTest.MakeConfig(t))
	defer authShutdown()

	var config refImpl.Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)
	config.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	config.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	config.Service.OAuth.Audience = testCfg.OAUTH_MANAGER_AUDIENCE

	got, err := refImpl.Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	defer got.Close()
}
