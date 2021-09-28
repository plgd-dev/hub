package service

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/test/config"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDevicesList(t *testing.T) {
	jwtWithSubTestUserID := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": testUserID,
	})
	srv := newMockRetrieveResources(kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUserID))
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, newTestDevice())
	err := s.service.GetOwnerDevices(newGetOwnerDevicesRequest(), srv)
	assert.NoError(t, err)
	r := map[string]*pb.OwnerDevice{
		testDeviceID: {
			DeviceId: testDeviceID,
		},
	}
	assert.Equal(t, r, srv.resourceValues)
}

func TestListingMoreDevices(t *testing.T) {
	jwtWithSubTestUserID := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": testUserID,
	})
	srv := newMockRetrieveResources(kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUserID))
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

	err := s.service.GetOwnerDevices(newGetOwnerDevicesRequest(), srv)
	assert := assert.New(t)
	assert.NoError(err)
	r := map[string]*pb.OwnerDevice{
		testDeviceID: {
			DeviceId: testDeviceID,
		},
		d.DeviceID: {
			DeviceId: d.DeviceID,
		},
	}
	assert.Equal(r, srv.resourceValues)
}

func newGetOwnerDevicesRequest() *pb.GetOwnerDevicesRequest {
	return &pb.GetOwnerDevicesRequest{}
}

type mockGetOwnerDevicesServer struct {
	resourceValues map[string]*pb.OwnerDevice
	ctx            context.Context
	grpc.ServerStream
}

func newMockRetrieveResources(ctx context.Context) *mockGetOwnerDevicesServer {
	return &mockGetOwnerDevicesServer{
		ctx: ctx,
	}
}

func (d *mockGetOwnerDevicesServer) Send(r *pb.OwnerDevice) error {
	if d.resourceValues == nil {
		d.resourceValues = make(map[string]*pb.OwnerDevice)
	}
	d.resourceValues[r.DeviceId] = r
	return nil
}

func (d *mockGetOwnerDevicesServer) Context() context.Context {
	return d.ctx
}
