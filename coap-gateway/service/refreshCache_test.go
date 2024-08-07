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
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func getProvider(t *testing.T, fileWatcher *fsnotify.Watcher, logger log.Logger) *oauth2.PlgdProvider {
	cfg := config.MakeDeviceAuthorization()
	cfg.ClientID = oauthTest.ClientTestRestrictedAuth
	provider, err := oauth2.NewPlgdProvider(context.Background(), cfg, fileWatcher, logger, noop.NewTracerProvider(), "", "", nil)
	require.NoError(t, err)
	return provider
}

func TestRefreshCacheExecute(t *testing.T) {
	logger := log.NewLogger(log.MakeDefaultConfig())

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	provider1 := getProvider(t, fileWatcher, logger)
	defer provider1.Close()
	code := oauthTest.GetAuthorizationCode(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestRestrictedAuth, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	token1, err := provider1.Exchange(ctx, code)
	require.NoError(t, err)
	require.NotEmpty(t, token1.RefreshToken)

	_, err = provider1.Refresh(ctx, token1.RefreshToken)
	require.NoError(t, err)
	_, err = provider1.Refresh(ctx, token1.RefreshToken)
	require.Error(t, err)

	code = oauthTest.GetAuthorizationCode(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestRestrictedAuth, "", "")
	token2, err := provider1.Exchange(ctx, code)
	require.NoError(t, err)
	require.NotEmpty(t, token2.RefreshToken)
	require.NotEqual(t, token1.RefreshToken, token2.RefreshToken)

	provider2 := getProvider(t, fileWatcher, logger)
	defer provider2.Close()
	provider3 := getProvider(t, fileWatcher, logger)
	defer provider3.Close()
	providers := map[string]*oauth2.PlgdProvider{
		"1": provider1,
		"2": provider2,
		"3": provider3,
	}

	taskQueue, err := queue.New(queue.Config{
		GoPoolSize: 8,
		Size:       8,
	})
	require.NoError(t, err)

	code = oauthTest.GetAuthorizationCode(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestRestrictedAuth, "", "")
	token3, err := provider2.Exchange(ctx, code)
	require.NoError(t, err)
	require.NotEqual(t, token3.RefreshToken, token1.RefreshToken)
	require.NotEqual(t, token3.RefreshToken, token2.RefreshToken)
	rc := service.NewRefreshCache()
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
			results[idx].token, results[idx].err = rc.Execute(ctx, providers, taskQueue, token3.RefreshToken, log.Get())
		}(i)
	}
	wg.Wait()
	for _, r := range results {
		require.NoError(t, r.err)
		require.NotEmpty(t, r.token)
		require.Equal(t, results[0].token, r.token)
	}

	rc.Clear()
	_, err = rc.Execute(ctx, providers, taskQueue, token3.RefreshToken, log.Get())
	require.Error(t, err)

	code = oauthTest.GetAuthorizationCode(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestRestrictedAuth, "", "")
	token6, err := provider3.Exchange(ctx, code)
	require.NoError(t, err)
	require.NotEqual(t, token6.RefreshToken, token1.RefreshToken)
	require.NotEqual(t, token6.RefreshToken, token2.RefreshToken)
	require.NotEqual(t, token6.RefreshToken, token3.RefreshToken)
	token7, err := rc.Execute(ctx, providers, taskQueue, token6.RefreshToken, log.Get())
	require.NoError(t, err)
	require.NotEqual(t, results[0].token, token7)
}
