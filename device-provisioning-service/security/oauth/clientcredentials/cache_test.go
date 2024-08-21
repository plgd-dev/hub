package clientcredentials_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/security/oauth/clientcredentials"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNew(t *testing.T) {
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesOAuth)
	defer hubShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	logger := log.NewLogger(log.Config{})

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()

	cfg := test.MakeAuthorizationConfig()
	p, err := cfg.ToProto()
	require.NoError(t, err)
	require.NoError(t, p.Validate())
	got, err := clientcredentials.New(ctx, cfg.Provider.Config, fileWatcher, logger, noop.NewTracerProvider(), time.Millisecond*10)
	require.NoError(t, err)
	defer got.Close()

	tokenA, err := got.GetToken(ctx, "A", map[string]string{
		"key": "value",
	}, nil)
	require.NoError(t, err)
	tokenB, err := got.GetToken(ctx, "B", map[string]string{
		"key": "value",
	}, nil)
	require.NoError(t, err)
	require.NotEqual(t, tokenA, tokenB)
	dupTokenA, err := got.GetToken(ctx, "A", map[string]string{
		"key": "value",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, tokenA, dupTokenA)
	time.Sleep(time.Second)
	tokenAwithoutCache, err := got.GetTokenFromOAuth(ctx, map[string]string{
		"key": "value",
	}, map[string]interface{}{
		"sub": "1",
	})
	require.NoError(t, err)
	require.NotEqual(t, tokenA, tokenAwithoutCache)
	_, err = got.GetToken(ctx, "C", map[string]string{
		"key": "value",
	}, map[string]interface{}{
		"sub": "1",
	})
	require.NoError(t, err)
	_, err = got.GetToken(ctx, "C", map[string]string{
		"key": "value",
	}, map[string]interface{}{
		"notExist": 123,
	})
	require.Error(t, err)
	_, err = got.GetToken(ctx, "C", map[string]string{
		"key": "value",
	}, map[string]interface{}{
		"sub": 123,
	})
	require.Error(t, err)
}
