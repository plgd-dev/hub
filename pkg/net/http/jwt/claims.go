package jwt

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
)

type DeniedClaimsError struct {
	jwt.MapClaims
	Err error
}

func (c DeniedClaimsError) Validate() error {
	return c.Err
}

func (c DeniedClaimsError) Error() string {
	return c.Err.Error()
}

func (c DeniedClaimsError) Unwrap() error {
	return c.Err
}

func MakeClaimsFunc(methods map[string][]AuthArgs) ClaimsFunc {
	return func(_ context.Context, method, uri string) jwt.ClaimsValidator {
		args, ok := methods[method]
		if !ok {
			return &DeniedClaimsError{Err: fmt.Errorf("inaccessible method: %v", method)}
		}
		for _, arg := range args {
			if arg.URI.MatchString(uri) {
				return pkgJwt.NewRegexpScopeClaims(arg.Scopes...)
			}
		}
		return &DeniedClaimsError{Err: fmt.Errorf("inaccessible uri: %v %v", method, uri)}
	}
}
