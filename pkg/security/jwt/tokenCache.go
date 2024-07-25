package jwt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"go.uber.org/atomic"
)

type Client struct {
	*http.Client
	tokenEndpoint string
}

func NewClient(client *http.Client, tokenEndpoint string) *Client {
	return &Client{
		Client:        client,
		tokenEndpoint: tokenEndpoint,
	}
}

type tokenRecord struct {
	blacklisted bool
	validUntil  atomic.Time
	onExpire    func(uuid.UUID)
}

func newTokenRecord(blacklisted bool, validUntil time.Time, onExpire func(uuid.UUID)) *tokenRecord {
	t := tokenRecord{
		blacklisted: blacklisted,
		onExpire:    onExpire,
	}
	t.validUntil.Store(validUntil)
	return &t
}

func (t *tokenRecord) IsExpired(now time.Time) bool {
	value := t.validUntil.Load()
	if value.IsZero() {
		return false
	}
	return now.After(value)
}

type tokenOrFuture struct {
	tokenOrFuture interface{}
}

func makeTokenOrFuture(token *tokenRecord, tokenFuture *future.Future) tokenOrFuture {
	if token != nil {
		return tokenOrFuture{token}
	}
	return tokenOrFuture{tokenFuture}
}

func (tf *tokenOrFuture) Get(ctx context.Context) (*tokenRecord, error) {
	if tr, ok := tf.tokenOrFuture.(*tokenRecord); ok {
		return tr, nil
	}
	tv, err := tf.tokenOrFuture.(*future.Future).Get(ctx)
	if err != nil {
		return nil, err
	}
	return tv.(*tokenRecord), nil
}

type tokenIssuerCache struct {
	client        *http.Client
	tokenEndpoint string
	tokens        map[uuid.UUID]tokenOrFuture
	mutex         sync.Mutex
}

func newTokenIssuerCache(client *Client) *tokenIssuerCache {
	return &tokenIssuerCache{
		client:        client.Client,
		tokenEndpoint: client.tokenEndpoint,
		tokens:        make(map[uuid.UUID]tokenOrFuture),
	}
}

func (tc *tokenIssuerCache) getValidTokenRecordOrFuture(tokenID uuid.UUID) (tokenOrFuture, future.SetFunc) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tf, ok := tc.tokens[tokenID]
	if !ok {
		f, set := future.New()
		newTf := makeTokenOrFuture(nil, f)
		tc.tokens[tokenID] = newTf
		return newTf, set
	}

	if tr, ok := tf.tokenOrFuture.(*tokenRecord); ok && tr.IsExpired(time.Now()) {
		if tr.onExpire != nil {
			tr.onExpire(tokenID)
		}
		f, set := future.New()
		newTr := makeTokenOrFuture(nil, f)
		tc.tokens[tokenID] = newTr
		return newTr, set
	}
	return tf, nil
}

func (tc *tokenIssuerCache) removeToken(tokenID uuid.UUID) {
	delete(tc.tokens, tokenID)
}

func (tc *tokenIssuerCache) setTokenRecord(tokenID uuid.UUID, tr *tokenRecord) {
	tf := makeTokenOrFuture(tr, nil)
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.tokens[tokenID] = tf
}

type TokenCache struct {
	expiration time.Duration
	cache      map[string]*tokenIssuerCache
	logger     log.Logger
}

func NewTokenCache(clients map[string]*Client, expiration time.Duration, logger log.Logger) *TokenCache {
	tc := &TokenCache{
		expiration: expiration,
		logger:     logger,
	}
	if len(clients) > 0 {
		tc.cache = make(map[string]*tokenIssuerCache)
		for issuer, client := range clients {
			tc.cache[issuer] = newTokenIssuerCache(client)
		}
	}
	return tc
}

func (t *TokenCache) getValidUntil(token *pb.Token) time.Time {
	blacklisted := token.GetBlacklisted().GetFlag()
	if blacklisted {
		expiration := token.GetExpiration()
		if expiration == 0 {
			return time.Time{}
		}
		return time.Unix(expiration, 0)
	}

	if t.expiration == 0 {
		return time.Time{}
	}
	return time.Now().Add(t.expiration)
}

func (t *TokenCache) VerifyTrust(ctx context.Context, issuer, token string, tokenClaims jwt.Claims) error {
	ic, ok := t.cache[issuer]
	if !ok {
		t.logger.Debugf("client not set for issuer %v, trust verification skipped", issuer)
		return nil
	}
	tokenID, err := getID(tokenClaims)
	if err != nil {
		return err
	}
	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		return err
	}
	t.logger.Debugf("checking trust for issuer(%v) for token(id=%s)", issuer, tokenID)
	tf, set := ic.getValidTokenRecordOrFuture(tokenUUID)
	if set == nil {
		tv, errG := tf.Get(ctx)
		if errG != nil {
			return errG
		}
		t.logger.Debugf("token(id=%s) found in cache (blacklisted=%v, validUntil=%v)", tokenID, tv.blacklisted, tv.validUntil.Load())
		if tv.blacklisted {
			return ErrBlackListedToken
		}
		return nil
	}

	uri, err := url.Parse(ic.tokenEndpoint)
	if err != nil {
		ic.removeToken(tokenUUID)
		set(nil, err)
		return fmt.Errorf("cannot parse tokenEndpoint %v: %w", ic.tokenEndpoint, err)
	}
	query := uri.Query()
	query.Add("idFilter", tokenID)
	query.Add("includeBlacklisted", "true")
	uri.RawQuery = query.Encode()

	t.logger.Debugf("requesting token(id=%s) verification by m2m", tokenID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
	if err != nil {
		ic.removeToken(tokenUUID)
		set(nil, err)
		return fmt.Errorf("cannot create request for GET %v: %w", uri.String(), err)
	}

	req.Header.Set("Accept", "application/protojson")
	req.Header.Set("Authorization", "bearer "+token)
	resp, err := ic.client.Do(req)
	if err != nil {
		ic.removeToken(tokenUUID)
		set(nil, err)
		return fmt.Errorf("cannot send request for GET %v: %w", ic.tokenEndpoint, err)
	}
	defer func() {
		if errC := resp.Body.Close(); errC != nil {
			t.logger.Errorf("cannot close response body: %w", errC)
		}
	}()

	var gotToken pb.Token
	err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &gotToken)
	if err != nil {
		ic.removeToken(tokenUUID)
		set(nil, err)
		return err
	}

	var onExpire func(uuid.UUID)
	if t.logger.Check(log.DebugLevel) {
		onExpire = func(tid uuid.UUID) {
			t.logger.Debugf("token(id=%s) expired", tid.String())
		}
	}

	blacklisted := gotToken.GetBlacklisted().GetFlag()
	validUntil := t.getValidUntil(&gotToken)
	tr := newTokenRecord(blacklisted, validUntil, onExpire)
	t.logger.Debugf("token(id=%s) set (blacklisted=%v, validUntil=%v)", tokenID, blacklisted, validUntil)
	ic.setTokenRecord(tokenUUID, tr)
	set(tr, nil)

	if blacklisted {
		return ErrBlackListedToken
	}
	return nil
}
