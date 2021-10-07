package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/pkg/security/oauth2"
)

// Non-thread safe cache for Exchange operation.
//
// Exchange takes authorization code and returns access token. Cache keeps track of
// the last (code, oauth2.token) pair and if the authorization code for next Exchange
// call is the same as the cached value then the call is skipped and the stored token
// is returned instead.
type exchangeCache struct {
	code  string
	token oauth2.Token
}

func NewExchangeCache() *exchangeCache {
	return &exchangeCache{}
}

// Execute Exchange or returned cached value.
func (e *exchangeCache) Execute(ctx context.Context, provider *oauth2.PlgdProvider, authorizationCode string) (oauth2.Token, error) {
	if authorizationCode == "" {
		return oauth2.Token{}, fmt.Errorf("invalid authorization code")
	}

	if authorizationCode == e.code {
		return e.token, nil
	}

	token, err := provider.Exchange(ctx, authorizationCode)
	if err != nil {
		return oauth2.Token{}, err
	}

	e.code = authorizationCode
	e.token = *token
	return e.token, nil
}

// Clear stored value.
func (e *exchangeCache) Clear() {
	e.code = ""
	e.token = oauth2.Token{}
}
