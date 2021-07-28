package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignUp(t *testing.T) {
	s, o, shutdown := newSignUpTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()

	r, err := s.service.SignUp(context.Background(), newSignUpRequest())
	assert := assert.New(t)
	assert.NoError(err)

	assert.Equal(o.t.AccessToken, r.AccessToken)
	assert.Equal(o.t.RefreshToken, r.RefreshToken)
	assert.True(r.ValidUntil > 0)
	assert.Equal(o.t.Owner, r.UserId)
	_, ok := retrieveDevice(t, s.service.persistence, testDeviceID, r.UserId)
	assert.True(ok)
}

func TestUnknownProvider(t *testing.T) {
	r := newSignUpRequest()
	r.AuthorizationProvider = "unknown"

	s, _, shutdown := newSignUpTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	s.service.deviceProvider = &providerT{t: nil, err: errors.New("invalid provider")}

	_, err := s.service.SignUp(context.Background(), r)
	assert := assert.New(t)
	assert.Error(err)
}

func TestUnauthorizedSignUp(t *testing.T) {
	s, _, shutdown := newSignUpTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	s.service.deviceProvider = &providerT{t: nil, err: errors.New("unauthorized")}
	_, err := s.service.SignUp(context.Background(), newSignUpRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func TestPermanentToken(t *testing.T) {
	s, o, shutdown := newSignUpTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	o.t.Expiry = time.Time{} // 0 means permanent

	r, err := s.service.SignUp(context.Background(), newSignUpRequest())
	assert := assert.New(t)
	assert.NoError(err)
	assert.Equal(o.t.AccessToken, r.AccessToken)
	assert.Equal(o.t.RefreshToken, r.RefreshToken)
	assert.Equal(int64(0), r.ValidUntil)
	assert.Equal(o.t.Owner, r.UserId)
}

type providerT struct {
	t   *provider.Token
	err error
}

func (p *providerT) GetProviderName() string {
	return "test"
}

func (p *providerT) Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*provider.Token, error) {
	return p.t, p.err
}

func (p *providerT) Refresh(ctx context.Context, refreshToken string) (*provider.Token, error) {
	return p.t, p.err
}

func (p *providerT) AuthCodeURL(csrfToken string) string {
	return "redirect-url"
}

func newTestProvider() *providerT {
	t := provider.Token{
		AccessToken:  "test access token",
		RefreshToken: "test refresh token",
		Expiry:       time.Now().Add(3600 * time.Second),
		Owner:        "test user id",
	}
	return &providerT{t: &t, err: nil}
}

func newSignUpTestService(t *testing.T) (*Server, *providerT, func()) {
	s, shutdown := newTestService(t)
	o := newTestProvider()
	s.service.deviceProvider = o
	return s, o, shutdown
}

func newSignUpRequest() *pb.SignUpRequest {
	return &pb.SignUpRequest{
		DeviceId:              testDeviceID,
		AuthorizationCode:     "authCode",
		AuthorizationProvider: "test",
	}
}
