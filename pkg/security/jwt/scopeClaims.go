package jwt

import (
	"fmt"
	"regexp"
	"time"
)

type ScopeClaims Claims

const PlgdRequiredScope = "plgd:required:scope"

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

func (c *ScopeClaims) Valid() error {
	v := Claims(*c)
	if err := v.ValidTimes(time.Now()); err != nil {
		return err
	}
	rs, ok := v[PlgdRequiredScope]
	if !ok {
		return fmt.Errorf("required scope not found")
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

	for _, scope := range v.Scope() {
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
	return fmt.Errorf("missing scopes: %+v", missingRequiredScopes)
}
