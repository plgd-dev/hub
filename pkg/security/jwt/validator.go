package jwt

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	pkgErrors "github.com/plgd-dev/hub/v2/pkg/errors"
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

func errInvalidClaimsType(t *jwt.Token, err error) error {
	return pkgErrors.NewError(fmt.Sprintf("unsupported type %T", t.Claims), ErrCannotParseToken, err)
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	if token == "" {
		return nil, ErrMissingToken
	}
	t, err := jwt.Parse(token, v.keys.GetOrFetchKey)
	if t == nil {
		return nil, pkgErrors.NewError("", ErrCannotParseToken, err)
	}
	c, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errInvalidClaimsType(t, err)
	}
	if err != nil {
		return c, err
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

	t, err := jwt.Parse(token, jwtKeyfunc)
	if err != nil {
		return nil, pkgErrors.NewError("", ErrCannotParseToken, err)
	}
	c, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errInvalidClaimsType(t, err)
	}

	return c, nil
}

func (v *Validator) ParseWithClaims(token string, claims jwt.Claims) error {
	if token == "" {
		return ErrMissingToken
	}

	_, err := jwt.ParseWithClaims(token, claims, v.keys.GetOrFetchKey)
	if err != nil {
		return pkgErrors.NewError("", ErrCannotParseToken, err)
	}
	return nil
}
