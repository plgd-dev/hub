package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/service"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	test "github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config

	cfg.APIs.HTTP.Authorization = config.MakeAuthorizationConfig()
	//cfg.APIs.HTTP.WebSocket.ReadLimit = 8 * 1024
	//cfg.APIs.HTTP.WebSocket.ReadTimeout = time.Second * 4
	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.HTTP_GW_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false

	cfg.Clients.GrpcGateway.Connection = testCfg.MakeGrpcClientConfig(testCfg.GRPC_HOST)

	err := cfg.Validate()
	require.NoError(t, err)

	fmt.Printf("cfg\n%v\n", cfg.String())

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config) func() {
	ctx := context.Background()
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)

	s, err := service.New(ctx, cfg, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Serve()
	}()

	return func() {
		s.Shutdown()
		wg.Wait()
	}
}

func TestRequestHandler_GetResourceLinks(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetResourceLinksRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*events.ResourceLinksPublished
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetResourceLinksRequest{},
			},
			wantErr: false,
			want: []*events.ResourceLinksPublished{
				{
					DeviceId:  deviceID,
					Resources: test.ResourceLinksToResources(deviceID, test.GetAllBackendResourceLinks()),
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	shutdownHttp := New(t, MakeConfig(t))
	defer shutdownHttp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%v/api/v1/resourceLinks", config.HTTP_GW_HOST), nil)
			require.NoError(t, err)
			request.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))
			trans := http.DefaultTransport.(*http.Transport).Clone()
			trans.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
			c := http.Client{
				Transport: trans,
			}
			resp, err := c.Do(request)
			require.NoError(t, err)
			defer resp.Body.Close()

			var links []*events.ResourceLinksPublished
			marshaler := runtime.JSONPb{}
			decoder := marshaler.NewDecoder(resp.Body)
			for {
				var v events.ResourceLinksPublished
				err = service.Unmarshal(resp.StatusCode, decoder, &v)
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				links = append(links, test.CleanUpResourceLinksPublished(&v))
			}
			test.CheckProtobufs(t, tt.want, links, test.RequireToCheckFunc(require.Equal))
		})
	}
}
