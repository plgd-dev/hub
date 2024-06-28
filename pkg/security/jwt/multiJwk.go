package jwt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingClaims = errors.New("missing claims")
	ErrMissingIssuer = errors.New("missing issuer")
)

type MultiKeyCache struct {
	keysCache map[string]*KeyCache
}

func NewMultiKeyCache() *MultiKeyCache {
	return &MultiKeyCache{
		keysCache: make(map[string]*KeyCache),
	}
}

func (c *MultiKeyCache) Add(authority, url string, client *http.Client) {
	c.keysCache[strings.TrimSuffix(authority, "/")] = NewKeyCache(url, client)
}

func (c *MultiKeyCache) GetOrFetchKey(token *jwt.Token) (interface{}, error) {
	return c.GetOrFetchKeyWithContext(context.Background(), token)
}

func getIssuer(token *jwt.Token) (string, error) {
	if token == nil {
		return "", ErrMissingToken
	}
	if token.Claims == nil {
		return "", ErrMissingClaims
	}

	switch claims := token.Claims.(type) {
	case jwt.MapClaims:
		issuer, ok := claims["iss"].(string)
		if !ok {
			return "", ErrMissingIssuer
		}
		return strings.TrimSuffix(issuer, "/"), nil
	case interface{ GetIssuer() (string, error) }:
		issuer, err := claims.GetIssuer()
		if err != nil {
			return "", ErrMissingIssuer
		}
		return strings.TrimSuffix(issuer, "/"), nil
	default:
		return "", fmt.Errorf("unsupported type %T", token.Claims)
	}
}

func (c *MultiKeyCache) GetOrFetchKeyWithContext(ctx context.Context, token *jwt.Token) (interface{}, error) {
	issuer, err := getIssuer(token)
	if err != nil {
		return nil, err
	}
	keyCache, ok := c.keysCache[issuer]
	if !ok {
		return nil, fmt.Errorf("unknown issuer %v", issuer)
	}
	return keyCache.GetOrFetchKeyWithContext(ctx, token)
}
