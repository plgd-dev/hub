package observation_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/resources"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/observation"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/coap"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"github.com/plgd-dev/hub/v2/test"
	coapgwTestService "github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/test/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type observerHandlerWithCoap struct {
	t                     *testing.T
	coapConn              *coapTcpClient.Conn
	setInitializedHandler future.SetFunc
	coapgwTest.DefaultObserverHandler
}

func (h *observerHandlerWithCoap) SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error) {
	resp, err := h.DefaultObserverHandler.SignIn(req)
	require.NoError(h.t, err)
	h.setInitializedHandler(h, nil)
	return resp, nil
}

func TestIsResourceObservableWithInterface(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	const services = service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesOAuth | service.SetUpServicesId | service.SetUpServicesResourceDirectory |
		service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	handlerFuture, setHandler := future.New()
	makeHandler := func(_ *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		cfg := coapgwTestService.ServiceHandlerConfig{}
		for _, o := range opts {
			o.Apply(&cfg)
		}
		h := &observerHandlerWithCoap{
			DefaultObserverHandler: coapgwTest.MakeDefaultObserverHandler(int64(tokenLifetime.Seconds())),
			t:                      t,
			coapConn:               cfg.GetCoapConnection(),
			setInitializedHandler:  setHandler,
		}
		return h
	}

	coapShutdown := coapgwTest.SetUp(t, makeHandler, nil)
	defer coapShutdown()

	grpcConn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = grpcConn.Close()
	}()
	grpcClient := pb.NewGrpcGatewayClient(grpcConn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, grpcClient, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	defer shutdownDevSim()

	// wait for handler with coap connection to be initialized
	h, err := handlerFuture.Get(ctx)
	require.NoError(t, err)
	handler, ok := h.(*observerHandlerWithCoap)
	require.True(t, ok)
	require.NotNil(t, handler)

	testResourceName := func(href string) string {
		return "resource (" + href + ") "
	}
	type args struct {
		href string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: testResourceName(resources.ResourceURI),
			args: args{
				href: resources.ResourceURI,
			},
		},
		{
			name: testResourceName(device.ResourceURI),
			args: args{
				href: device.ResourceURI,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links, _, err := coap.GetResourceLinksWithLinkInterface(ctx, handler.coapConn, tt.args.href)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			observable, err := observation.IsDiscoveryResourceObservable(links)
			require.NoError(t, err)
			require.Equal(t, tt.want, observable)
		})
	}
}
