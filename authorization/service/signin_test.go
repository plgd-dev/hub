package service

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/cloud/authorization/pb"

	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignIn(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, newTestDevice())
	r, err := s.service.SignIn(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignInRequest())
	require.NoError(t, err)
	assert := assert.New(t)
	assert.True(3595 < r.ExpiresIn && r.ExpiresIn <= 3600)
	_, ok := retrieveDevice(t, s.service.persistence, testDeviceID, testUserID)
	assert.True(ok)
}

func TestPermanentTokensExpiration(t *testing.T) {
	d := newTestDevice()
	d.Expiry = time.Time{}
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, d)

	r, err := s.service.SignIn(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignInRequest())

	assert := assert.New(t)
	assert.NoError(err)
	assert.Equal(int64(-1), r.ExpiresIn)
}

func TestUnauthorizedDevice(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()

	_, err := s.service.SignIn(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignInRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func TestExpiredAccessToken(t *testing.T) {
	d := newTestDevice()
	d.Expiry = time.Now().Add(-time.Minute)
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, d)

	_, err := s.service.SignIn(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignInRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func newSignInRequest() *pb.SignInRequest {
	return &pb.SignInRequest{
		DeviceId: testDeviceID,
		UserId:   testUserID,
	}
}
