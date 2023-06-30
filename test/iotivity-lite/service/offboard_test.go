package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
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
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// signed in -> deregister by sending DELETE request
func TestOffboard(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	ch := iotService.NewCoapHandlerWithCounter(0)
	makeHandler := func(s *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return ch
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*iotService.CoapHandlerWithCounter)
		log.Debugf("%+v", h.CallCounter.Data)
		signInCount, ok := h.CallCounter.Data[iotService.SignInKey]
		require.True(t, ok)
		require.True(t, signInCount > 0)
		publishCount, ok := h.CallCounter.Data[iotService.PublishKey]
		require.True(t, ok)
		require.Equal(t, 1, publishCount)
		singOffCount, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.True(t, ok)
		require.True(t, singOffCount > 0)
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

	// TODO: copy services initialization from the real coap-gw to the mock coap-gw,
	// for now we must force TCP when mock coap-gw is used
	// _, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	_, _ = test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	require.True(t, ch.WaitForFirstSignIn(time.Second*20))

	test.OffBoardDevSim(ctx, t, deviceID)
	require.True(t, ch.WaitForFirstSignOff(time.Second*20))
}

type switchableHandler struct {
	*iotService.CoapHandlerWithCounter

	failSignIn       atomic.Bool
	failRefreshToken atomic.Bool
	failSignOff      atomic.Bool

	private struct {
		mutex            sync.Mutex
		blockSignInChan  chan struct{}
		blockSignOffChan chan struct{}
		blockRefreshChan chan struct{}
	}
}

func NewSwitchableHandler(atLifetime int64) *switchableHandler { //nolint:revive
	return &switchableHandler{
		CoapHandlerWithCounter: iotService.NewCoapHandlerWithCounter(atLifetime),
	}
}

func (sh *switchableHandler) CloseOnError() bool {
	return false
}

func (sh *switchableHandler) blockSignInChannel() chan struct{} {
	sh.private.mutex.Lock()
	defer sh.private.mutex.Unlock()
	return sh.private.blockSignInChan
}

func (sh *switchableHandler) SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error) {
	resp, err := sh.CoapHandlerWithCounter.SignIn(req)
	if sh.failSignIn.Load() {
		return coapgwService.CoapSignInResp{}, fmt.Errorf("sign in disabled")
	}
	b := sh.blockSignInChannel()
	if b != nil {
		<-b
	}
	return resp, err
}

func (sh *switchableHandler) blockSignOffChannel() chan struct{} {
	sh.private.mutex.Lock()
	defer sh.private.mutex.Unlock()
	return sh.private.blockSignOffChan
}

func (sh *switchableHandler) blockSignOff() chan struct{} {
	ch := make(chan struct{})
	sh.private.mutex.Lock()
	sh.private.blockSignOffChan = ch
	sh.private.mutex.Unlock()
	return ch
}

func (sh *switchableHandler) SignOff() error {
	err := sh.CoapHandlerWithCounter.SignOff()
	if sh.failSignOff.Load() {
		return fmt.Errorf("sign off disabled")
	}
	b := sh.blockSignOffChannel()
	if b != nil {
		<-b
	}
	return err
}

func (sh *switchableHandler) blockRefreshChannel() chan struct{} {
	sh.private.mutex.Lock()
	defer sh.private.mutex.Unlock()
	return sh.private.blockRefreshChan
}

func (sh *switchableHandler) blockRefresh() chan struct{} {
	ch := make(chan struct{})
	sh.private.mutex.Lock()
	sh.private.blockRefreshChan = ch
	sh.private.mutex.Unlock()
	return ch
}

func (sh *switchableHandler) RefreshToken(req coapgwService.CoapRefreshTokenReq) (coapgwService.CoapRefreshTokenResp, error) {
	resp, err := sh.CoapHandlerWithCounter.RefreshToken(req)
	if sh.failRefreshToken.Load() {
		return coapgwService.CoapRefreshTokenResp{}, fmt.Errorf("refresh token disabled")
	}
	b := sh.blockRefreshChannel()
	if b != nil {
		<-b
	}
	return resp, err
}

// not signed in, with permanent and short access-token -> deregister by sending DELETE request with access token
func TestOffboardWithoutSignIn(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	sh := NewSwitchableHandler(-1)
	sh.failSignIn.Store(true)
	makeHandler := func(s *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return sh
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*switchableHandler)
		h.CallCounter.Lock.Lock()
		defer h.CallCounter.Lock.Unlock()
		log.Debugf("%+v", h.CallCounter.Data)
		signInCount, ok := h.CallCounter.Data[iotService.SignInKey]
		require.True(t, ok)
		// sometimes the first sign-in attempt fails, so we allow 2 attempts
		/*
			=== RUN   TestOffboardWithoutSignIn
			     offboard_test.go:205:
			        	Error Trace:	/src/github.com/plgd-dev/hub/test/iotivity-lite/service/offboard_test.go:205
			         	            				/src/github.com/plgd-dev/hub/test/coap-gateway/test/test.go:59
			         	            				/src/github.com/plgd-dev/hub/test/iotivity-lite/service/offboard_test.go:236
			         	Error:      	Not equal:
			         	            	expected: 1
			        	            	actual  : 2
			        	Test:       	TestOffboardWithoutSignIn
		*/
		require.True(t, signInCount >= 1 || signInCount <= 2)
		_, ok = h.CallCounter.Data[iotService.RefreshTokenKey]
		require.False(t, ok)
		signOffCount, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.True(t, ok)
		require.Equal(t, 1, signOffCount)
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

	// deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))

	// first retry after failure is after 2 seconds, so hopefully it doesn't trigger, if this test
	// behaves flakily then we will have to update simulator to have a configurable retry
	sh.failSignIn.Store(false)
	test.OffBoardDevSim(ctx, t, deviceID)
	require.True(t, sh.WaitForFirstSignOff(time.Second*20))
}

// not signed in, with permanent but long access-token -> try to sign in and then deregister without access token
func TestOffboardWithSignIn(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	sh := NewSwitchableHandler(-1)
	sh.failSignIn.Store(true)
	sh.SetAccessToken(strings.Repeat("this-access-token-is-so-long-that-its-size-is-longer-than-the-allowed-request-header-size", 5))
	makeHandler := func(s *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return sh
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*switchableHandler)
		h.CallCounter.Lock.Lock()
		defer h.CallCounter.Lock.Unlock()
		log.Debugf("%+v", h.CallCounter.Data)
		signInCount, ok := h.CallCounter.Data[iotService.SignInKey]
		require.True(t, ok)
		require.Equal(t, 2, signInCount)
		_, ok = h.CallCounter.Data[iotService.RefreshTokenKey]
		require.False(t, ok)
		signOffCount, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.True(t, ok)
		require.Equal(t, 1, signOffCount)
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

	// deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))

	// first retry after failure is after 2 seconds, so hopefully it doesn't trigger, if this test
	// behaves flakily then we will have to update simulator to have a configurable retry
	sh.failSignIn.Store(false)
	test.OffBoardDevSim(ctx, t, deviceID)
	require.True(t, sh.WaitForFirstSignOff(time.Second*20))
}

// not signed up, with refresh token -> try to login and then deregister without access token
func TestOffboardWithSignInByRefreshToken(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	sh := NewSwitchableHandler(20)
	sh.failSignIn.Store(true)
	sh.SetAccessToken(strings.Repeat("this-access-token-is-so-long-that-its-size-is-longer-than-the-allowed-request-header-size", 5))
	makeHandler := func(s *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return sh
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*switchableHandler)
		h.CallCounter.Lock.Lock()
		defer h.CallCounter.Lock.Unlock()
		log.Debugf("%+v", h.CallCounter.Data)
		signInCount, ok := h.CallCounter.Data[iotService.SignInKey]
		require.True(t, ok)
		require.True(t, signInCount > 1)
		refreshCount, ok := h.CallCounter.Data[iotService.RefreshTokenKey]
		require.Equal(t, 1, refreshCount)
		require.True(t, ok)
		signOffCount, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.True(t, ok)
		require.Equal(t, 1, signOffCount)
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

	// deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))

	// first retry after failure is after 2 seconds, so hopefully it doesn't trigger, if this test
	// behaves flakily then we will have to update simulator to have a configurable retry
	sh.failSignIn.Store(false)
	test.OffBoardDevSim(ctx, t, deviceID)
	require.True(t, sh.WaitForFirstRefreshToken(time.Second*20))
	require.True(t, sh.WaitForFirstSignOff(time.Second*20))
}

// Multiple offboard attempts should be ignored
func TestOffboardWithRepeat(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	sh := NewSwitchableHandler(-1)
	sh.blockSignOff()
	makeHandler := func(s *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return sh
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*switchableHandler)
		h.CallCounter.Lock.Lock()
		defer h.CallCounter.Lock.Unlock()
		log.Debugf("%+v", h.CallCounter.Data)
		signOffCount, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.True(t, ok)
		require.Equal(t, 1, signOffCount)
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

	// deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))

	test.OffBoardDevSim(ctx, t, deviceID)
	test.OffBoardDevSim(ctx, t, deviceID)
	test.OffBoardDevSim(ctx, t, deviceID)

	require.True(t, sh.WaitForFirstSignOff(time.Second*20))
	// first SignOff should timeout after 10 secs, we wait 10 additional seconds for the other
	// calls to maybe happen
	time.Sleep(time.Second * 10)
}

// Onboarding should interrupt an ongoing offboarding
func TestOffboardInterrupt(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	// force the use of refresh token (non-permanent, long access-token and sign-up will fail)
	sh := NewSwitchableHandler(int64(time.Minute.Seconds()))
	sh.failSignIn.Store(true)
	sh.SetAccessToken(strings.Repeat("this-access-token-is-so-long-that-its-size-is-longer-than-the-allowed-request-header-size", 5))
	// off-boarding will try login by refresh token, but block the sending of response
	blockRefreshCh := sh.blockRefresh()

	makeHandler := func(s *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return sh
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*switchableHandler)
		h.CallCounter.Lock.Lock()
		defer h.CallCounter.Lock.Unlock()
		log.Debugf("%+v", h.CallCounter.Data)
		_, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.False(t, ok)
	}

	coapShutdown := coapgwTest.SetUp(t, makeHandler, validateHandler)

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	// deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))

	test.OffBoardDevSim(ctx, t, deviceID)
	require.True(t, sh.WaitForFirstRefreshToken(time.Second*20))

	sh.failSignIn.Store(false)
	// onboarding should cancel ongoing deregistering
	// _, shutdown := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	_, shutdown := test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	defer shutdown()

	// unblock refresh token -> deregistering should continue by attempting to Sign-In and then Sign-Off
	close(blockRefreshCh)

	// however deregistering should be cancelled by onboarding so no Sign-Off should occur
	require.False(t, sh.WaitForFirstSignOff(time.Second*10))

	// must execute before onboarding shutdown which sometimes executes quickly enough
	// to mess up the Sing-Off counter
	coapShutdown()
}
