package jwt

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

type Validator struct {
	keys *KeyCache
}

var errMissingToken = fmt.Errorf("missing token")

func NewValidator(keyCache *KeyCache) *Validator {
	return &Validator{keys: keyCache}
}

func errParseToken(err error) error {
	return fmt.Errorf("could not parse token: %w", err)
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	if token == "" {
		return nil, errMissingToken
	}
	t, err := jwt.Parse(token, v.keys.GetOrFetchKey)
	if t == nil {
		return nil, errParseToken(err)
	}
	c := t.Claims.(jwt.MapClaims)
	if err != nil {
		return c, errParseToken(err)
	}
	return c, nil
}

func (v *Validator) ParseWithContext(ctx context.Context, token string) (jwt.MapClaims, error) {
	if token == "" {
		return nil, errMissingToken
	}

	jwtKeyfunc := func(token *jwt.Token) (interface{}, error) {
		return v.keys.GetOrFetchKeyWithContext(ctx, token)
	}

	t, err := jwt.Parse(token, jwtKeyfunc)
	if err != nil {
		return nil, errParseToken(err)
	}
	c, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unsupported type %T", t.Claims)
	}

	return c, nil
}

func (v *Validator) ParseWithClaims(token string, claims jwt.Claims) error {
	if token == "" {
		return errMissingToken
	}

	_, err := jwt.ParseWithClaims(token, claims, v.keys.GetOrFetchKey)
	if err != nil {
		return errParseToken(err)
	}
	return nil
}
