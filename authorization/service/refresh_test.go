package service

import (
	"context"
	"errors"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken(t *testing.T) {
	s, shutdown := newTestServiceWithProviders(t, NewTestProvider(), NewTestProvider())
	defer shutdown()
	defer s.cleanUp()
	r, err := s.service.RefreshToken(context.Background(), newRefreshInRequest())
	require.NoError(t, err)
	assert := assert.New(t)
	assert.NotEmpty(r.AccessToken)
	assert.Equal("refresh-token", r.RefreshToken)
	assert.True(r.ExpiresIn > 0)
}

func TestUnauthorizedRefreshToken(t *testing.T) {
	s, shutdown := newTestServiceWithProviders(t, NewTestProvider(), NewTestProvider())
	defer shutdown()
	defer s.cleanUp()
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
