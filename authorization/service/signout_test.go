package service

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/ocf-cloud/authorization/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/assert"
)

func TestSignOut(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	d := newTestDevice()
	persistDevice(t, s.service.persistence, d)

	r, err := s.service.SignOut(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOutRequest())
	assert := assert.New(t)
	assert.NoError(err)
	assert.Equal(&pb.SignOutResponse{}, r)

	_, ok := retrieveDevice(t, s.service.persistence, testDeviceID, testUserID)
	assert.True(ok)
}

func TestSigningOutUnknownDevice(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()

	_, err := s.service.SignOut(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOutRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func TestUnexpectedAccessTokenOnSignOut(t *testing.T) {
	d := newTestDevice()
	d.AccessToken = "unexpected"
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, d)

	_, err := s.service.SignOut(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOutRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func TestExpiredAccessTokenOnSignOut(t *testing.T) {
	d := newTestDevice()
	d.Expiry = time.Now().Add(-time.Minute)
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, d)

	_, err := s.service.SignOut(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOutRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func newSignOutRequest() *pb.SignOutRequest {
	return &pb.SignOutRequest{
		DeviceId: testDeviceID,
		UserId:   testUserID,
	}
}
