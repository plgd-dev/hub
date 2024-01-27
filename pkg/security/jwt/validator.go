package jwt

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type Validator struct {
	keys *KeyCache
}

var (
	ErrMissingToken     = errors.New("missing token")
	ErrCannotParseToken = errors.New("could not parse token")
)

func NewValidator(keyCache *KeyCache) *Validator {
	return &Validator{keys: keyCache}
}

func errParseToken(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotParseToken, err)
}

func errParseTokenInvalidClaimsType(t *jwt.Token) error {
	return fmt.Errorf("%w: unsupported type %T", ErrCannotParseToken, t.Claims)
}

func (v *Validator) ParseToken(token string) (*jwt.Token, error) {
	if token == "" {
		return nil, ErrMissingToken
	}
	return jwt.Parse(token, v.keys.GetOrFetchKey, jwt.WithIssuedAt())
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	t, err := v.ParseToken(token)
	if err != nil {
		return nil, errParseToken(err)
	}
	c, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errParseTokenInvalidClaimsType(t)
	}
	return c, nil
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
