package jwt

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type KeyCache struct {
	url  string
	http *http.Client
	m    sync.Mutex
	keys jwk.Set
}

func NewKeyCache(url string, client *http.Client) *KeyCache {
	return &KeyCache{url: url, http: client}
}

func (c *KeyCache) GetOrFetchKeyWithContext(ctx context.Context, token *jwt.Token) (interface{}, error) {
	if k, err := c.GetKey(token); err == nil {
		return k, nil
	}
	if err := c.FetchKeysWithContext(ctx); err != nil {
		return nil, err
	}
	return c.GetKey(token)
}

func (c *KeyCache) GetOrFetchKey(token *jwt.Token) (interface{}, error) {
	if k, err := c.GetKey(token); err == nil {
		return k, nil
	}
	if err := c.FetchKeys(); err != nil {
		return nil, err
	}
	return c.GetKey(token)
}

func (c *KeyCache) GetKey(token *jwt.Token) (interface{}, error) {
	key, err := c.LookupKey(token)
	if err != nil {
		return nil, err
	}
	var v interface{}
	return v, key.Raw(&v)
}

func (c *KeyCache) LookupKey(token *jwt.Token) (jwk.Key, error) {
	id, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("missing key id in token")
	}

	c.m.Lock()
	defer c.m.Unlock()

	if c.keys == nil {
		return nil, fmt.Errorf("empty JWK cache")
	}
	if key, ok := c.keys.LookupKeyID(id); ok {
		if key.Algorithm().String() == token.Method.Alg() {
			return key, nil
		}
	}
	return nil, fmt.Errorf("could not find JWK")
}

func (c *KeyCache) FetchKeys() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.http.Timeout)
	defer cancel()

	return c.FetchKeysWithContext(ctx)
}

func (c *KeyCache) FetchKeysWithContext(ctx context.Context) error {
	keys, err := jwk.Fetch(ctx, c.url, jwk.WithHTTPClient(c.http))
	if err != nil {
		return fmt.Errorf("could not fetch JWK: %w", err)
	}

	c.m.Lock()
	defer c.m.Unlock()

	c.keys = keys
	return nil
}
