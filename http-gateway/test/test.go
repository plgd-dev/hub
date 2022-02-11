package test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/service"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func MakeWebConfigurationConfig() service.WebConfiguration {
	return service.WebConfiguration{
		Authority:          "https://" + config.OAUTH_SERVER_HOST,
		HTTPGatewayAddress: "https://" + config.HTTP_GW_HOST,
		WebOAuthClient: service.BasicOAuthClient{
			ClientID: config.OAUTH_MANAGER_CLIENT_ID,
			Audience: config.OAUTH_MANAGER_AUDIENCE,
			Scopes:   []string{"openid", "offline_access"},
		},
		DeviceOAuthClient: service.DeviceOAuthClient{
			BasicOAuthClient: service.BasicOAuthClient{
				ClientID: config.OAUTH_MANAGER_CLIENT_ID,
				Audience: config.OAUTH_MANAGER_AUDIENCE,
				Scopes:   []string{"profile", "openid", "offline_access"},
			},
			ProviderName: config.DEVICE_PROVIDER,
		},
	}
}

func MakeConfig(t *testing.T, enableUI bool) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.HTTP.Authorization = config.MakeAuthorizationConfig()
	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.HTTP_GW_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.WebSocket.StreamBodyLimit = 256 * 1024
	cfg.APIs.HTTP.WebSocket.PingFrequency = 10 * time.Second

	cfg.Clients.GrpcGateway.Connection = config.MakeGrpcClientConfig(config.GRPC_HOST)

	if enableUI {
		cfg.UI.Enabled = true
		cfg.UI.Directory = os.Getenv("TEST_HTTP_GW_WWW_ROOT")
		cfg.UI.WebConfiguration = MakeWebConfigurationConfig()
	}

	err := cfg.Validate()
	require.NoError(t, err)

	fmt.Printf("cfg\n%v\n", cfg.String())

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t, false))
}

func New(t *testing.T, cfg service.Config) func() {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	s, err := service.New(ctx, cfg, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()
	return func() {
		_ = s.Shutdown()
		wg.Wait()
	}
}

func GetContentData(content *pb.Content, desiredContentType string) ([]byte, error) {
	if desiredContentType == uri.ApplicationProtoJsonContentType {
		data, err := protojson.Marshal(content)
		if err != nil {
			return nil, err
		}
		return data, err
	}
	v, err := cbor.ToJSON(content.GetData())
	if err != nil {
		return nil, err
	}
	return []byte(v), err
}
