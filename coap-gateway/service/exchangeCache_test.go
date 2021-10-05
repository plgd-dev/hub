package service_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/coap-gateway/service"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/security/oauth2"
	"github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func TestExchangeCacheExecute(t *testing.T) {
	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	cfg := config.MakeDeviceAuthorization()
	cfg.ClientID = oauthService.ClientTestRestrictedAuth
	provider, err := oauth2.NewPlgdProvider(context.Background(), cfg, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	code := oauthTest.GetDefaultDeviceAuthorizationCode(t, "")
	// repeated use of the same auth code within a minute should return an error
	_, err = provider.Exchange(ctx, code)
	require.NoError(t, err)
	_, err = provider.Exchange(ctx, code)
	require.Error(t, err)

	code = oauthTest.GetDefaultDeviceAuthorizationCode(t, "")
	// token cache prevents multiple requests with the same auth code being sent to the oauth server
	ec := service.NewExchangeCache()
	token1, err := ec.Execute(ctx, provider, code)
	require.NoError(t, err)
	token2, err := ec.Execute(ctx, provider, code)
	require.NoError(t, err)
	require.Equal(t, token1, token2)

	// remove code from token cache, code will be send to the auth server and should return an error
	ec.Clear()
	_, err = ec.Execute(ctx, provider, code)
	require.Error(t, err)

	// get a new code and auth server should give us a new access token
	code = oauthTest.GetDefaultDeviceAuthorizationCode(t, "")
	token3, err := ec.Execute(ctx, provider, code)
	require.NoError(t, err)
	require.NotEqual(t, token1, token3)
}
