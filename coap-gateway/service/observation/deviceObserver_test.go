package observation_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/observation"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isPb "github.com/plgd-dev/hub/v2/identity-store/pb"
	isTest "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raPb "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	pbRD "github.com/plgd-dev/hub/v2/resource-directory/pb"
	"github.com/plgd-dev/hub/v2/test"
	coapgwTestService "github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/test/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/device/ocf"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	virtualdevice "github.com/plgd-dev/hub/v2/test/virtual-device"
	"github.com/plgd-dev/kit/v2/strings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type RDClient struct {
	pb.GrpcGatewayClient
	pbRD.ResourceDirectoryClient
}

type deviceObserverFactory struct {
	rdClient RDClient
	raClient raPb.ResourceAggregateClient
	deviceID string
}

func (f deviceObserverFactory) makeDeviceObserver(ctx context.Context, coapConn *coapTcpClient.Conn, onObserveResource observation.OnObserveResource,
	onGetResourceContent observation.OnGetResourceContent, updateTwinSynchronization observation.UpdateTwinSynchronization,
	opts ...observation.Option,
) (*observation.DeviceObserver, error) {
	return observation.NewDeviceObserver(ctx, f.deviceID, coapConn, f.rdClient, f.raClient,
		observation.ResourcesObserverCallbacks{
			OnObserveResource:         onObserveResource,
			OnGetResourceContent:      onGetResourceContent,
			UpdateTwinSynchronization: updateTwinSynchronization,
		}, opts...)
}

type observerHandler struct {
	deviceObserverFactory deviceObserverFactory
	ctx                   context.Context
	t                     *testing.T
	coapConn              *coapTcpClient.Conn
	service               *coapgwTestService.Service

	retrievedResourceChan chan *commands.ResourceId
	observedResourceChan  chan *commands.ResourceId
	coapgwTest.DefaultObserverHandler
	done atomic.Bool

	requireBatchObserveEnabled bool
	private                    struct { // guarded by deviceObserverLock
		deviceObserverLock sync.Mutex
		deviceObserver     *future.Future
	}
}

const (
	tokenLifetime time.Duration = time.Hour
)

func (h *observerHandler) getDeviceObserver(ctx context.Context) *observation.DeviceObserver {
	var f *future.Future
	h.private.deviceObserverLock.Lock()
	f = h.private.deviceObserver
	h.private.deviceObserverLock.Unlock()
	v, err := f.Get(ctx)
	require.NoError(h.t, err)
	return v.(*observation.DeviceObserver)
}

func (h *observerHandler) replaceDeviceObserver(deviceObserverFuture *future.Future) *future.Future {
	var prevObs *future.Future
	h.private.deviceObserverLock.Lock()
	defer h.private.deviceObserverLock.Unlock()
	prevObs = h.private.deviceObserver
	h.private.deviceObserver = deviceObserverFuture
	return prevObs
}

func (h *observerHandler) SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error) {
	resp, err := h.DefaultObserverHandler.SignIn(req)
	require.NoError(h.t, err)

	newDeviceObserver, setDeviceObserver := future.New()
	prevDeviceObserver := h.replaceDeviceObserver(newDeviceObserver)

	err = h.service.Submit(func() {
		if prevDeviceObserver != nil {
			v, errD := prevDeviceObserver.Get(h.ctx)
			require.NoError(h.t, errD)
			obs := v.(*observation.DeviceObserver)
			obs.Clean(h.ctx)
		}
		deviceObserver, errD := h.deviceObserverFactory.makeDeviceObserver(h.ctx, h.coapConn, h.OnObserveResource, h.OnGetResourceContent, h.UpdateTwinSynchronizationStatus, observation.WithRequireBatchObserveEnabled(h.requireBatchObserveEnabled))
		require.NoError(h.t, errD)
		setDeviceObserver(deviceObserver, nil)
	})
	require.NoError(h.t, err)
	return resp, nil
}

func (h *observerHandler) SignOff() error {
	err := h.DefaultObserverHandler.SignOff()
	require.NoError(h.t, err)
	h.done.Store(true)
	return nil
}

func (h *observerHandler) PublishResources(req coapgwTestService.PublishRequest) error {
	err := h.DefaultObserverHandler.PublishResources(req)
	require.NoError(h.t, err)

	var validUntil time.Time
	if req.TimeToLive > 0 {
		validUntil = time.Now().Add(time.Second * time.Duration(req.TimeToLive))
	}
	resources := commands.SchemaResourceLinksToResources(req.Links, validUntil)

	err = h.service.Submit(func() {
		obs := h.getDeviceObserver(h.ctx)
		errPub := obs.AddPublishedResources(h.ctx, resources)
		require.NoError(h.t, errPub)
	})
	require.NoError(h.t, err)
	return nil
}

func (h *observerHandler) OnObserveResource(ctx context.Context, deviceID, resourceHref string, resourceTypes []string, _ bool, notification *pool.Message) error {
	err := h.DefaultObserverHandler.OnObserveResource(ctx, deviceID, resourceHref, resourceTypes, notification)
	require.NoError(h.t, err)
	if !h.done.Load() {
		h.observedResourceChan <- commands.NewResourceID(deviceID, resourceHref)
	}
	return nil
}

func (h *observerHandler) OnGetResourceContent(ctx context.Context, deviceID, resourceHref string, resourceTypes []string, notification *pool.Message) error {
	err := h.DefaultObserverHandler.OnGetResourceContent(ctx, deviceID, resourceHref, resourceTypes, notification)
	require.NoError(h.t, err)
	if !h.done.Load() {
		h.retrievedResourceChan <- commands.NewResourceID(deviceID, resourceHref)
	}
	return nil
}

func (h *observerHandler) UpdateTwinSynchronizationStatus(ctx context.Context, deviceID string, state commands.TwinSynchronization_State, t time.Time) error {
	err := h.DefaultObserverHandler.UpdateTwinSynchronization(ctx, deviceID, state, t)
	require.NoError(h.t, err)
	return nil
}

func TestDeviceObserverRegisterForPublishedResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	discoveryObservable := test.DeviceIsBatchObservable(ctx, t, deviceID)
	if discoveryObservable {
		t.Logf("skipping test for device with %v observable", resources.ResourceURI)
		return
	}
	validateData := func(ctx context.Context, oh *observerHandler) {
		obs := oh.getDeviceObserver(ctx)
		require.Equal(t, observation.ObservationType_PerResource, obs.GetObservationType())
		res, err := obs.GetResources()
		require.NoError(t, err)
		pbTest.CmpResourceIds(t, test.ResourceLinksToResourceIds(deviceID, test.GetAllBackendResourceLinks()), res)
	}

	expectedObserved := strings.MakeSet()
	for _, resID := range test.ResourceLinksToResourceIds(deviceID, test.FilterResourceLink(func(rl schema.ResourceLink) bool {
		return rl.Policy.BitMask.Has(schema.Observable)
	}, test.GetAllBackendResourceLinks())) {
		expectedObserved.Add(resID.ToString())
	}
	expectedRetrieved := strings.MakeSet()
	for _, resID := range test.ResourceLinksToResourceIds(deviceID, test.FilterResourceLink(func(rl schema.ResourceLink) bool {
		return !rl.Policy.BitMask.Has(schema.Observable)
	}, test.GetAllBackendResourceLinks())) {
		expectedRetrieved.Add(resID.ToString())
	}
	runTestDeviceObserverRegister(ctx, t, deviceID, expectedObserved, expectedRetrieved, validateData, nil, nil, false)
}

func TestDeviceObserverRegisterForPublishedResourcesWithAlreadyPublishedResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	discoveryObservable := test.DeviceIsBatchObservable(ctx, t, deviceID)
	if discoveryObservable {
		t.Logf("skipping test for device with %v observable", resources.ResourceURI)
		return
	}
	validateData := func(ctx context.Context, oh *observerHandler) {
		obs := oh.getDeviceObserver(ctx)
		require.Equal(t, observation.ObservationType_PerResource, obs.GetObservationType())
		res, err := obs.GetResources()
		require.NoError(t, err)
		pbTest.CmpResourceIds(t, test.ResourceLinksToResourceIds(deviceID, test.GetAllBackendResourceLinks()), res)
	}

	expectedObserved := strings.MakeSet()
	for _, resID := range test.ResourceLinksToResourceIds(deviceID, test.FilterResourceLink(func(rl schema.ResourceLink) bool {
		return rl.Policy.BitMask.Has(schema.Observable)
	}, test.GetAllBackendResourceLinks())) {
		expectedObserved.Add(resID.ToString())
	}
	expectedRetrieved := strings.MakeSet()
	for _, resID := range test.ResourceLinksToResourceIds(deviceID, test.FilterResourceLink(func(rl schema.ResourceLink) bool {
		return !rl.Policy.BitMask.Has(schema.Observable)
	}, test.GetAllBackendResourceLinks())) {
		expectedRetrieved.Add(resID.ToString())
	}
	runTestDeviceObserverRegister(ctx, t, deviceID, expectedObserved, expectedRetrieved, validateData, testPreregisterVirtualDevice, testValidateResourceLinks, false)
}

func TestDeviceObserverRegisterForDiscoveryResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceNameWithOicResObservable)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	discoveryObservable := test.DeviceIsBatchObservable(ctx, t, deviceID)
	if !discoveryObservable {
		t.Logf("skipping test for device with %v non-observable", resources.ResourceURI)
		return
	}
	validateData := func(ctx context.Context, oh *observerHandler) {
		obs := oh.getDeviceObserver(ctx)
		require.Equal(t, observation.ObservationType_PerDevice, obs.GetObservationType())
		res, err := obs.GetResources()
		require.NoError(t, err)
		pbTest.CmpResourceIds(t, []*commands.ResourceId{{DeviceId: deviceID, Href: resources.ResourceURI}}, res)
	}

	expectedObserved := strings.MakeSet(commands.NewResourceID(deviceID, resources.ResourceURI).ToString())
	runTestDeviceObserverRegister(ctx, t, deviceID, expectedObserved, nil, validateData, nil, nil, true)
}

func testPreregisterVirtualDevice(ctx context.Context, t *testing.T, deviceID string, grpcClient pb.GrpcGatewayClient, raClient raPb.ResourceAggregateClient) {
	isConn, err := grpc.NewClient(config.IDENTITY_STORE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = isConn.Close()
	}()
	isClient := isPb.NewIdentityStoreClient(isConn)
	client, err := grpcClient.SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func() {
		errC := client.CloseSend()
		require.NoError(t, errC)
	}()

	err = client.Send(&pb.SubscribeToEvents{
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
			},
		},
	})
	require.NoError(t, err)

	numResources := 10
	ev, err := client.Recv()
	require.NoError(t, err)
	require.NotEmpty(t, ev.GetOperationProcessed())
	require.Equal(t, pb.Event_OperationProcessed_ErrorStatus_OK, ev.GetOperationProcessed().GetErrorStatus().GetCode())
	virtualdevice.CreateDevice(ctx, t, "name-"+deviceID, deviceID, numResources, false, test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), isClient, raClient)
	resources := virtualdevice.CreateDeviceResourceLinks(deviceID, numResources, false)
	links := make([]schema.ResourceLink, 0, len(resources))
	for _, r := range resources {
		links = append(links, r.ToSchema())
	}
	test.WaitForDevice(t, client, ocf.NewDevice(deviceID, test.TestDeviceName), ev.GetSubscriptionId(), ev.GetCorrelationId(), links)
}

func testValidateResourceLinks(ctx context.Context, t *testing.T, deviceID string, grpcClient pb.GrpcGatewayClient, _ raPb.ResourceAggregateClient) {
	client, err := grpcClient.GetResourceLinks(ctx, &pb.GetResourceLinksRequest{
		DeviceIdFilter: []string{deviceID},
	})
	require.NoError(t, err)
	expected := []*commands.Resource{
		{
			Href:          device.ResourceURI,
			DeviceId:      deviceID,
			ResourceTypes: []string{device.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_BASELINE},
			Policy: &commands.Policy{
				BitFlags: commands.ToPolicyBitFlags(schema.Observable | schema.Discoverable),
			},
		},
		{
			Href:          platform.ResourceURI,
			DeviceId:      deviceID,
			ResourceTypes: []string{platform.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_BASELINE},
			Policy: &commands.Policy{
				BitFlags: commands.ToPolicyBitFlags(schema.Observable | schema.Discoverable),
			},
		},
	}
	var got []*commands.Resource
	for {
		v, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		got = v.GetResources()
	}
	expected = test.CleanUpResourcesArray(expected)
	got = test.CleanUpResourcesArray(got)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func TestDeviceObserverRegisterForDiscoveryResourceWithAlreadyPublishedResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceNameWithOicResObservable)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	discoveryObservable := test.DeviceIsBatchObservable(ctx, t, deviceID)
	if !discoveryObservable {
		t.Logf("skipping test for device with %v non-observable", resources.ResourceURI)
		return
	}
	validateData := func(ctx context.Context, oh *observerHandler) {
		obs := oh.getDeviceObserver(ctx)
		require.Equal(t, observation.ObservationType_PerDevice, obs.GetObservationType())
		res, err := obs.GetResources()
		require.NoError(t, err)
		pbTest.CmpResourceIds(t, []*commands.ResourceId{{DeviceId: deviceID, Href: resources.ResourceURI}}, res)
	}

	expectedObserved := strings.MakeSet(commands.NewResourceID(deviceID, resources.ResourceURI).ToString())
	runTestDeviceObserverRegister(ctx, t, deviceID, expectedObserved, nil, validateData, testPreregisterVirtualDevice, testValidateResourceLinks, true)
}

type (
	verifyHandlerFn = func(context.Context, *observerHandler)
	actioneHubFn    = func(ctx context.Context, t *testing.T, deviceID string, grpcClient pb.GrpcGatewayClient, raClient raPb.ResourceAggregateClient)
)

func runTestDeviceObserverRegister(ctx context.Context, t *testing.T, deviceID string, expectedObserved, expectedRetrieved strings.Set, verifyHandler verifyHandlerFn, prepareHub, postHub actioneHubFn, requireBatchObserveEnabled bool) {
	// TODO: add test with expectedRetrieved
	const services = service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesOAuth | service.SetUpServicesId | service.SetUpServicesResourceDirectory |
		service.SetUpServicesGrpcGateway | service.SetUpServicesResourceAggregate

	isConfig := isTest.MakeConfig(t)
	isConfig.APIs.GRPC.TLS.ClientCertificateRequired = false

	raConfig := raTest.MakeConfig(t)
	raConfig.APIs.GRPC.TLS.ClientCertificateRequired = false

	tearDown := service.SetUpServices(ctx, t, services, service.WithISConfig(isConfig), service.WithRAConfig(raConfig))
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()

	rdConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.GRPC_GW_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = rdConn.Close()
	}()
	grpcC := pb.NewGrpcGatewayClient(rdConn.GRPC())
	rdC := pbRD.NewResourceDirectoryClient(rdConn.GRPC())
	rdClient := RDClient{
		GrpcGatewayClient:       grpcC,
		ResourceDirectoryClient: rdC,
	}

	raConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	raClient := raPb.NewResourceAggregateClient(raConn.GRPC())

	if prepareHub != nil {
		prepareHub(ctx, t, deviceID, rdClient, raClient)
	}

	retrieveChan := make(chan *commands.ResourceId, 8)
	observeChan := make(chan *commands.ResourceId, 8)
	makeHandler := func(service *coapgwTestService.Service, opts ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		cfg := coapgwTestService.ServiceHandlerConfig{}
		for _, o := range opts {
			o.Apply(&cfg)
		}
		h := &observerHandler{
			DefaultObserverHandler: coapgwTest.MakeDefaultObserverHandler(int64(tokenLifetime.Seconds())),
			t:                      t,
			ctx:                    ctx,
			coapConn:               cfg.GetCoapConnection(),
			deviceObserverFactory: deviceObserverFactory{
				deviceID: deviceID,
				rdClient: rdClient,
				raClient: raClient,
			},
			service:                    service,
			retrievedResourceChan:      retrieveChan,
			observedResourceChan:       observeChan,
			requireBatchObserveEnabled: requireBatchObserveEnabled,
		}
		return h
	}
	validateHandler := func(h coapgwTestService.ServiceHandler) {
		handler := h.(*observerHandler)
		verifyHandler(ctx, handler)
	}

	coapShutdown := coapgwTest.SetUp(t, makeHandler, validateHandler)
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

	done := false
	isDone := func() bool {
		return len(expectedRetrieved) == 0 && len(expectedObserved) == 0
	}
	// give time to wait for data
	ctxWait, waitCancel := context.WithTimeout(context.Background(), time.Second*10)
	closeWaitChans := func() {
		close(retrieveChan)
		close(observeChan)
	}
	defer waitCancel()
	for !done {
		select {
		case res := <-retrieveChan:
			if expectedRetrieved == nil || !expectedRetrieved.HasOneOf(res.ToString()) {
				assert.Failf(t, "unexpected retrieved resource", "resource (%v)", res.ToString())
				closeWaitChans()
				done = true
				break
			}
			delete(expectedRetrieved, res.ToString())
			done = isDone()
		case res := <-observeChan:
			if expectedObserved == nil || !expectedObserved.HasOneOf(res.ToString()) {
				assert.Failf(t, "unexpected observed resource", "resource (%v)", res.ToString())
				closeWaitChans()
				done = true
				break
			}
			delete(expectedObserved, res.ToString())
			done = isDone()
		case <-ctxWait.Done():
			t.Log("waiting timeouted")
			done = true
		}
	}
	require.Empty(t, expectedObserved)
	require.Empty(t, expectedRetrieved)

	if postHub != nil {
		postHub(ctx, t, deviceID, rdClient, raClient)
	}
}
