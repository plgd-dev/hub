//go:build test
// +build test

package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestExchangeCacheExecute(t *testing.T) {
	logger := log.NewLogger(log.MakeDefaultConfig())

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	cfg := config.MakeDeviceAuthorization()
	cfg.ClientID = oauthTest.ClientTestRestrictedAuth
	provider, err := oauth2.NewPlgdProvider(context.Background(), cfg, fileWatcher, logger, trace.NewNoopTracerProvider(), "", "")
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
	results := []struct {
		token *oauth2.Token
		err   error
	}{
		{},
		{},
		{},
		{},
		{},
		{},
		{},
		{},
	}
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx].token, results[idx].err = ec.Execute(ctx, provider, code)
		}(i)
	}
	wg.Wait()
	for _, r := range results {
		require.NoError(t, r.err)
		require.NotEmpty(t, r.token)
		require.Equal(t, results[0].token, r.token)
	}

	// remove code from token cache, code will be send to the auth server and should return an error
	ec.Clear()
	_, err = ec.Execute(ctx, provider, code)
	require.Error(t, err)

	// get a new code and auth server should give us a new access token
	code = oauthTest.GetDefaultDeviceAuthorizationCode(t, "")
	token3, err := ec.Execute(ctx, provider, code)
	require.NoError(t, err)
	require.NotEqual(t, results[0].token, token3)
}
