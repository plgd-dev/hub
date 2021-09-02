package client_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	authClient "github.com/plgd-dev/cloud/authorization/client"
	"github.com/plgd-dev/cloud/authorization/events"
	"github.com/plgd-dev/cloud/authorization/pb"
	authService "github.com/plgd-dev/cloud/authorization/test"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	clientCertManager "github.com/plgd-dev/cloud/pkg/security/certManager/client"
	natsTest "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

func TestOwnerCache_Subscribe(t *testing.T) {
	test.ClearDB(context.Background(), t)

	devices := []string{"device1", "device2", "device3"}
	cfg := authService.MakeConfig(t)
	cfg.APIs.GRPC.Addr = "localhost:1234"

	oauthShutdown := oauthService.SetUp(t)
	defer oauthShutdown()

	shutdown := authService.New(t, cfg)
	defer shutdown()

	token := oauthService.GetServiceToken(t)

	conn, err := client.New(client.Config{
		Addr: cfg.APIs.GRPC.Addr,
		TLS: clientCertManager.Config{
			CAPool:   cfg.APIs.GRPC.TLS.CAPool,
			CertFile: cfg.APIs.GRPC.TLS.CertFile,
			KeyFile:  cfg.APIs.GRPC.TLS.KeyFile,
		},
	}, log.Get(), grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(func(ctx context.Context) (*oauth2.Token, error) {
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

	c := pb.NewAuthorizationServiceClient(conn.GRPC())

	ctx := kitNetGrpc.CtxWithToken(context.Background(), token)
	ctx = kitNetGrpc.CtxWithIncomingToken(ctx, token)
	owner, err := kitNetGrpc.ParseOwnerFromJwtToken("sub", token)
	require.NoError(t, err)

	naClient, subscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), log.Get())
	require.NoError(t, err)
	defer func() {
		subscriber.Close()
		naClient.Close()
	}()

	cacheExpiration := time.Second
	cache := authClient.NewOwnerCache("sub", cacheExpiration, subscriber.Conn(), c, func(err error) { fmt.Printf("%v\n", err) })
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
		_, err := c.AddDevice(ctx, &pb.AddDeviceRequest{DeviceId: d, UserId: owner})
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
		UserId:    owner,
	})
	require.NoError(t, err)
	assert.Equal(t, devices[:1], deleted.DeviceIds)

	// check update - after expiration
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

	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{DeviceId: devices[0], UserId: owner})
	require.NoError(t, err)
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

	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{DeviceId: devices[2], UserId: owner})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)
	cacheDevices, err = cache.GetDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:3], cacheDevices)

	_, err = c.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIds: devices[1:],
		UserId:    owner,
	})
	require.NoError(t, err)
	time.Sleep(cacheExpiration * 2)
	ownedDevices, err = cache.GetSelectedDevices(ctx, devices[1:])
	require.NoError(t, err)
	assert.Empty(t, ownedDevices)
}
