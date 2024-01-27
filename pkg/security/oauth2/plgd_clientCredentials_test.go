package oauth2_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func getPlgdProvider(t *testing.T, cfg oauth2.Config, ownerClaim, deviceIDclaim string) *oauth2.PlgdProvider {
	logger := log.NewLogger(log.MakeDefaultConfig())

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	provider, err := oauth2.NewPlgdProvider(context.Background(), cfg, fileWatcher, logger, noop.NewTracerProvider(), ownerClaim, deviceIDclaim)
	require.NoError(t, err)
	return provider
}

func getPlgdProviderConfig() oauth2.Config {
	cfg := config.MakeDeviceAuthorization()
	cfg.GrantType = oauth.ClientCredentials
	return cfg
}

func getToken(t *testing.T, claimOverrides map[string]interface{}) string {
	return oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, claimOverrides)
}

func TestPlgdProviderExchange(t *testing.T) {
	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	type args struct {
		cfg           oauth2.Config
		ownerClaim    string
		deviceIDclaim string
		token         string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid token",
			args: args{
				cfg: getPlgdProviderConfig(),
			},
			wantErr: true,
		},
		{
			name: "invalid deviceIDClaim",
			args: args{
				cfg:           getPlgdProviderConfig(),
				deviceIDclaim: "deviceIDClaim",
				token:         getToken(t, map[string]interface{}{"deviceIDClaim": 42}),
			},
			wantErr: true,
		},
		{
			name: "missing deviceIDClaim",
			args: args{
				cfg:           getPlgdProviderConfig(),
				deviceIDclaim: "deviceIDClaim",
				token:         getToken(t, nil),
			},
			wantErr: true,
		},
		{
			name: "invalid ownerClaim",
			args: args{
				cfg:        getPlgdProviderConfig(),
				ownerClaim: "ownerClaim",
				token:      getToken(t, map[string]interface{}{"ownerClaim": 42}),
			},
			wantErr: true,
		},
		{
			name: "missing ownerClaim",
			args: args{
				cfg:        getPlgdProviderConfig(),
				ownerClaim: "ownerClaim",
				token:      getToken(t, nil),
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				cfg:        getPlgdProviderConfig(),
				ownerClaim: config.OWNER_CLAIM,
				token:      getToken(t, nil),
			},
			wantErr: false,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := getPlgdProvider(t, tt.args.cfg, tt.args.ownerClaim, tt.args.deviceIDclaim)

			_, err := provider.Exchange(ctx, tt.args.token)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
