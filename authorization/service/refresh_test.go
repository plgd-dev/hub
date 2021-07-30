package service

import (
	"context"
	"errors"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/authorization/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestRefreshToken(t *testing.T) {
	s, shutdown := newTestServiceWithProviders(t, NewTestProvider(), NewTestProvider())
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	r, err := s.service.RefreshToken(context.Background(), newRefreshInRequest())
	require.NoError(t, err)
	assert := assert.New(t)
	assert.NotEmpty(r.AccessToken)
	assert.Equal("refresh-token", r.RefreshToken)
	assert.True(r.ValidUntil > 0)
}

func TestUnauthorizedRefreshToken(t *testing.T) {
	s, shutdown := newTestServiceWithProviders(t, NewTestProvider(), NewTestProvider())
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	s.service.deviceProvider = &providerT{t: nil, err: errors.New("unauthorized")}
	_, err := s.service.RefreshToken(context.Background(), newRefreshInRequest())
	assert.Error(t, err)
}

func newRefreshInRequest() *pb.RefreshTokenRequest {
	return &pb.RefreshTokenRequest{
		DeviceId:     "ignored",
		UserId:       "ignored",
		RefreshToken: "ignored",
	}
}
