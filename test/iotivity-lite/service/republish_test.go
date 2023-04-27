package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	coapgwTestService "github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/test/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/test/config"
	iotService "github.com/plgd-dev/hub/v2/test/iotivity-lite/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRepublishAfterRefresh(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	atLifetime := time.Second * 20
	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	makeHandler := func(s *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return iotService.NewCoapHandlerWithCounter(int64(atLifetime.Seconds()))
	}
	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*iotService.CoapHandlerWithCounter)
		log.Debugf("%+v", h.CallCounter.Data)
		signInCount, ok := h.CallCounter.Data[iotService.SignInKey]
		require.True(t, ok)
		require.True(t, signInCount > 1)
		refreshCount, ok := h.CallCounter.Data[iotService.RefreshTokenKey]
		require.True(t, ok)
		require.True(t, refreshCount > 0)
		publishCount, ok := h.CallCounter.Data[iotService.PublishKey]
		require.True(t, ok)
		require.Equal(t, 1, publishCount)
	}

	coapShutdown := coapgwTest.SetUp(t, makeHandler, validateHandler)
	defer coapShutdown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	// _, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	defer shutdownDevSim()

	for {
		if time.Now().Add(time.Second * 10).After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
}
