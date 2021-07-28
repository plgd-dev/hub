package service

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignOut(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
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
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()

	_, err := s.service.SignOut(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOutRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func TestUnexpectedAccessTokenOnSignOut(t *testing.T) {
	d := newTestDevice()
	d.AccessToken = "unexpected"
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
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
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, d)

	_, err := s.service.SignOut(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOutRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func newSignOutRequest() *pb.SignOutRequest {
	return &pb.SignOutRequest{
		DeviceId:    testDeviceID,
		UserId:      testUserID,
		AccessToken: testAccessToken,
	}
}
