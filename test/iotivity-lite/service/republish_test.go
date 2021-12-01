package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	coapgwService "github.com/plgd-dev/hub/coap-gateway/service"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
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

const (
	signUpKey       = "SignUp"
	signOffKey      = "SignOff"
	signInKey       = "SignIn"
	signOutKey      = "SignOut"
	publishKey      = "Publish"
	unpublishKey    = "Unpublish"
	refreshTokenKey = "RefreshToken"

	accessTokenLifetime time.Duration = time.Second * 20
)

type republishHandler struct {
	callCounter map[string]int
}

func (r *republishHandler) SignUp(coapgwService.CoapSignUpRequest) (coapgwService.CoapSignUpResponse, error) {
	r.callCounter[signUpKey]++
	return coapgwService.CoapSignUpResponse{
		AccessToken:  "access-token",
		UserID:       "1",
		RefreshToken: "refresh-token",
		ExpiresIn:    int64(accessTokenLifetime.Seconds()),
		RedirectURI:  "",
	}, nil
}

func (r *republishHandler) SignOff() error {
	r.callCounter[signOffKey]++
	return nil
}

func (r *republishHandler) SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error) {
	r.callCounter[signInKey]++
	return coapgwService.CoapSignInResp{
		ExpiresIn: int64(accessTokenLifetime.Seconds()),
	}, nil
}

func (r *republishHandler) SignOut(req coapgwService.CoapSignInReq) error {
	r.callCounter[signOutKey]++
	return nil
}

func (r *republishHandler) PublishResources(req coapgwTestService.PublishRequest) error {
	r.callCounter[publishKey]++
	return nil
}

func (r *republishHandler) UnpublishResources(req coapgwTestService.UnpublishRequest) error {
	r.callCounter[unpublishKey]++
	return nil
}

func (r *republishHandler) RefreshToken(req coapgwService.CoapRefreshTokenReq) (coapgwService.CoapRefreshTokenResp, error) {
	r.callCounter[refreshTokenKey]++
	return coapgwService.CoapRefreshTokenResp{
		RefreshToken: "refresh-token",
		AccessToken:  "access-token",
		ExpiresIn:    int64(accessTokenLifetime.Seconds()),
	}, nil
}

func TestRepublishAfterRefresh(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	h := &republishHandler{
		callCounter: make(map[string]int),
	}
	coapShutdown := coapgwTest.SetUp(t, h, func() {
		log.Debugf("%+v", h.callCounter)
		signInCount, ok := h.callCounter[signInKey]
		require.True(t, ok)
		require.True(t, signInCount > 1)
		refreshCount, ok := h.callCounter[refreshTokenKey]
		require.True(t, ok)
		require.True(t, refreshCount > 0)
		publishCount, ok := h.callCounter[publishKey]
		require.True(t, ok)
		require.Equal(t, 1, publishCount)
	})
	defer coapShutdown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	log.Setup(log.Config{Debug: true})
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, nil)
	defer shutdownDevSim()

	for {
		if time.Now().Add(time.Second * 10).After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
}
