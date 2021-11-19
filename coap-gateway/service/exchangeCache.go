package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/hub/pkg/security/oauth2"
	"github.com/plgd-dev/hub/pkg/sync/task/future"
)

// Thread safe cache for Exchange operation.
//
// Exchange takes authorization code and returns access token. Cache keeps track of
// the last (code, oauth2.token) pair and if the authorization code for next Exchange
// call is the same as the cached value then the call is skipped and the stored token
// is returned instead.
type exchangeCache struct {
	token *future.Future
	code  string
	mutex sync.Mutex
}

func NewExchangeCache() *exchangeCache {
	return &exchangeCache{}
}

func (e *exchangeCache) getFutureToken(authorizationCode string) (*future.Future, future.SetFunc) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.token == nil || e.code != authorizationCode {
		f, set := future.New()
		e.token = f
		e.code = authorizationCode
		return f, set
	}
	return e.token, nil
}

// Execute Exchange or returned cached value.
func (e *exchangeCache) Execute(ctx context.Context, provider *oauth2.PlgdProvider, authorizationCode string) (*oauth2.Token, error) {
	if authorizationCode == "" {
		return nil, fmt.Errorf("invalid authorization code")
	}

	f, set := e.getFutureToken(authorizationCode)

	if set == nil {
		v, err := f.Get(ctx)
		if err != nil {
			return nil, err
		}
		return v.(*oauth2.Token), nil
	}

	token, err := provider.Exchange(ctx, authorizationCode)
	set(token, err)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// Clear stored value.
func (e *exchangeCache) Clear() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.code = ""
	e.token = nil
}
