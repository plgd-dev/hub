package jwt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
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
	// tokenCache  TokenCache
}

var (
	ErrMissingToken      = errors.New("missing token")
	ErrCannotParseToken  = errors.New("could not parse token")
	ErrCannotVerifyTrust = errors.New("could not verify token trust")
	ErrBlackListedToken  = errors.New("token is blacklisted")
)

type config struct {
	clients     map[string]*Client
	verifyTrust bool
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithTrustVerification(clients map[string]*Client) Option {
	return optionFunc(func(c *config) {
		c.verifyTrust = true
		c.clients = clients
	})
}

func NewValidator(keyCache KeyCacheI, logger log.Logger, opts ...Option) *Validator {
	c := &config{}
	for _, opt := range opts {
		opt.apply(c)
	}
	return &Validator{
		keys:   keyCache,
		logger: logger,
		// tokenCache:  make(TokenCache),
		clients:     c.clients,
		verifyTrust: c.verifyTrust,
	}
}

func errParseToken(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotParseToken, err)
}

func errParseTokenInvalidClaimsType(t *jwt.Token) error {
	return fmt.Errorf("%w: unsupported type %T", ErrCannotParseToken, t.Claims)
}

func (v *Validator) checkTrust(ctx context.Context, claims jwt.Claims) error {
	issuer, err := claims.GetIssuer()
	if err != nil {
		return err
	}
	issuer = pkgHttpUri.CanonicalURI(issuer)

	client, ok := v.clients[issuer]
	if !ok {
		// TODO: set to debug
		v.logger.Infof("client not set for issuer %v, trust verification skipped", issuer)
		return nil
	}

	tokenID, err := getID(claims)
	if err != nil {
		return err
	}

	// tr, ok := v.tokenCache.Get(tokenID, issuer)
	// if ok {
	// 	// TODO: check expiration
	// 	if tr.Blacklisted {
	// 		return ErrBlackListedToken
	// 	}
	// 	return nil
	// }

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

	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return fmt.Errorf("cannot get token from context: %w", err)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected statusCode %v", resp.StatusCode)
	}

	var gotToken pb.Token
	err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &gotToken)
	if err != nil {
		return err
	}

	// tr = TokenRecord{
	// 	Expiration:  gotToken.GetExpiration(),
	// 	Blacklisted: gotToken.GetBlacklisted().GetFlag(),
	// }
	// v.tokenCache.Add(tokenID, issuer, tr)
	// if tr.Blacklisted {
	// 	return ErrBlackListedToken
	// }
	if gotToken.GetBlacklisted().GetFlag() {
		return ErrBlackListedToken
	}

	return nil
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	return v.ParseWithContext(context.Background(), token)
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
	c, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errParseTokenInvalidClaimsType(t)
	}
	if v.verifyTrust {
		if err = v.checkTrust(ctx, c); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCannotVerifyTrust, err)
		}
	}
	return c, nil
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
		if err = v.checkTrust(ctx, claims); err != nil {
			return fmt.Errorf("%w: %w", ErrCannotVerifyTrust, err)
		}
	}

	return nil
}
