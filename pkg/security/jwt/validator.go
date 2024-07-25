package jwt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
)

type KeyCacheI interface {
	GetOrFetchKey(token *jwt.Token) (interface{}, error)
	GetOrFetchKeyWithContext(ctx context.Context, token *jwt.Token) (interface{}, error)
}

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

type Validator struct {
	keys        KeyCacheI
	logger      log.Logger
	clients     map[string]*Client
	verifyTrust bool
	tokenCache  *TokenCache
}

var (
	ErrMissingToken      = errors.New("missing token")
	ErrCannotParseToken  = errors.New("could not parse token")
	ErrCannotVerifyTrust = errors.New("could not verify token trust")
	ErrBlackListedToken  = errors.New("token is blacklisted")
)

type config struct {
	verifyTrust     bool
	clients         map[string]*Client
	cacheExpiration time.Duration
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithTrustVerification(clients map[string]*Client, cacheExpiration time.Duration) Option {
	return optionFunc(func(c *config) {
		c.verifyTrust = true
		c.clients = clients
		c.cacheExpiration = cacheExpiration
	})
}

func NewValidator(keyCache KeyCacheI, logger log.Logger, opts ...Option) *Validator {
	c := &config{}
	for _, opt := range opts {
		opt.apply(c)
	}
	v := &Validator{
		keys:        keyCache,
		logger:      logger,
		clients:     c.clients,
		verifyTrust: c.verifyTrust,
	}
	if c.verifyTrust {
		v.tokenCache = NewTokenCache(c.cacheExpiration)
	}
	return v
}

func errParseToken(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotParseToken, err)
}

func errParseTokenInvalidClaimsType(t *jwt.Token) error {
	return fmt.Errorf("%w: unsupported type %T", ErrCannotParseToken, t.Claims)
}

func (v *Validator) checkTrust(ctx context.Context, token string, claims jwt.Claims) error {
	issuer, err := claims.GetIssuer()
	if err != nil {
		return err
	}
	issuer = pkgHttpUri.CanonicalURI(issuer)

	client, ok := v.clients[issuer]
	if !ok {
		v.logger.Debugf("client not set for issuer %v, trust verification skipped", issuer)
		return nil
	}

	tokenID, err := getID(claims)
	if err != nil {
		return err
	}

	tr, ok := v.tokenCache.Load(tokenID, issuer)
	if ok {
		if tr.Blacklisted {
			return ErrBlackListedToken
		}
		return nil
	}

	uri, err := url.Parse(client.tokenEndpoint)
	if err != nil {
		return fmt.Errorf("cannot parse tokenEndpoint %v: %w", client.tokenEndpoint, err)
	}
	query := uri.Query()
	query.Add("idFilter", tokenID)
	query.Add("includeBlacklisted", "true")
	uri.RawQuery = query.Encode()

	v.logger.Infof("checking trust for issuer %v for token(id=%s)", issuer, tokenID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
	if err != nil {
		return fmt.Errorf("cannot create request for GET %v: %w", uri.String(), err)
	}

	req.Header.Set("Accept", "application/protojson")
	req.Header.Set("Authorization", "bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send request for GET %v: %w", client.tokenEndpoint, err)
	}
	defer func() {
		if errC := resp.Body.Close(); errC != nil {
			v.logger.Errorf("cannot close response body: %w", errC)
		}
	}()

	var gotToken pb.Token
	err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &gotToken)
	if err != nil {
		return err
	}

	tr = TokenRecord{
		Blacklisted: gotToken.GetBlacklisted().GetFlag(),
	}
	if tr.Blacklisted {
		v.tokenCache.AddBlacklisted(tokenID, issuer, pkgTime.Unix(gotToken.GetExpiration(), 0))
		return ErrBlackListedToken
	}
	//// TODO: configure expiration interval -> for now use 10s
	if !v.tokenCache.AddWhitelisted(tokenID, issuer) {
		return ErrBlackListedToken
	}
	return nil
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	return v.ParseWithContext(context.Background(), token)
}

func errVerifyTrust(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotVerifyTrust, err)
}

func (v *Validator) ParseWithContext(ctx context.Context, token string) (jwt.MapClaims, error) {
	if token == "" {
		return nil, ErrMissingToken
	}

	jwtKeyfunc := func(token *jwt.Token) (interface{}, error) {
		return v.keys.GetOrFetchKeyWithContext(ctx, token)
	}

	t, err := jwt.Parse(token, jwtKeyfunc, jwt.WithIssuedAt())
	if err != nil {
		return nil, errParseToken(err)
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errParseTokenInvalidClaimsType(t)
	}
	if v.verifyTrust {
		if err = v.checkTrust(ctx, token, claims); err != nil {
			return nil, errVerifyTrust(err)
		}
	}
	return claims, nil
}

func (v *Validator) ParseWithClaims(ctx context.Context, token string, claims jwt.Claims) error {
	if token == "" {
		return ErrMissingToken
	}

	_, err := jwt.ParseWithClaims(token, claims, v.keys.GetOrFetchKey, jwt.WithIssuedAt())
	if err != nil {
		return errParseToken(err)
	}
	if v.verifyTrust {
		if err = v.checkTrust(ctx, token, claims); err != nil {
			return errVerifyTrust(err)
		}
	}
	return nil
}
