package jwt

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/golang-jwt/jwt/v5"
)

type ScopeClaims Claims

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
	v := make(ScopeClaims)
	v[PlgdRequiredScope] = scope
	return &v
}

func (c *ScopeClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return Claims(*c).GetExpirationTime()
}

func (c *ScopeClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return Claims(*c).GetIssuedAt()
}

func (c *ScopeClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return Claims(*c).GetNotBefore()
}

func (c *ScopeClaims) GetIssuer() (string, error) {
	return Claims(*c).GetIssuer()
}

func (c *ScopeClaims) GetSubject() (string, error) {
	return Claims(*c).GetSubject()
}

func (c *ScopeClaims) GetAudience() (jwt.ClaimStrings, error) {
	return Claims(*c).GetAudience()
}

func (c *ScopeClaims) GetID() (string, error) {
	return Claims(*c).GetID()
}

func (c *ScopeClaims) Validate() error {
	v := Claims(*c)
	rs, ok := v[PlgdRequiredScope]
	if !ok {
		return fmt.Errorf("%w: plgd:required:scope missing", ErrMissingRequiredScopes)
	}
	if rs == nil {
		return nil
	}
	requiredScopes, ok := rs.([]*regexp.Regexp)
	if !ok {
		return fmt.Errorf("%w: plgd:required:scope invalid", ErrMissingRequiredScopes)
	}
	if len(requiredScopes) == 0 {
		return nil
	}
	notMatched := make(map[string]bool)
	for _, req := range requiredScopes {
		notMatched[req.String()] = true
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
	return fmt.Errorf("%w: %+v missing", ErrMissingRequiredScopes, missingRequiredScopes)
}
