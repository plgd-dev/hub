package service

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/authorization/pb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignIn(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, newTestDevice())
	r, err := s.service.SignIn(context.Background(), newSignInRequest())
	require.NoError(t, err)
	assert := assert.New(t)
	assert.True(0 < r.ValidUntil)
	_, ok := retrieveDevice(t, s.service.persistence, testDeviceID, testUserID)
	assert.True(ok)
}

func TestPermanentTokensExpiration(t *testing.T) {
	d := newTestDevice()
	d.Expiry = time.Time{}
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, d)

	r, err := s.service.SignIn(context.Background(), newSignInRequest())
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Equal(int64(0), r.GetValidUntil())
}

func TestUnauthorizedDevice(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()

	_, err := s.service.SignIn(context.Background(), newSignInRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func TestExpiredAccessToken(t *testing.T) {
	d := newTestDevice()
	d.Expiry = time.Now().Add(-time.Minute)
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, d)

	_, err := s.service.SignIn(context.Background(), newSignInRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func newSignInRequest() *pb.SignInRequest {
	return &pb.SignInRequest{
		DeviceId:    testDeviceID,
		UserId:      testUserID,
		AccessToken: testAccessToken,
	}
}
