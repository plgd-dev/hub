package jwt

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	pkgErrors "github.com/plgd-dev/hub/v2/pkg/errors"
)

type ScopeClaims struct {
	Claims
}

const PlgdRequiredScope = "plgd:required:scope"

var ErrMissingRequiredScopes = errors.New("required scopes not found")

func NewScopeClaims(scope ...string) *ScopeClaims {
	requiredScopes := make([]*regexp.Regexp, 0, len(scope))
	for _, s := range scope {
		requiredScopes = append(requiredScopes, regexp.MustCompile(regexp.QuoteMeta(s)))
	}
	return NewRegexpScopeClaims(requiredScopes...)
}

func NewRegexpScopeClaims(scope ...*regexp.Regexp) *ScopeClaims {
	v := ScopeClaims{make(Claims)}
	v.Claims[PlgdRequiredScope] = scope
	return &v
}

func (c *ScopeClaims) Validate() error {
	v := c.Claims
	if err := v.ValidTimes(time.Now()); err != nil {
		return err
	}
	rs, ok := v[PlgdRequiredScope]
	if !ok {
		return pkgErrors.NewError("plgd:required:scope missing", ErrMissingRequiredScopes)
	}
	if rs == nil {
		return nil
	}
	requiredScopes := rs.([]*regexp.Regexp)
	if len(requiredScopes) == 0 {
		return nil
	}
	notMatched := make(map[string]bool)
	for _, reg := range requiredScopes {
		notMatched[reg.String()] = true
	}

	scopes, err := v.GetScope()
	if err != nil {
		return err
	}

	for _, scope := range scopes {
		for _, requiredScope := range requiredScopes {
			if requiredScope.MatchString(scope) {
				delete(notMatched, requiredScope.String())
			}
		}
	}
	if len(notMatched) == 0 {
		return nil
	}
	missingRequiredScopes := make([]string, 0, len(notMatched))
	for scope := range notMatched {
		missingRequiredScopes = append(missingRequiredScopes, scope)
	}
	return pkgErrors.NewError(fmt.Sprintf("%+v missing", missingRequiredScopes), ErrMissingRequiredScopes)
}
