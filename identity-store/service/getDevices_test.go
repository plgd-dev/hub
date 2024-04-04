package service

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
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
	err := s.service.GetDevices(newGetDevicesRequest(), srv)
	require.NoError(t, err)
	r := map[string]*pb.Device{
		testDeviceID: {
			DeviceId: testDeviceID,
		},
	}
	require.Equal(t, r, srv.resourceValues)
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
	d.DeviceID = test.GenerateDeviceIDbyIdx(1)
	persistDevice(t, s.service.persistence, d)

	err := s.service.GetDevices(newGetDevicesRequest(), srv)
	require.NoError(t, err)
	r := map[string]*pb.Device{
		testDeviceID: {
			DeviceId: testDeviceID,
		},
		d.DeviceID: {
			DeviceId: d.DeviceID,
		},
	}
	require.Equal(t, r, srv.resourceValues)
}

func newGetDevicesRequest() *pb.GetDevicesRequest {
	return &pb.GetDevicesRequest{}
}

type mockGeDevicesServer struct {
	resourceValues map[string]*pb.Device
	ctx            context.Context
	grpc.ServerStream
}

func newMockRetrieveResources(ctx context.Context) *mockGeDevicesServer {
	return &mockGeDevicesServer{
		ctx: ctx,
	}
}

func (d *mockGeDevicesServer) Send(r *pb.Device) error {
	if d.resourceValues == nil {
		d.resourceValues = make(map[string]*pb.Device)
	}
	d.resourceValues[r.GetDeviceId()] = r
	return nil
}

func (d *mockGeDevicesServer) Context() context.Context {
	return d.ctx
}
