package service_test

import (
	"context"
	"crypto/tls"
	"errors"
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
	d := test.MustFindTestDevice()

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	ch := iotService.NewCoapHandlerWithCounter(0)
	makeHandler := func(*coapgwTestService.Service, ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return ch
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*iotService.CoapHandlerWithCounter)
		log.Debugf("%+v", h.CallCounter.Data)
		signInCount, ok := h.CallCounter.Data[iotService.SignInKey]
		require.True(t, ok)
		require.Positive(t, signInCount)
		publishCount, ok := h.CallCounter.Data[iotService.PublishKey]
		require.True(t, ok)
		require.Equal(t, 1, publishCount)
		singOffCount, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.True(t, ok)
		require.Positive(t, singOffCount)
	}

	coapShutdown := coapgwTest.SetUp(t, makeHandler, validateHandler)
	defer coapShutdown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	// TODO: copy services initialization from the real coap-gw to the mock coap-gw,
	// for now we must force TCP when mock coap-gw is used
	// shutdown := test.OnboardDevice(ctx, t, c, d, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	shutdown := test.OnboardDevice(ctx, t, c, d, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	require.True(t, ch.WaitForFirstSignIn(time.Second*20))
	t.Cleanup(func() {
		shutdown()
	})

	test.OffboardDevice(ctx, t, d)
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
		return coapgwService.CoapSignInResp{}, errors.New("sign in disabled")
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
		return errors.New("sign off disabled")
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
		return coapgwService.CoapRefreshTokenResp{}, errors.New("refresh token disabled")
	}
	b := sh.blockRefreshChannel()
	if b != nil {
		<-b
	}
	return resp, err
}

// not signed in, with permanent and short access-token -> deregister by sending DELETE request with access token
func TestOffboardWithoutSignIn(t *testing.T) {
	d := test.MustFindTestDevice()

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	sh := NewSwitchableHandler(-1)
	sh.failSignIn.Store(true)
	makeHandler := func(*coapgwTestService.Service, ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return sh
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*switchableHandler)
		h.CallCounter.Lock.Lock()
		defer h.CallCounter.Lock.Unlock()
		log.Debugf("%+v", h.CallCounter.Data)
		signInCount, ok := h.CallCounter.Data[iotService.SignInKey]
		require.True(t, ok)
		// depending on the timing of the test, the sign-in may be called once or twice
		require.True(t, signInCount >= 1 && signInCount <= 2)
		_, ok = h.CallCounter.Data[iotService.RefreshTokenKey]
		require.False(t, ok)
		signOffCount, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.True(t, ok)
		require.Equal(t, 1, signOffCount)
	}

	coapShutdown := coapgwTest.SetUp(t, makeHandler, validateHandler)
	defer coapShutdown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	// shutdown := test.OnboardDevSim(ctx, t, c, d, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	shutdown := test.OnboardDevice(ctx, t, c, d, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	t.Cleanup(func() {
		shutdown()
	})
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))
	// first retry after failure is after 2 seconds, so hopefully it doesn't trigger, if this test
	// behaves flakily then we will have to update simulator to have a configurable retry
	sh.failSignIn.Store(false)
	time.Sleep(d.GetRetryInterval(1) + time.Second)
	test.OffboardDevice(ctx, t, d)
	require.True(t, sh.WaitForFirstSignOff(time.Second*20))
}

// not signed in, with permanent but long access-token -> try to sign in and then deregister without access token
// OCF device specific behavior
func TestOffboardWithSignIn(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	sh := NewSwitchableHandler(-1)
	sh.failSignIn.Store(true)
	sh.SetAccessToken(strings.Repeat("this-access-token-is-so-long-that-its-size-is-longer-than-the-allowed-request-header-size", 5))
	makeHandler := func(*coapgwTestService.Service, ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
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

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	// deviceID, shutdown := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	deviceID, shutdown := test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	t.Cleanup(func() {
		shutdown()
	})
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))

	// first retry after failure is after 2 seconds, so hopefully it doesn't trigger, if this test
	// behaves flakily then we will have to update simulator to have a configurable retry
	sh.failSignIn.Store(false)
	// wait for sign-in to be called again
	time.Sleep(time.Second * 3)
	test.OffBoardDevSim(ctx, t, deviceID)
	require.True(t, sh.WaitForFirstSignOff(time.Second*20))
}

// not signed up, with refresh token -> try to login and then deregister without access token
// OCF device specific behavior
func TestOffboardWithSignInByRefreshToken(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	sh := NewSwitchableHandler(20)
	sh.failSignIn.Store(true)
	sh.SetAccessToken(strings.Repeat("this-access-token-is-so-long-that-its-size-is-longer-than-the-allowed-request-header-size", 5))
	makeHandler := func(*coapgwTestService.Service, ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return sh
	}

	validateHandler := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*switchableHandler)
		h.CallCounter.Lock.Lock()
		defer h.CallCounter.Lock.Unlock()
		log.Debugf("%+v", h.CallCounter.Data)
		signInCount, ok := h.CallCounter.Data[iotService.SignInKey]
		require.True(t, ok)
		require.Greater(t, signInCount, 1)
		refreshCount, ok := h.CallCounter.Data[iotService.RefreshTokenKey]
		require.True(t, ok)
		require.Positive(t, refreshCount)
		signOffCount, ok := h.CallCounter.Data[iotService.SignOffKey]
		require.True(t, ok)
		require.Equal(t, 1, signOffCount)
	}

	coapShutdown := coapgwTest.SetUp(t, makeHandler, func(coapgwTestService.ServiceHandler) {})

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	// register device first time
	// deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	deviceID, shutdown := test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	t.Cleanup(func() {
		shutdown()
	})
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))

	// first retry after failure is after 2 seconds, so hopefully it doesn't trigger, if this test
	// behaves flakily then we will have to update simulator to have a configurable retry

	coapShutdown()

	// force to sign in with refresh token
	sh.failSignIn.Store(false)
	sh.ResetRefreshToken()
	sh.ResetSignIn()
	sh.ResetSignOff()
	coapShutdown = coapgwTest.SetUp(t, makeHandler, validateHandler)
	defer coapShutdown()

	// wait for refresh token and sign-in to be called again
	sh.WaitForSignIn(time.Second * 20)

	test.OffBoardDevSim(ctx, t, deviceID)
	require.True(t, sh.WaitForFirstRefreshToken(time.Second*20))
	require.True(t, sh.WaitForFirstSignOff(time.Second*20))
}

// Multiple offboard attempts should be ignored
func TestOffboardWithRepeat(t *testing.T) {
	d := test.MustFindTestDevice()

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	sh := NewSwitchableHandler(-1)
	sh.blockSignOff()
	makeHandler := func(*coapgwTestService.Service, ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
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

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	// deviceID, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	shutdown := test.OnboardDevice(ctx, t, c, d, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	t.Cleanup(func() {
		shutdown()
	})
	require.True(t, sh.WaitForFirstSignIn(time.Second*20))

	time.Sleep(time.Second)

	test.OffboardDevice(ctx, t, d)
	test.OffboardDevice(ctx, t, d)
	test.OffboardDevice(ctx, t, d)

	require.True(t, sh.WaitForFirstSignOff(time.Second*20))
	// first SignOff should timeout after 10 secs, we wait 10 additional seconds for the other
	// calls to maybe happen
	time.Sleep(time.Second * 10)
}

// Onboarding should interrupt an ongoing offboarding
// OCF device specific behavior
func TestOffboardInterrupt(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesGrpcGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	// force the use of refresh token (non-permanent, long access-token and sign-up will fail)
	sh := NewSwitchableHandler(int64(time.Minute.Seconds()))
	sh.failSignIn.Store(true)
	sh.SetAccessToken(strings.Repeat("this-access-token-is-so-long-that-its-size-is-longer-than-the-allowed-request-header-size", 5))
	// off-boarding will try login by refresh token, but block the sending of response
	blockRefreshCh := sh.blockRefresh()

	makeHandler := func(*coapgwTestService.Service, ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
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

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
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

	time.Sleep(time.Second * 3)

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
