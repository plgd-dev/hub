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

type HTTPClient struct {
	*http.Client
	tokenEndpoint string
}

func (c *HTTPClient) VerifyTokenByRequest(ctx context.Context, token, tokenID string) (*pb.Token, error) {
	uri, err := url.Parse(c.tokenEndpoint)
	if err != nil {
		return nil, fmt.Errorf("cannot parse tokenEndpoint %v: %w", c.tokenEndpoint, err)
	}
	query := uri.Query()
	query.Add("idFilter", tokenID)
	query.Add("includeBlacklisted", "true")
	uri.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request for GET %v: %w", uri.String(), err)
	}

	req.Header.Set("Accept", "application/protojson")
	req.Header.Set("Authorization", "bearer "+token)
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot send request for GET %v: %w", c.tokenEndpoint, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var gotToken pb.Token
	err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &gotToken)
	if err != nil {
		return nil, err
	}
	return &gotToken, nil
}

func NewHTTPClient(client *http.Client, tokenEndpoint string) *HTTPClient {
	return &HTTPClient{
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
	client TokenIssuerClient
	tokens map[uuid.UUID]tokenOrFuture
	mutex  sync.Mutex
}

type TokenIssuerClient interface {
	VerifyTokenByRequest(ctx context.Context, token, tokenID string) (*pb.Token, error)
}

func newTokenIssuerCache(client TokenIssuerClient) *tokenIssuerCache {
	return &tokenIssuerCache{
		client: client,
		tokens: make(map[uuid.UUID]tokenOrFuture),
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

func (tc *tokenIssuerCache) removeTokenRecord(tokenID uuid.UUID) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	delete(tc.tokens, tokenID)
}

func (tc *tokenIssuerCache) removeTokenRecordAndSetErrorOnFuture(tokenUUID uuid.UUID, setTRFuture future.SetFunc, err error) {
	tc.removeTokenRecord(tokenUUID)
	setTRFuture(nil, err)
}

func (tc *tokenIssuerCache) setTokenRecord(tokenUUID uuid.UUID, tr *tokenRecord) {
	tf := makeTokenOrFuture(tr, nil)
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.tokens[tokenUUID] = tf
}

func (tc *tokenIssuerCache) setTokenRecordAndWaitingFuture(tokenUUID uuid.UUID, tr *tokenRecord, setTRFuture future.SetFunc) {
	tc.setTokenRecord(tokenUUID, tr)
	setTRFuture(tr, nil)
}

func (tc *tokenIssuerCache) checkExpirations(now time.Time) {
	expired := make(map[uuid.UUID]*tokenRecord, 8)
	tc.mutex.Lock()
	for tokenID, tf := range tc.tokens {
		if tr, ok := tf.tokenOrFuture.(*tokenRecord); ok && tr.IsExpired(now) {
			if tr.onExpire != nil {
				expired[tokenID] = tr
			}
			delete(tc.tokens, tokenID)
		}
	}
	tc.mutex.Unlock()
	for tokenID, tr := range expired {
		tr.onExpire(tokenID)
	}
}

func (tc *tokenIssuerCache) verifyTokenByRequest(ctx context.Context, token, tokenID string) (*pb.Token, error) {
	return tc.client.VerifyTokenByRequest(ctx, token, tokenID)
}

type TokenCache struct {
	expiration time.Duration
	cache      map[string]*tokenIssuerCache
	logger     log.Logger
}

func NewTokenCache(clients map[string]TokenIssuerClient, expiration time.Duration, logger log.Logger) *TokenCache {
	tc := &TokenCache{
		expiration: expiration,
		logger:     logger,
	}
	tc.cache = make(map[string]*tokenIssuerCache)
	for issuer, client := range clients {
		tc.cache[issuer] = newTokenIssuerCache(client)
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

func getTokenUUID(tokenClaims jwt.Claims) (string, uuid.UUID, error) {
	tokenID, err := getID(tokenClaims)
	if err != nil {
		return "", uuid.Nil, err
	}
	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		return "", uuid.Nil, err
	}
	return tokenID, tokenUUID, nil
}

func (t *TokenCache) VerifyTrust(ctx context.Context, issuer, token string, tokenClaims jwt.Claims) error {
	tc, ok := t.cache[issuer]
	if !ok {
		t.logger.Debugf("client not set for issuer %v, trust verification skipped", issuer)
		return nil
	}

	tokenID, tokenUUID, err := getTokenUUID(tokenClaims)
	if err != nil {
		return err
	}
	t.logger.Debugf("checking trust for issuer(%v) for token(id=%s)", issuer, tokenID)
	tf, set := tc.getValidTokenRecordOrFuture(tokenUUID)
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

	t.logger.Debugf("requesting token(id=%s) verification by m2m", tokenID)
	respToken, err := tc.verifyTokenByRequest(ctx, token, tokenID)
	if err != nil {
		tc.removeTokenRecordAndSetErrorOnFuture(tokenUUID, set, err)
		return err
	}

	var onExpire func(uuid.UUID)
	if t.logger.Check(log.DebugLevel) {
		onExpire = func(tid uuid.UUID) {
			t.logger.Debugf("token(id=%s) expired", tid.String())
		}
	}

	blacklisted := respToken.GetBlacklisted().GetFlag()
	validUntil := t.getValidUntil(respToken)
	tr := newTokenRecord(blacklisted, validUntil, onExpire)
	t.logger.Debugf("token(id=%s) set (blacklisted=%v, validUntil=%v)", tokenID, blacklisted, validUntil)
	tc.setTokenRecordAndWaitingFuture(tokenUUID, tr, set)

	if blacklisted {
		return ErrBlackListedToken
	}
	return nil
}

func (t *TokenCache) CheckExpirations(now time.Time) {
	for _, ic := range t.cache {
		ic.checkExpirations(now)
	}
}
