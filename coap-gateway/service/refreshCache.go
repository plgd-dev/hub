package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/pkg/security/oauth2"
	"github.com/plgd-dev/hub/pkg/sync/task/queue"
)

// Non-thread safe cache for Refresh operation.
//
// Refresh takes a refreshToken and returns access token. Cache keeps track of
// the last (refreshToken, oauth2.token) pair and if the authorization code for next Refresh
// call is the same as the cache value then the call is skipped and the stored token
// is returned instead.
type refreshCache struct {
	refreshToken string
	token        oauth2.Token
}

func NewRefreshCache() *refreshCache {
	return &refreshCache{}
}

func refresh(ctx context.Context, providers map[string]*oauth2.PlgdProvider, queue *queue.Queue, refreshToken string) (oauth2.Token, error) {
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
			log.Errorf("cannot refresh token for provider(%v): %w", name, errSubmit)
		}
	}
	wg.Wait()

	if token != nil {
		return *token, nil
	}

	if err != nil {
		return oauth2.Token{}, err
	}

	return oauth2.Token{}, fmt.Errorf("invalid token")
}

func (r *refreshCache) Execute(ctx context.Context, providers map[string]*oauth2.PlgdProvider, queue *queue.Queue, refreshToken string) (oauth2.Token, error) {
	if refreshToken == "" {
		return oauth2.Token{}, fmt.Errorf("invalid refreshToken")
	}

	if refreshToken == r.refreshToken {
		return r.token, nil
	}

	token, err := refresh(ctx, providers, queue, refreshToken)
	if err != nil {
		return oauth2.Token{}, err
	}

	r.refreshToken = refreshToken
	r.token = token
	return r.token, nil
}

func (r *refreshCache) Clear() {
	r.refreshToken = ""
	r.token = oauth2.Token{}
}
