package refImpl

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	authShutdown := oauthTest.New(t, oauthTest.MakeConfig(t))
	defer authShutdown()

	var config Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)
	config.Service.Addr = "localhost:20006"
	config.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	config.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	config.Service.OAuth.Audience = testCfg.OAUTH_MANAGER_AUDIENCE
	config.Service.Nats = testCfg.MakeSubscriberConfig()

	got, err := Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	got.Shutdown()
}
