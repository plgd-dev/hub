package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/cloud/v2/pkg/strings"
	pkgTime "github.com/plgd-dev/cloud/v2/pkg/time"
)

type Claims jwt.MapClaims

const (
	ClaimExpiresAt = "exp"
	ClaimScope     = "scope"
	ClaimAudience  = "aud"
	ClaimId        = "jti"
	ClaimIssuer    = "iss"
	ClaimSubject   = "sub"
	ClaimIssuedAt  = "iat"
	ClaimNotBefore = "nbf"
	ClaimClientID  = "client_id"
	ClaimEmail     = "email"
	ClaimName      = "n"
)

func toNum(v interface{}) (time.Time, error) {
	switch val := v.(type) {
	case int64:
		return pkgTime.Unix(val, 0), nil
	case float64:
		return pkgTime.Unix(int64(val), 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid type '%T'", val)
	}
}

/// Get expiration time (exp) from user info map.
/// It might not be set, in that case zero time and no error are returned.
func (u Claims) ExpiresAt() (time.Time, error) {
	const expKey = ClaimExpiresAt
	v, ok := u[expKey]
	if !ok {
		return time.Time{}, nil
	}

	exp, err := toNum(v)
	if err != nil {
		return time.Time{}, fmt.Errorf("expiration claim('%v') is present, but it has an %w", v, err)
	}
	return exp, nil
}

/// Validate that ownerClaim is set and that it matches given user ID
func (u Claims) ValidateOwnerClaim(ownerClaim string, userID string) error {
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

func (c Claims) Scope() []string {
	return strings.ToSlice(c[ClaimScope])
}

func (c Claims) Owner(ownerClaim string) string {
	s, _ := strings.ToString(c[ownerClaim])
	return s
}

func (c Claims) Email() string {
	s, _ := strings.ToString(c[ClaimEmail])
	return s
}

func (c Claims) ClientID() string {
	s, _ := strings.ToString(c[ClaimClientID])
	return s
}

func (c Claims) Name() string {
	s, _ := strings.ToString(c[ClaimName])
	return s
}

func (c Claims) DeviceID(deviceIDClaim string) string {
	s, _ := strings.ToString(c[deviceIDClaim])
	return s
}

func (c Claims) Audience() []string {
	return strings.ToSlice(c[ClaimAudience])
}

func (c Claims) ID() string {
	s, _ := strings.ToString(c[ClaimId])
	return s
}

func (c Claims) Issuer() string {
	s, _ := strings.ToString(c[ClaimIssuer])
	return s
}

func (c Claims) Subject() string {
	s, _ := strings.ToString(c[ClaimSubject])
	return s
}

func (c Claims) IssuedAt() (time.Time, error) {
	const expKey = ClaimIssuedAt
	v, ok := c[expKey]
	if !ok {
		return time.Time{}, nil
	}
	iat, err := toNum(v)
	if err != nil {
		return time.Time{}, fmt.Errorf("issued at claim('%v') is present, but it has an %w", v, err)
	}
	return iat, nil
}

func (c Claims) NotBefore() (time.Time, error) {
	const expKey = ClaimNotBefore
	v, ok := c[expKey]
	if !ok {
		return time.Time{}, nil
	}

	nbf, err := toNum(v)
	if err != nil {
		return time.Time{}, fmt.Errorf("not before at claim('%v') is present, but it has an %w", v, err)
	}
	return nbf, nil
}

func (c Claims) ValidTimes(now time.Time) error {
	exp, err := c.ExpiresAt()
	if err != nil {
		return err
	}
	if !exp.IsZero() && now.After(exp) {
		return fmt.Errorf("token is expired")
	}
	iat, err := c.IssuedAt()
	if err != nil {
		return err
	}
	if !iat.IsZero() && now.Before(iat) {
		return fmt.Errorf("token used before issued")
	}
	nbf, err := c.NotBefore()
	if err != nil {
		return err
	}
	if !nbf.IsZero() && now.Before(nbf) {
		return fmt.Errorf("token is not valid yet")
	}
	return nil
}

func (c Claims) Valid() error {
	return c.ValidTimes(time.Now())
}

func ParseToken(token string) (Claims, error) {
	parser := &jwt.Parser{
		SkipClaimsValidation: true,
	}

	var claims Claims
	_, _, err := parser.ParseUnverified(token, &claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
