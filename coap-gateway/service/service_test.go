//go:build test
// +build test

package service_test

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"testing"

	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	test "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v3"
)

type testDevsObs struct {
	err atomic.Error
	ch  chan client.DevicesObservationEvent
}

func (t *testDevsObs) Error(err error) {
	t.err.Store(err)
}

func (t *testDevsObs) Handle(ctx context.Context, event client.DevicesObservationEvent) error {
	t.ch <- event
	return nil
}

func (t *testDevsObs) OnClose() {}

func TestServiceConfig(t *testing.T) {
	cfg := coapgwTest.MakeConfig(t)
	cfg.APIs.COAP.InjectedCOAPConfig.TLSConfig.IdentityPropertiesRequired = false
	cfg.APIs.COAP.Authorization.DeviceIDClaim = "di"
	require.Error(t, cfg.Validate())
	cfg.APIs.COAP.InjectedCOAPConfig.TLSConfig.IdentityPropertiesRequired = true
	cfg.APIs.COAP.Authorization.DeviceIDClaim = ""
	require.NoError(t, cfg.Validate())
	var cfg2 coapgwService.Config
	fmt.Printf(cfg.String())
	err := yaml.Unmarshal([]byte(cfg.String()), &cfg2)
	require.NoError(t, err)
	require.NoError(t, cfg2.Validate())
	require.Equal(t, cfg.String(), cfg2.String())
}

func TestServiceConfigWithDataScheme(t *testing.T) {
	cfg := coapgwTest.MakeConfig(t)
	data, err := cfg.APIs.COAP.Config.TLS.Embedded.CertFile.Read()
	require.NoError(t, err)
	cfg.APIs.COAP.Config.TLS.Embedded.CertFile = urischeme.URIScheme("data:;base64," + base64.StdEncoding.EncodeToString(data))
	fmt.Printf("cfg: %v\n", cfg.String())
	require.NoError(t, cfg.Validate())
	var cfg2 coapgwService.Config
	err = yaml.Unmarshal([]byte(cfg.String()), &cfg2)
	require.NoError(t, cfg2.Validate())
	require.Equal(t, cfg.String(), cfg2.String())

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesId | service.SetUpServicesResourceDirectory |
		service.SetUpServicesGrpcGateway | service.SetUpServicesResourceAggregate
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	coapShutdown := coapgwTest.New(t, cfg)
	defer coapShutdown()
}

func TestShutdownServiceWithDeviceIssue627(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesId | service.SetUpServicesResourceDirectory |
		service.SetUpServicesGrpcGateway | service.SetUpServicesResourceAggregate
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	coapShutdown := coapgwTest.SetUp(t)
	defer coapShutdown()

	grpcConn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = grpcConn.Close()
	}()
	grpcClient := client.New(pb.NewGrpcGatewayClient(grpcConn))

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, pb.NewGrpcGatewayClient(grpcConn), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	ch := make(chan client.DevicesObservationEvent, 1000)

	v := testDevsObs{
		ch: ch,
	}

	observationID, err := grpcClient.ObserveDevices(ctx, &v)
	require.NoError(t, err)
	defer func(observationID string) {
		err := grpcClient.StopObservingDevices(observationID)
		require.NoError(t, err)
		require.NoError(t, v.err.Load())
	}(observationID)

	coapShutdown()

	for {
		select {
		case e := <-ch:
			if e.Event != client.DevicesObservationEvent_OFFLINE {
				continue
			}
			require.Len(t, e.DeviceIDs, 1)
			require.Equal(t, deviceID, e.DeviceIDs[0])
			return
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		}
	}
}
