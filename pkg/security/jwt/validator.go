package jwt

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

type KeyCacheI interface {
	GetOrFetchKey(token *jwt.Token) (interface{}, error)
	GetOrFetchKeyWithContext(ctx context.Context, token *jwt.Token) (interface{}, error)
}

type Validator struct {
	keys   KeyCacheI
	logger log.Logger
}

var (
	ErrMissingToken     = errors.New("missing token")
	ErrCannotParseToken = errors.New("could not parse token")
)

func NewValidator(keyCache KeyCacheI, logger log.Logger) *Validator {
	return &Validator{
		keys:   keyCache,
		logger: logger,
	}
}

func errParseToken(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotParseToken, err)
}

func errParseTokenInvalidClaimsType(t *jwt.Token) error {
	return fmt.Errorf("%w: unsupported type %T", ErrCannotParseToken, t.Claims)
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
	return c, nil
}

func (v *Validator) ParseWithClaims(token string, claims jwt.Claims) error {
	if token == "" {
		return ErrMissingToken
	}

	_, err := jwt.ParseWithClaims(token, claims, v.keys.GetOrFetchKey, jwt.WithIssuedAt())
	if err != nil {
		return errParseToken(err)
	}
	return nil
}
