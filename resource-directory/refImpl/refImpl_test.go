package refImpl_test

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	oauthService "github.com/plgd-dev/cloud/oauth-server/service"
	oauthTest "github.com/plgd-dev/cloud/oauth-server/test"
	"github.com/plgd-dev/cloud/resource-directory/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	authShutdown := oauthTest.New(t, oauthTest.MakeConfig(t))
	defer authShutdown()

	var config refImpl.Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)
	config.Service.OAuth.ClientID = oauthService.ClientTest
	config.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL

	got, err := refImpl.Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	defer got.Close()
}
