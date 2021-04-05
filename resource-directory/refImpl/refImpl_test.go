package refImpl_test

import (
	"github.com/plgd-dev/cloud/resource-directory/service"
	"github.com/plgd-dev/kit/config"
	"testing"

	"github.com/plgd-dev/cloud/resource-directory/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	authShutdown := oauthTest.New(t, oauthTest.MakeConfig(t))
	defer authShutdown()

	var cfg service.Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	cfg.Clients.OAuthProvider.OAuth.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	cfg.Clients.OAuthProvider.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	cfg.Clients.OAuthProvider.OAuth.Audience = testCfg.OAUTH_MANAGER_AUDIENCE

	got, err := refImpl.Init(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	defer got.Shutdown()
}
