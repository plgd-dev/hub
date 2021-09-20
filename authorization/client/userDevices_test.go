package client

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/authorization/pb"
	authService "github.com/plgd-dev/cloud/authorization/test"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	clientCertManager "github.com/plgd-dev/cloud/pkg/security/certManager/client"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

type testTrigger struct {
	sync.Mutex
	addedDevices   map[string]map[string]bool
	removedDevices map[string]map[string]bool
	allDevices     map[string]map[string]bool
}

func (t *testTrigger) Clone() *testTrigger {
	t.Lock()
	defer t.Unlock()
	a := newTestTrigger()
	for userID, addedDevices := range t.addedDevices {
		a.Trigger(context.TODO(), userID, addedDevices, nil, nil)
	}
	for userID, removedDevices := range t.removedDevices {
		a.Trigger(context.TODO(), userID, nil, removedDevices, nil)
	}
	for userID, allDevices := range t.allDevices {
		a.Trigger(context.TODO(), userID, nil, nil, allDevices)
	}

	return a
}

func newTestTrigger() *testTrigger {
	return &testTrigger{}
}

func (t *testTrigger) Trigger(ctx context.Context, userID string, addedDevices, removedDevices, allDevices map[string]bool) {
	t.Lock()
	defer t.Unlock()
	if len(addedDevices) > 0 {
		if t.addedDevices == nil {
			t.addedDevices = make(map[string]map[string]bool)
		}
		devices, ok := t.addedDevices[userID]
		if !ok {
			devices = make(map[string]bool)
			t.addedDevices[userID] = devices
		}
		for deviceID := range addedDevices {
			devices[deviceID] = true
		}
	}
	if len(removedDevices) > 0 {
		if t.removedDevices == nil {
			t.removedDevices = make(map[string]map[string]bool)
		}
		devices, ok := t.removedDevices[userID]
		if !ok {
			devices = make(map[string]bool)
			t.removedDevices[userID] = devices
		}
		for deviceID := range removedDevices {
			devices[deviceID] = true
		}
	}
	if len(allDevices) == 0 {
		t.allDevices = nil
		return
	}
	if t.allDevices == nil {
		t.allDevices = make(map[string]map[string]bool)
	}
	devices := make(map[string]bool)
	t.allDevices[userID] = devices

	for deviceID := range allDevices {
		devices[deviceID] = true
	}
}

func TestAddDeviceAfterRegister(t *testing.T) {
	trigger := newTestTrigger()

	cfg := authService.MakeConfig(t)
	cfg.APIs.GRPC.Addr = "localhost:1234"

	oauthShutdown := oauthService.SetUp(t)
	defer oauthShutdown()

	shutdown := authService.New(t, cfg)
	defer shutdown()

	token := oauthService.GetDefaultServiceToken(t)

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

	m := NewUserDevicesManager(trigger.Trigger, c, time.Millisecond*200, time.Millisecond*500, func(err error) { fmt.Println(err) })
	defer m.Close()
	err = m.Acquire(ctx, t.Name())
	require.NoError(t, err)

	deviceID := "deviceId_" + t.Name()
	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{
		UserId:   t.Name(),
		DeviceId: deviceID,
	})
	require.NoError(t, err)

	time.Sleep(time.Second * 2)
	require.Equal(t, map[string]map[string]bool{
		t.Name(): {
			deviceID: true,
		},
	}, trigger.Clone().allDevices)

	for i := 0; i < 5; i++ {
		devs, err := m.GetUserDevices(ctx, t.Name())
		require.NoError(t, err)
		require.NotEmpty(t, devs)
	}

	resp, err := c.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		UserId:    t.Name(),
		DeviceIds: []string{deviceID},
	})
	require.NoError(t, err)
	require.Equal(t, []string{deviceID}, resp.DeviceIds)

	time.Sleep(time.Second * 2)
	require.Equal(t, map[string]map[string]bool(nil), trigger.Clone().allDevices)

	err = m.Release(t.Name())
	require.NoError(t, err)

	devs, err := m.GetUserDevices(ctx, t.Name())
	require.NoError(t, err)
	require.Empty(t, devs)

	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{
		UserId:   t.Name(),
		DeviceId: deviceID,
	})
	require.NoError(t, err)
	time.Sleep(time.Second * 2)

	devs, err = m.GetUserDevices(ctx, t.Name())
	require.NoError(t, err)
	require.NotEmpty(t, devs)

	err = m.Release(t.Name())
	require.NoError(t, err)
}

func TestUserDevicesManager_Acquire(t *testing.T) {
	type fields struct {
		trigger *testTrigger
	}
	type args struct {
		userID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *testTrigger
	}{
		{
			name: "empty - user not exist",
			fields: fields{
				trigger: newTestTrigger(),
			},
			args: args{
				userID: "notExist",
			},
			want: &testTrigger{},
		},
		{
			name: "valid",
			fields: fields{
				trigger: newTestTrigger(),
			},
			args: args{
				userID: t.Name(),
			},
			want: &testTrigger{
				addedDevices: map[string]map[string]bool{
					t.Name(): {
						"deviceId_" + t.Name(): true,
					},
				},
				allDevices: map[string]map[string]bool{
					t.Name(): {
						"deviceId_" + t.Name(): true,
					},
				},
			},
		},
	}

	cfg := authService.MakeConfig(t)
	cfg.APIs.GRPC.Addr = "localhost:1234"

	oauthShutdown := oauthService.SetUp(t)
	defer oauthShutdown()

	token := oauthService.GetDefaultServiceToken(t)

	shutdown := authService.New(t, cfg)
	defer shutdown()

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

	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{
		UserId:   t.Name(),
		DeviceId: "deviceId_" + t.Name(),
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewUserDevicesManager(tt.fields.trigger.Trigger, c, time.Millisecond*200, time.Second, func(err error) { fmt.Println(err) })
			defer m.Close()
			err := m.Acquire(ctx, tt.args.userID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				time.Sleep(time.Second)
				require.Equal(t, tt.want, tt.fields.trigger.Clone())
				err := m.Release(tt.args.userID)
				require.NoError(t, err)
			}
		})
	}
}

func TestUserDevicesManager_Release(t *testing.T) {
	type fields struct {
		trigger *testTrigger
	}
	type args struct {
		userID string
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		want         *testTrigger
		wantMgmtSize int
	}{
		{
			name: "empty - user not exist",
			fields: fields{
				trigger: newTestTrigger(),
			},
			args: args{
				userID: "notExist",
			},
			want: &testTrigger{},
		},
		{
			name: "valid",
			fields: fields{
				trigger: newTestTrigger(),
			},
			args: args{
				userID: t.Name(),
			},
			want: &testTrigger{
				addedDevices: map[string]map[string]bool{
					t.Name(): {
						"deviceId_" + t.Name(): true,
					},
				},
				removedDevices: map[string]map[string]bool{
					t.Name(): {
						"deviceId_" + t.Name(): true,
					},
				},
			},
			wantMgmtSize: 0,
		},
	}

	cfg := authService.MakeConfig(t)
	cfg.APIs.GRPC.Addr = "localhost:1234"

	oauthShutdown := oauthService.SetUp(t)
	defer oauthShutdown()

	token := oauthService.GetDefaultServiceToken(t)

	shutdown := authService.New(t, cfg)
	defer shutdown()

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

	_, err = c.AddDevice(ctx, &pb.AddDeviceRequest{
		UserId:   t.Name(),
		DeviceId: "deviceId_" + t.Name(),
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewUserDevicesManager(tt.fields.trigger.Trigger, c, time.Millisecond*200, time.Millisecond*500, func(err error) { fmt.Println(err) })
			defer m.Close()
			err := m.Acquire(ctx, tt.args.userID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				time.Sleep(time.Second)
				err := m.Release(tt.args.userID)
				require.NoError(t, err)
				require.Equal(t, tt.want, tt.fields.trigger.Clone())
				require.Equal(t, tt.wantMgmtSize, len(m.users))
			}
		})
	}
}
