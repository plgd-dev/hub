package refImpl

import (
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/kit/config"
	"testing"

	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
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
	cfg.Clients.OAuthProvider.OAuth.ClientID = oauthService.ClientTest
	cfg.Clients.OAuthProvider.OAuth.Audience = testCfg.OAUTH_MANAGER_AUDIENCE

	got, err := Init(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	defer got.Shutdown()

}
