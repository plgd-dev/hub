package service

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDevicesList(t *testing.T) {
	srv := newMockRetrieveResources(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken))
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, newTestDevice())
	err := s.service.GetUserDevices(newGetUserDevicesRequest(), srv)
	assert.NoError(t, err)
	r := map[string]*pb.UserDevice{
		testDeviceID: {
			DeviceId: testDeviceID,
			UserId:   testUserID,
		},
	}
	assert.Equal(t, r, srv.resourceValues)
}

func TestListingMoreDevices(t *testing.T) {
	srv := newMockRetrieveResources(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken))
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, newTestDevice())
	d := newTestDevice()
	d.DeviceID = "anotherDeviceID"
	persistDevice(t, s.service.persistence, d)

	err := s.service.GetUserDevices(newGetUserDevicesRequest(), srv)
	assert := assert.New(t)
	assert.NoError(err)
	r := map[string]*pb.UserDevice{
		testDeviceID: {
			DeviceId: testDeviceID,
			UserId:   testUserID,
		},
		d.DeviceID: {
			DeviceId: d.DeviceID,
			UserId:   testUserID,
		},
	}
	assert.Equal(r, srv.resourceValues)
}

func newGetUserDevicesRequest() *pb.GetUserDevicesRequest {
	return &pb.GetUserDevicesRequest{
		UserIdsFilter: []string{testUserID},
	}
}

type mockGetUserDevicesServer struct {
	resourceValues map[string]*pb.UserDevice
	ctx            context.Context
	grpc.ServerStream
}

func newMockRetrieveResources(ctx context.Context) *mockGetUserDevicesServer {
	return &mockGetUserDevicesServer{
		ctx: ctx,
	}
}

func (d *mockGetUserDevicesServer) Send(r *pb.UserDevice) error {
	if d.resourceValues == nil {
		d.resourceValues = make(map[string]*pb.UserDevice)
	}
	d.resourceValues[r.DeviceId] = r
	return nil
}

func (d *mockGetUserDevicesServer) Context() context.Context {
	return d.ctx
}
