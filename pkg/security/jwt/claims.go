package jwt

import (
	"fmt"
	"time"

	"github.com/plgd-dev/kit/strings"
)

type Claims struct {
	ClientID string      `json:"client_id"`
	Email    string      `json:"email"`
	Scope    interface{} `json:"scope"`
	StandardClaims
}

func (c Claims) GetScope() []string {
	return strings.ToSlice(c.Scope)
}

// https://tools.ietf.org/html/rfc7519#section-4.1
type StandardClaims struct {
	Audience  interface{} `json:"aud,omitempty"`
	ExpiresAt int64       `json:"exp,omitempty"`
	Id        string      `json:"jti,omitempty"`
	IssuedAt  int64       `json:"iat,omitempty"`
	Issuer    string      `json:"iss,omitempty"`
	NotBefore int64       `json:"nbf,omitempty"`
	Subject   string      `json:"sub,omitempty"`
}

func (c StandardClaims) GetAudience() []string {
	return strings.ToSlice(c.Audience)
}

func (c StandardClaims) Valid() error {
	now := timeFunc().Unix()
	if now > c.ExpiresAt {
		return fmt.Errorf("token is expired")
	}
	if now < c.IssuedAt {
		return fmt.Errorf("token used before issued")
	}
	if now < c.NotBefore {
		return fmt.Errorf("token is not valid yet")
	}
	return nil
}

var timeFunc = time.Now

func SetTimeFunc(f func() time.Time) (restore func()) {
	prev := timeFunc
	timeFunc = f
	return func() { timeFunc = prev }
}
