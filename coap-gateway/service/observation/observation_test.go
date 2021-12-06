package observation_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/go-coap/v2/tcp"
	coapgwService "github.com/plgd-dev/hub/coap-gateway/service"
	"github.com/plgd-dev/hub/coap-gateway/service/observation"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/pkg/sync/task/future"
	"github.com/plgd-dev/hub/test"
	coapgwTestService "github.com/plgd-dev/hub/test/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/test/coap-gateway/test"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type observerHandlerWithCoap struct {
	coapgwTest.DefaultObserverHandler
	t                     *testing.T
	coapConn              *tcp.ClientConn
	setInitializedHandler future.SetFunc
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

	const services = service.SetUpServicesOAuth | service.SetUpServicesId | service.SetUpServicesResourceDirectory |
		service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()
	// log.Setup(log.Config{Debug: true})

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	handlerFuture, setHandler := future.New()
	makeHandler := func(service *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		cfg := coapgwTestService.ServiceHandlerConfig{}
		for _, o := range opts {
			o.Apply(&cfg)
		}
		h := &observerHandlerWithCoap{
			DefaultObserverHandler: coapgwTest.MakeDefaultObserverHandler(tokenLifetime),
			t:                      t,
			coapConn:               cfg.GetCoapConnection(),
			setInitializedHandler:  setHandler,
		}
		return h
	}

	coapShutdown := coapgwTest.SetUp(t, makeHandler, nil)
	defer coapShutdown()

	grpcConn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = grpcConn.Close()
	}()
	grpcClient := pb.NewGrpcGatewayClient(grpcConn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, grpcClient, deviceID, config.GW_HOST, nil)
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
		resourceHref     string
		resourceType     string
		observeInterface string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: testResourceName("invalid"),
			args: args{
				resourceHref: "invalidHref",
			},
			wantErr: true,
		},
		{
			name: testResourceName(resources.ResourceURI),
			args: args{
				resourceHref: resources.ResourceURI,
			},
			want: false,
		},
		{
			name: testResourceName(resources.ResourceURI) + "with resourceType",
			args: args{
				resourceHref: resources.ResourceURI,
				resourceType: resources.ResourceType,
			},
			want: false,
		},
		{
			name: testResourceName(resources.ResourceURI) + "with invalid resourceType",
			args: args{
				resourceHref: resources.ResourceURI,
				resourceType: "invalidResourceType",
			},
			wantErr: true,
		},
		{
			name: testResourceName(device.ResourceURI),
			args: args{
				resourceHref: device.ResourceURI,
			},
			want: true,
		},
		{
			name: testResourceName(device.ResourceURI) + "with resourceType",
			args: args{
				resourceHref: device.ResourceURI,
				resourceType: device.ResourceType,
			},
			want: true,
		},
		{
			name: testResourceName(device.ResourceURI) + "with wrong resourceType",
			args: args{
				resourceHref: device.ResourceURI,
				resourceType: resources.ResourceType,
			},
			wantErr: true,
		},
		{
			name: testResourceName(device.ResourceURI) + "with observeInterface",
			args: args{
				resourceHref:     device.ResourceURI,
				resourceType:     device.ResourceType,
				observeInterface: interfaces.OC_IF_BASELINE,
			},
			want: true,
		},
		{
			name: testResourceName(device.ResourceURI) + "with not supported observeInterface",
			args: args{
				resourceHref:     device.ResourceURI,
				resourceType:     device.ResourceType,
				observeInterface: interfaces.OC_IF_B,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observable, err := observation.IsResourceObservableWithInterface(ctx, handler.coapConn, tt.args.resourceHref,
				tt.args.resourceType, tt.args.observeInterface)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, observable)
		})
	}
}
