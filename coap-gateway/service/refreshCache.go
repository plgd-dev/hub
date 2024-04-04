package service

import (
	"context"
	"errors"
	"sync"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
)

// Thread safe cache for Refresh operation.
//
// Refresh takes a refreshToken and returns access token. Cache keeps track of
// the last (refreshToken, oauth2.token) pair and if the authorization code for next Refresh
// call is the same as the cache value then the call is skipped and the stored token
// is returned instead.
type RefreshCache struct {
	token        *future.Future
	refreshToken string
	mutex        sync.Mutex
}

func NewRefreshCache() *RefreshCache {
	return &RefreshCache{}
}

func refresh(ctx context.Context, providers map[string]*oauth2.PlgdProvider, queue *queue.Queue, refreshToken string, logger log.Logger) (*oauth2.Token, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var token *oauth2.Token
	var err error
	for name, p := range providers {
		wg.Add(1)
		task := func(provider *oauth2.PlgdProvider) func() {
			return func() {
				defer wg.Done()
				tokenTmp, errTmp := provider.Refresh(ctx, refreshToken)
				mutex.Lock()
				defer mutex.Unlock()
				if errTmp == nil && token == nil {
					token = tokenTmp
					cancel()
				}
				if err == nil {
					err = errTmp
				}
			}
		}
		if errSubmit := queue.Submit(task(p)); errSubmit != nil {
			wg.Done()
			logger.Errorf("cannot refresh token for provider(%v): %w", name, errSubmit)
		}
	}
	wg.Wait()

	if token != nil {
		return token, nil
	}

	if err != nil {
		return nil, err
	}

	return nil, errors.New("invalid token")
}

func (r *RefreshCache) getFutureToken(refreshToken string) (*future.Future, future.SetFunc) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.token == nil || r.refreshToken != refreshToken {
		f, set := future.New()
		r.token = f
		r.refreshToken = refreshToken
		return f, set
	}
	return r.token, nil
}

func (r *RefreshCache) Execute(ctx context.Context, providers map[string]*oauth2.PlgdProvider, queue *queue.Queue, refreshToken string, logger log.Logger) (*oauth2.Token, error) {
	if refreshToken == "" {
		return nil, errors.New("invalid refreshToken")
	}

	f, set := r.getFutureToken(refreshToken)

	if set == nil {
		v, err := f.Get(ctx)
		if err != nil {
			return nil, err
		}
		return v.(*oauth2.Token), nil
	}

	token, err := refresh(ctx, providers, queue, refreshToken, logger)
	set(token, err)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (r *RefreshCache) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.refreshToken = ""
	r.token = nil
}
