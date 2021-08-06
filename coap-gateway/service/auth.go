package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/go-coap/v2/message/codes"
)

type Interceptor = func(ctx context.Context, code codes.Code, path string) (context.Context, error)

func NewAuthInterceptor() Interceptor {
	return func(ctx context.Context, code codes.Code, path string) (context.Context, error) {
		switch path {
		case uri.RefreshToken, uri.SignUp, uri.SignIn:
			return ctx, nil
		}
		e := ctx.Value(&authCtxKey)
		if e == nil {
			return ctx, fmt.Errorf("invalid authorization context")
		}
		expire := e.(*authorizationContext)
		return ctx, expire.IsValid()
	}
}

type jwtClaims jwt.MapClaims

func (s *Service) ValidateToken(ctx context.Context, token string) (jwtClaims, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	m, err := s.jwtValidator.ParseWithContext(ctx, token)
	if err != nil {
		return nil, err
	}
	return jwtClaims(m), nil
}

/// Get expiration time (exp) from user info map.
/// It might not be set, in that case zero time and no error are returned.
func (u jwtClaims) getExpirationTime() (time.Time, error) {
	const expKey = "exp"
	v, ok := u[expKey]
	if !ok {
		return time.Time{}, nil
	}

	exp, ok := v.(float64) // all integers are float64 in json
	if !ok {
		return time.Time{}, fmt.Errorf("expiration claim '%v' is present, but it has an invalid type '%T'", expKey, v)
	}
	return pkgTime.Unix(int64(exp), 0), nil
}

/// Validate that ownerClaim is set and that it matches given user ID
func (u jwtClaims) validateOwnerClaim(ownerClaim string, userID string) error {
	v, ok := u[ownerClaim]
	if !ok {
		return fmt.Errorf("owner claim '%v' is not present", ownerClaim)
	}
	owner, ok := v.(string)
	if !ok {
		return fmt.Errorf("owner claim '%v' is present, but it has an invalid type '%T'", ownerClaim, v)
	}
	if owner != userID {
		return fmt.Errorf("owner identifier from the token '%v' doesn't match userID '%v' from the device", owner, userID)
	}
	return nil
}
