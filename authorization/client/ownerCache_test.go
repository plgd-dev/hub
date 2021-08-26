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

func Test_ownerCache_Subscribe(t *testing.T) {
	test.ClearDB(context.Background(), t)

	devices := []string{"device1", "device2", "device3"}
	cfg := authService.MakeConfig(t)
	cfg.APIs.GRPC.Addr = "localhost:1234"

	oauthShutdown := oauthService.SetUp(t)
	defer oauthShutdown()

	shutdown := authService.New(t, cfg)
	defer shutdown()

	naClient, subscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), log.Get())
	require.NoError(t, err)
	defer func() {
		subscriber.Close()
		naClient.Close()
	}()

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

	cache := authClient.NewOwnerCache("sub", time.Second, subscriber.Conn(), c, func(err error) { fmt.Printf("%v\n", err) })
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
	cacheDevices, ok := cache.GetDevices(owner)
	assert.Empty(t, cacheDevices)
	assert.False(t, ok)

	lock.Lock()
	sort.Strings(sub1registeredDevices)
	sort.Strings(sub2registeredDevices)
	assert.Equal(t, devices[:2], sub1registeredDevices)
	assert.Equal(t, devices[:2], sub2registeredDevices)
	lock.Unlock()

	// check update
	added, removed, err := cache.Update(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], added)
	assert.Empty(t, removed)

	cacheDevices, ok = cache.GetDevices(owner)
	assert.True(t, ok)
	assert.Equal(t, devices[:2], cacheDevices)

	deleted, err := c.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIds: devices[:1],
		UserId:    owner,
	})
	require.NoError(t, err)
	require.Equal(t, devices[:1], deleted.DeviceIds)
	cacheDevices, ok = cache.GetDevices(owner)
	assert.Equal(t, devices[1:2], cacheDevices)
	assert.True(t, ok)

	lock.Lock()
	sort.Strings(sub1unregisteredDevices)
	assert.Equal(t, devices[0:1], sub1unregisteredDevices)
	sort.Strings(sub2unregisteredDevices)
	assert.Equal(t, devices[0:1], sub2unregisteredDevices)
	lock.Unlock()

	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{DeviceId: devices[0], UserId: owner})
	require.NoError(t, err)
	cacheDevices, ok = cache.GetDevices(owner)
	assert.Equal(t, devices[:2], cacheDevices)
	assert.True(t, ok)

	// check cleanup cache
	sub1Close()
	time.Sleep(time.Second * 2)
	cacheDevices, ok = cache.GetDevices(owner)
	assert.Empty(t, cacheDevices)
	assert.False(t, ok)
	sub2Close()
	cacheDevices, ok = cache.GetDevices(owner)
	assert.Empty(t, cacheDevices)
	assert.False(t, ok)

	// cache without subscription to events
	added, removed, err = cache.Update(ctx)
	require.NoError(t, err)
	assert.Equal(t, devices[:2], added)
	assert.Empty(t, removed)

	cacheDevices, ok = cache.GetDevices(owner)
	assert.Equal(t, devices[:2], cacheDevices)
	assert.True(t, ok)

	added, removed, err = cache.Update(ctx)
	require.NoError(t, err)
	assert.Empty(t, added)
	assert.Empty(t, removed)
	cacheDevices, ok = cache.GetDevices(owner)
	assert.Equal(t, devices[:2], cacheDevices)
	assert.True(t, ok)
	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{DeviceId: devices[2], UserId: owner})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)
	cacheDevices, ok = cache.GetDevices(owner)
	assert.Equal(t, devices[:3], cacheDevices)
	assert.True(t, ok)

	time.Sleep(time.Second * 2)
	cacheDevices, ok = cache.GetDevices(owner)
	assert.Empty(t, cacheDevices)
	assert.False(t, ok)
}
