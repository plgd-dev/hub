package jwt

import (
	"fmt"
	"regexp"
	"time"
)

type ScopeClaims struct {
	Claims
	requiredScopes []*regexp.Regexp
}

func NewScopeClaims(scope ...string) *ScopeClaims {
	requiredScopes := make([]*regexp.Regexp, 0, len(scope))
	for _, s := range scope {
		requiredScopes = append(requiredScopes, regexp.MustCompile(regexp.QuoteMeta(s)))
	}
	return NewRegexpScopeClaims(requiredScopes...)
}

func NewRegexpScopeClaims(scope ...*regexp.Regexp) *ScopeClaims {
	return &ScopeClaims{requiredScopes: scope}
}

func (c *ScopeClaims) Valid() error {
	if err := c.Claims.ValidTimes(time.Now()); err != nil {
		return err
	}
	if len(c.requiredScopes) == 0 {
		return nil
	}
	notMatched := make(map[string]bool)
	for _, reg := range c.requiredScopes {
		notMatched[reg.String()] = true
	}
	for _, scope := range c.Scope() {
		for _, requiredScope := range c.requiredScopes {
			if requiredScope.MatchString(scope) {
				delete(notMatched, requiredScope.String())
			}
		}
	}
	if len(notMatched) == 0 {
		return nil
	}
	requiredScopes := make([]string, 0, len(notMatched))
	for scope := range notMatched {
		requiredScopes = append(requiredScopes, scope)
	}
	return fmt.Errorf("missing scopes: %+v", requiredScopes)
}
