package clientcredentials

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

type Cache struct {
	ctx        context.Context
	tokens     map[string]*future.Future
	mutex      sync.Mutex
	httpClient *client.Client
	config     Config
	cancel     context.CancelFunc
	wg         *sync.WaitGroup
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, cleanUpInterval time.Duration) (*Cache, error) {
	err := config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid OAuth client credential config: %w", err)
	}
	httpClient, err := cmClient.NewHTTPClient(&config.HTTP, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, err
	}
	oidcfg, err := openid.GetConfiguration(ctx, httpClient.HTTP(), config.Authority)
	if err != nil {
		return nil, err
	}
	config.TokenURL = oidcfg.TokenURL

	ctx, cancel := context.WithCancel(ctx)
	c := &Cache{
		config:     config,
		httpClient: httpClient,
		tokens:     make(map[string]*future.Future),
		cancel:     cancel,
		wg:         new(sync.WaitGroup),
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		t := time.NewTicker(cleanUpInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				c.httpClient.Close()
				return
			case now := <-t.C:
				c.cleanUpExpiredTokens(now)
			}
		}
	}()

	return c, nil
}

func (c *Cache) cleanUpExpiredTokens(now time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for owner, f := range c.tokens {
		if !f.Ready() {
			continue
		}
		v, err := f.Get(c.ctx)
		if err != nil {
			delete(c.tokens, owner)
			continue
		}
		t, ok := v.(*oauth2.Token)
		if !ok || (!t.Expiry.IsZero() && now.After(t.Expiry)) {
			delete(c.tokens, owner)
		}
	}
}

type ClaimNotFoundError struct {
	Claim    string
	Expected interface{}
	Got      interface{}
}

func NewClaimNotFoundError(claim string, expected, got interface{}) ClaimNotFoundError {
	return ClaimNotFoundError{
		Claim:    claim,
		Expected: expected,
		Got:      got,
	}
}

func (e ClaimNotFoundError) Error() string {
	return fmt.Sprintf("invalid claim %v: expected %v, got %v", e.Claim, e.Expected, e.Got)
}

func checkRequiredClaims(accessToken string, requiredClaims map[string]interface{}) error {
	claims, err := jwt.ParseToken(accessToken)
	if err != nil {
		return fmt.Errorf("cannot parse access token: %w", err)
	}
	for k, v := range requiredClaims {
		if claims[k] != v {
			return NewClaimNotFoundError(k, v, claims[k])
		}
	}
	return nil
}

func (c *Cache) GetTokenFromOAuth(ctx context.Context, urlValues map[string]string, requiredClaims map[string]interface{}) (*oauth2.Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient.HTTP())
	csCfg := c.config.ToDefaultClientCredentials()
	for key, val := range urlValues {
		csCfg.EndpointParams.Add(key, val)
	}
	token, err := csCfg.Token(ctx)
	if err == nil {
		err = checkRequiredClaims(token.AccessToken, requiredClaims)
	}
	return token, err
}

func (c *Cache) getFutureToken(key string, old *future.Future) (*future.Future, future.SetFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	f, ok := c.tokens[key]
	if !ok || f == old {
		fu, set := future.New()
		c.tokens[key] = fu
		return fu, set
	}
	return f, nil
}

func (c *Cache) GetToken(ctx context.Context, key string, urlValues map[string]string, requiredClaims map[string]interface{}) (*oauth2.Token, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, errors.New("deadline is not set in ctx")
	}
	var oldF *future.Future
	var setF future.SetFunc
	for {
		f, set := c.getFutureToken(key, oldF)
		if set != nil {
			setF = set
			break
		}
		v, err := f.Get(ctx)
		if err == nil {
			t, ok := v.(*oauth2.Token)
			if !ok {
				return nil, fmt.Errorf("invalid object type(%T) in a future", v)
			}
			if checkRequiredClaims(t.AccessToken, requiredClaims) == nil && (deadline.Before(t.Expiry) || t.Expiry.IsZero()) {
				return t, nil
			}
		}
		oldF = f
	}
	token, err := c.GetTokenFromOAuth(ctx, urlValues, requiredClaims)
	setF(token, err)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (c *Cache) Close() {
	c.cancel()
	c.wg.Wait()
}
