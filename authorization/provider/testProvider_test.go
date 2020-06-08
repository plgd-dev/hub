package provider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSignUpTestProvider(t *testing.T) {
	p := NewTestProvider()
	ctx := context.Background()
	token, err := p.Exchange(ctx, "", "authCode")

	assert := assert.New(t)
	assert.Nil(err)
	assert.NotEmpty(token.AccessToken)
	assert.Equal("refresh-token", token.RefreshToken)
	expiresIn := int(token.Expiry.Sub(time.Now()).Seconds())
	assert.True(expiresIn > 0)
	assert.Equal("1", token.UserID)
}

func TestRefreshTokenTestProvider(t *testing.T) {
	p := NewTestProvider()
	ctx := context.Background()
	token, err := p.Refresh(ctx, "refresh-token")

	assert := assert.New(t)
	assert.Nil(err)
	assert.NotEmpty(token.AccessToken)
	assert.Equal("refresh-token", token.RefreshToken)
	expiresIn := int(token.Expiry.Sub(time.Now()).Seconds())
	assert.True(expiresIn > 0)
	assert.Equal("1", token.UserID)
}
