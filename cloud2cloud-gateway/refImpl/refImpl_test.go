package refImpl

import (
	"github.com/plgd-dev/cloud/cloud2cloud-gateway/service"
	"github.com/plgd-dev/kit/config"
	"testing"

	testAS "github.com/plgd-dev/cloud/authorization/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	asShutdown := testAS.SetUp(t)
	defer asShutdown()

	var cfg service.Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	cfg.Clients.OAuthProvider.OAuthConfig.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL

	got, err := Init(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
