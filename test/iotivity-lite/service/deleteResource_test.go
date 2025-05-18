package service_test

import (
	"context"
	"crypto/tls"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	coapgwTestService "github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/test/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/test/config"
	iotService "github.com/plgd-dev/hub/v2/test/iotivity-lite/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/sdk"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type (
	publishDoneCheck   = func(map[string]int64) bool
	unpublishDoneCheck = func(map[int64]struct{}) bool
)

type batchDeleteHandler struct {
	*iotService.CoapHandlerWithCounter

	resources struct {
		lock        sync.Mutex
		published   map[string]int64
		unpublished map[int64]struct{}
	}

	publishDone       publishDoneCheck
	publishDoneChan   chan struct{}
	unpublishDone     unpublishDoneCheck
	unpublishDoneChan chan struct{}
}

func newBatchDeleteHandler(publishDone publishDoneCheck, unpublishDone unpublishDoneCheck) *batchDeleteHandler {
	bh := batchDeleteHandler{
		CoapHandlerWithCounter: iotService.NewCoapHandlerWithCounter(0),

		publishDone:       publishDone,
		publishDoneChan:   make(chan struct{}),
		unpublishDone:     unpublishDone,
		unpublishDoneChan: make(chan struct{}),
	}
	bh.resources.published = make(map[string]int64)
	bh.resources.unpublished = make(map[int64]struct{})
	return &bh
}

func (h *batchDeleteHandler) PublishResources(req coapgwTestService.PublishRequest) error {
	if err := h.DefaultObserverHandler.PublishResources(req); err != nil {
		return err
	}

	for _, link := range req.Links {
		// we are interested only in the newly created "/switches/${i}" resources
		if !strings.Contains(link.Href, test.TestResourceSwitchesHref+"/") {
			continue
		}
		log.Debugf("published resource: %v(ins=%v)", link.Href, link.InstanceID)
		h.resources.lock.Lock()
		h.resources.published[link.Href] = link.InstanceID
		h.resources.lock.Unlock()
	}

	h.resources.lock.Lock()
	published := maps.Clone(h.resources.published)
	h.resources.lock.Unlock()

	if h.publishDone(published) {
		close(h.publishDoneChan)
	}
	return nil
}

func waitOnChannel(ch <-chan struct{}, timeout time.Duration) bool {
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (h *batchDeleteHandler) WaitForPublish(timeout time.Duration) bool {
	return waitOnChannel(h.publishDoneChan, timeout)
}

func (h *batchDeleteHandler) UnpublishResources(req coapgwTestService.UnpublishRequest) error {
	if err := h.DefaultObserverHandler.UnpublishResources(req); err != nil {
		return err
	}

	for _, instanceID := range req.InstanceIDs {
		log.Debugf("unpublished resource: %v", instanceID)
		h.resources.lock.Lock()
		h.resources.unpublished[instanceID] = struct{}{}
		h.resources.lock.Unlock()
	}

	h.resources.lock.Lock()
	unpublished := maps.Clone(h.resources.unpublished)
	h.resources.lock.Unlock()
	if h.unpublishDone(unpublished) {
		close(h.unpublishDoneChan)
	}
	return nil
}

func (h *batchDeleteHandler) WaitForUnpublish(timeout time.Duration) bool {
	return waitOnChannel(h.unpublishDoneChan, timeout)
}

func TestBatchDeleteResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	const services = service.SetUpServicesOAuth
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	const numSwitches = 16
	bh := newBatchDeleteHandler(func(published map[string]int64) bool {
		return len(published) == numSwitches
	}, func(unpublished map[int64]struct{}) bool {
		return len(unpublished) == numSwitches
	})
	getHandler := func(*coapgwTestService.Service, ...coapgwTestService.Option) coapgwTestService.ServiceHandler {
		return bh
	}

	shutDownChan := make(chan struct{})
	validatePublishOnShutdown := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*batchDeleteHandler)
		h.resources.lock.Lock()
		defer h.resources.lock.Unlock()
		require.Len(t, h.resources.published, numSwitches)
		if count, ok := h.CallCounter.Data[iotService.UnpublishKey]; !ok || count == 0 {
			close(shutDownChan)
			return
		}
	}
	coapShutdown := coapgwTest.SetUp(t, getHandler, validatePublishOnShutdown)

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
	// _, _ = test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	_, shutdown := test.OnboardDevSim(ctx, t, c, deviceID, string(schema.TCPSecureScheme)+"://"+config.COAP_GW_HOST, nil)
	require.True(t, bh.WaitForFirstSignIn(time.Second*20))
	t.Cleanup(func() {
		shutdown()
	})

	devClient, err := sdk.NewClient()
	require.NoError(t, err)
	defer func() {
		_ = devClient.Close(ctx)
	}()

	// use SDK to create multiple switch resources
	for range numSwitches {
		err = devClient.CreateResource(ctx, deviceID, test.TestResourceSwitchesHref, test.MakeSwitchResourceDefaultData(), nil)
		require.NoError(t, err)
	}
	// wait for publish of all resources
	require.True(t, bh.WaitForPublish(time.Second*20))
	// wait for response to last message to be processed
	time.Sleep(500 * time.Millisecond)

	// disconnect mock coap-gw
	coapShutdown()
	require.True(t, waitOnChannel(shutDownChan, time.Second*20))

	// use SDK to batch delete switch resources
	var resp interface{}
	err = devClient.DeleteResource(ctx, deviceID, test.TestResourceSwitchesHref, &resp, client.WithInterface(interfaces.OC_IF_B))
	require.NoError(t, err)

	validateOnShutdown := func(handler coapgwTestService.ServiceHandler) {
		h := handler.(*batchDeleteHandler)
		h.resources.lock.Lock()
		defer h.resources.lock.Unlock()
		require.Len(t, h.resources.published, numSwitches)
		require.Len(t, h.resources.unpublished, numSwitches)
		for _, instanceID := range h.resources.published {
			delete(h.resources.unpublished, instanceID)
		}
		require.Empty(t, h.resources.unpublished)
	}
	coapShutdown = coapgwTest.SetUp(t, getHandler, validateOnShutdown)
	defer coapShutdown()

	require.True(t, bh.WaitForUnpublish(time.Second*20))

	test.OffBoardDevSim(ctx, t, deviceID)
	require.True(t, bh.WaitForFirstSignOff(time.Second*20))
}
