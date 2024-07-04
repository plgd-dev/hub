package jwt

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
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
		return nil, errors.New("missing key id in token")
	}

	c.m.Lock()
	defer c.m.Unlock()

	if c.keys == nil {
		return nil, errors.New("empty JWK cache")
	}
	if key, ok := c.keys.LookupKeyID(id); ok {
		if key.Algorithm().String() == token.Method.Alg() {
			return key, nil
		}
	}
	return nil, errors.New("could not find JWK")
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

func setKeyError(key string, err error) error {
	return fmt.Errorf("failed to set key %s: %w", key, err)
}

func CreateJwkKey(privateKey interface{}) (jwk.Key, error) {
	var alg string
	var publicKey interface{}
	var publicKeyPtr any
	switch v := privateKey.(type) {
	case *rsa.PrivateKey:
		switch v.Size() {
		case 256:
			alg = jwa.RS256.String()
		case 384:
			alg = jwa.RS384.String()
		case 512:
			alg = jwa.RS512.String()
		default:
			alg = jwa.RS256.String() // Default to RS256 if unknown size
		}
		publicKey = v.PublicKey
		publicKeyPtr = &v.PublicKey
	case *ecdsa.PrivateKey:
		switch v.Curve.Params().Name {
		case "P-256":
			alg = jwa.ES256.String()
		case "P-384":
			alg = jwa.ES384.String()
		case "P-521":
			alg = jwa.ES512.String()
		default:
			alg = jwa.ES256.String() // Default to ES256 if unknown curve
		}
		publicKey = v.PublicKey
		publicKeyPtr = &v.PublicKey
	}

	jwkKey, err := jwk.FromRaw(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwk: %w", err)
	}
	data, err := x509.MarshalPKIXPublicKey(publicKeyPtr)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal public key: %w", err)
	}

	if err = jwkKey.Set(jwk.KeyIDKey, uuid.NewSHA1(uuid.NameSpaceX500, data).String()); err != nil {
		return nil, setKeyError(jwk.KeyIDKey, err)
	}
	if err = jwkKey.Set(jwk.AlgorithmKey, alg); err != nil {
		return nil, setKeyError(jwk.AlgorithmKey, err)
	}
	return jwkKey, nil
}
