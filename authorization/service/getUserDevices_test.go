package service

import (
	"context"
	"testing"

	"github.com/go-ocf/cloud/authorization/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
)

func TestUserDevicesList(t *testing.T) {
	srv := newMockRetrieveResourcesValues(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken))
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, newTestDevice())
	err := s.service.GetUserDevices(newGetUserDevicesRequest(), srv)
	assert.NoError(t, err)
	r := map[string]*pb.UserDevice{
		testDeviceID: &pb.UserDevice{
			DeviceId: testDeviceID,
			UserId:   testUserID,
		},
	}
	assert.Equal(t, r, srv.resourceValues)
}

func TestListingMoreDevices(t *testing.T) {
	srv := newMockRetrieveResourcesValues(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken))
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, newTestDevice())
	d := newTestDevice()
	d.DeviceID = "anotherDeviceID"
	persistDevice(t, s.service.persistence, d)

	err := s.service.GetUserDevices(newGetUserDevicesRequest(), srv)
	assert := assert.New(t)
	assert.NoError(err)
	r := map[string]*pb.UserDevice{
		testDeviceID: &pb.UserDevice{
			DeviceId: testDeviceID,
			UserId:   testUserID,
		},
		d.DeviceID: &pb.UserDevice{
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

func newMockRetrieveResourcesValues(ctx context.Context) *mockGetUserDevicesServer {
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
