package client_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	idClient "github.com/plgd-dev/hub/v2/identity-store/client"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	idService "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	clientCertManager "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

func TestOwnerCacheSubscribe(t *testing.T) {
	ctx := context.Background()
	service.ClearDB(ctx, t)

	devices := []string{test.GenerateDeviceIDbyIdx(1), test.GenerateDeviceIDbyIdx(2), test.GenerateDeviceIDbyIdx(3)}
	cfg := idService.MakeConfig(t)
	cfg.APIs.GRPC.Addr = "localhost:1234"
	tearDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth|service.SetUpServicesId, service.WithISConfig(cfg))
	defer tearDown()

	token := oauthService.GetDefaultAccessToken(t)

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	conn, err := client.New(client.Config{
		Addr: cfg.APIs.GRPC.Addr,
		TLS: clientCertManager.Config{
			CAPool:   cfg.APIs.GRPC.TLS.CAPool,
			CertFile: cfg.APIs.GRPC.TLS.CertFile,
			KeyFile:  cfg.APIs.GRPC.TLS.KeyFile,
		},
	}, fileWatcher, log.Get(), noop.NewTracerProvider(), grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(func(context.Context) (*oauth2.Token, error) {
		return &oauth2.Token{
			AccessToken:  token,
			TokenType:    "Bearer",
			RefreshToken: "",
			Expiry:       time.Time{},
		}, nil
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	c := pb.NewIdentityStoreClient(conn.GRPC())

	ctx = kitNetGrpc.CtxWithToken(ctx, token)
	ctx = kitNetGrpc.CtxWithIncomingToken(ctx, token)
	owner, err := kitNetGrpc.ParseOwnerFromJwtToken("sub", token)
	require.NoError(t, err)

	naClient, subscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher, log.Get())
	require.NoError(t, err)
	defer func() {
		subscriber.Close()
		naClient.Close()
	}()

	cacheExpiration := time.Second
	cache := idClient.NewOwnerCache("sub", cacheExpiration, subscriber.Conn(), c, func(err error) { fmt.Printf("%v\n", err) })
	defer cache.Close()

	// test 2 subscription to same owner
	var lock sync.Mutex
	var sub1registeredDevices []string
	var sub2registeredDevices []string

	var sub1unregisteredDevices []string
	var sub2unregisteredDevices []string

	sub1Close, err := cache.Subscribe(owner, func(e *events.Event) {
		if len(e.GetDevicesRegistered().GetDeviceIds()) > 0 {
			lock.Lock()
			defer lock.Unlock()
			sub1registeredDevices = append(sub1registeredDevices, e.GetDevicesRegistered().GetDeviceIds()...)
		}
		if len(e.GetDevicesUnregistered().GetDeviceIds()) > 0 {
			lock.Lock()
			defer lock.Unlock()
			sub1unregisteredDevices = append(sub1unregisteredDevices, e.GetDevicesUnregistered().GetDeviceIds()...)
		}
	})
	require.NoError(t, err)
	sub2Close, err := cache.Subscribe(owner, func(e *events.Event) {
		if len(e.GetDevicesRegistered().GetDeviceIds()) > 0 {
			lock.Lock()
			defer lock.Unlock()
			sub2registeredDevices = append(sub2registeredDevices, e.GetDevicesRegistered().GetDeviceIds()...)
		}
		if len(e.GetDevicesUnregistered().GetDeviceIds()) > 0 {
			lock.Lock()
			defer lock.Unlock()
			sub2unregisteredDevices = append(sub2unregisteredDevices, e.GetDevicesUnregistered().GetDeviceIds()...)
		}
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)

	for _, d := range devices[:2] {
		_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{DeviceId: d})
		require.NoError(t, err)
	}
	time.Sleep(time.Millisecond * 100)

	ok, err := cache.OwnsDevices(ctx, devices)
	require.NoError(t, err)
	assert.False(t, ok)

	lock.Lock()
	sort.Strings(sub1registeredDevices)
	sort.Strings(sub2registeredDevices)
	assert.Equal(t, devices[:2], sub1registeredDevices)
	assert.Equal(t, devices[:2], sub2registeredDevices)
	lock.Unlock()

	cacheDevices, err := cache.GetDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], cacheDevices)
	// expire data
	time.Sleep(cacheExpiration * 2)

	// check update
	added, removed, err := cache.Update(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], added)
	assert.Empty(t, removed)

	cacheDevices, err = cache.GetDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], cacheDevices)

	ownedDevices, err := cache.GetSelectedDevices(ctx, devices)
	require.NoError(t, err)
	assert.Equal(t, cacheDevices, ownedDevices)

	ok, err = cache.OwnsDevices(ctx, ownedDevices)
	require.NoError(t, err)
	assert.True(t, ok)

	deleted, err := c.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIds: devices[:1],
	})
	require.NoError(t, err)
	assert.Equal(t, devices[:1], deleted.GetDeviceIds())

	// check update - after expiration
	time.Sleep(time.Millisecond * 100) // wait for synchronize the cache, so update doesn't change the cache
	added, removed, err = cache.Update(ctx)
	require.NoError(t, err)
	assert.Empty(t, added)
	assert.Empty(t, removed)
	cacheDevices, err = cache.GetDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[1:2], cacheDevices)

	lock.Lock()
	sort.Strings(sub1unregisteredDevices)
	assert.Equal(t, devices[0:1], sub1unregisteredDevices)
	sort.Strings(sub2unregisteredDevices)
	assert.Equal(t, devices[0:1], sub2unregisteredDevices)
	lock.Unlock()

	ok, err = cache.OwnsDevice(ctx, devices[0])
	require.NoError(t, err)
	assert.False(t, ok)

	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{DeviceId: devices[0]})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)
	cacheDevices, err = cache.GetDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], cacheDevices)

	ok, err = cache.OwnsDevice(ctx, devices[0])
	require.NoError(t, err)
	assert.True(t, ok)

	// cache without subscription to events
	sub1Close()
	sub2Close()
	time.Sleep(cacheExpiration * 2)
	added, removed, err = cache.Update(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], added)
	assert.Empty(t, removed)

	// update or refresh by GetDevices after expiration
	time.Sleep(cacheExpiration * 2)
	cacheDevices, err = cache.GetDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], cacheDevices)

	// updated by GetDevices, Update does nothing
	added, removed, err = cache.Update(ctx)
	require.NoError(t, err)
	assert.Empty(t, added)
	assert.Empty(t, removed)
	cacheDevices, err = cache.GetDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], cacheDevices)

	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{DeviceId: devices[2]})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)
	cacheDevices, err = cache.GetDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:3], cacheDevices)

	_, err = c.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIds: devices[1:],
	})
	require.NoError(t, err)
	time.Sleep(cacheExpiration * 2)
	ownedDevices, err = cache.GetSelectedDevices(ctx, devices[1:])
	require.NoError(t, err)
	assert.Empty(t, ownedDevices)
}
