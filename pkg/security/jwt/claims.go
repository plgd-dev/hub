package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	pkgErrors "github.com/plgd-dev/hub/v2/pkg/errors"
	pkgStrings "github.com/plgd-dev/hub/v2/pkg/strings"
)

type Claims jwt.MapClaims

const (
	ClaimExpirationTime = "exp"
	ClaimNotBefore      = "nbf"
	ClaimIssuedAt       = "iat"
	ClaimAudience       = "aud"
	ClaimIssuer         = "iss"
	ClaimSubject        = "sub"
	ClaimScope          = "scope"
	ClaimID             = "jti"
	ClaimEmail          = "email"
	ClaimClientID       = "client_id"
	ClaimName           = "n"
)

var (
	ErrTokenExpired      = errors.New("token is expired")
	ErrTokenNotYetIssued = errors.New("token used before issued")
	ErrTokenNotYetValid  = errors.New("token is not valid yet")
	ErrOwnerClaimInvalid = errors.New("owner claim is invalid")
)

func (c Claims) parseClaimStrings(claim string) (jwt.ClaimStrings, error) {
	return pkgStrings.ToSlice(c[claim])
}

func (c Claims) parseString(claim string) (string, error) {
	s, ok := pkgStrings.ToString(c[claim])
	if !ok {
		return "", pkgErrors.NewError(fmt.Sprintf("%s is invalid", claim), jwt.ErrInvalidType)
	}
	return s, nil
}

func (c Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.MapClaims(c).GetExpirationTime()
}

func (c Claims) GetNotBefore() (*jwt.NumericDate, error) {
	return jwt.MapClaims(c).GetNotBefore()
}

func (c Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.MapClaims(c).GetIssuedAt()
}

func (c Claims) GetAudience() (jwt.ClaimStrings, error) {
	return c.parseClaimStrings(ClaimAudience)
}

func (c Claims) GetIssuer() (string, error) {
	return jwt.MapClaims(c).GetIssuer()
}

func (c Claims) GetSubject() (string, error) {
	return jwt.MapClaims(c).GetSubject()
}

func (c Claims) GetScope() (jwt.ClaimStrings, error) {
	s, err := c.parseClaimStrings(ClaimScope)
	if err != nil {
		return nil, err
	}
	if len(s) == 1 {
		return strings.Split(s[0], " "), nil
	}
	return s, nil
}

func (c Claims) GetID() (string, error) {
	return c.parseString(ClaimID)
}

func (c Claims) GetEmail() (string, error) {
	return c.parseString(ClaimEmail)
}

func (c Claims) GetClientID() (string, error) {
	return c.parseString(ClaimClientID)
}

func (c Claims) GetName() (string, error) {
	return c.parseString(ClaimName)
}

func (c Claims) GetOwner(ownerClaim string) (string, error) {
	return c.parseString(ownerClaim)
}

func (c Claims) GetDeviceID(deviceIDClaim string) (string, error) {
	return c.parseString(deviceIDClaim)
}

func (c Claims) ValidTimes(now time.Time) error {
	exp, err := c.GetExpirationTime()
	if err != nil {
		return err
	}
	if exp != nil && now.After(exp.Time) {
		return ErrTokenExpired
	}

	iat, err := c.GetIssuedAt()
	if err != nil {
		return err
	}
	if iat != nil && now.Add(time.Minute).Before(iat.Time) {
		return ErrTokenNotYetIssued
	}

	nbf, err := c.GetNotBefore()
	if err != nil {
		return err
	}
	if nbf != nil && now.Add(time.Minute).Before(nbf.Time) {
		return ErrTokenNotYetValid
	}
	return nil
}

// Validate that ownerClaim is set and that it matches given user ID
func (c Claims) ValidateOwnerClaim(ownerClaim string, userID string) error {
	v, ok := c[ownerClaim]
	if !ok {
		return pkgErrors.NewError(fmt.Sprintf("owner claim '%v' is not present", ownerClaim), ErrOwnerClaimInvalid)
	}
	owner, ok := v.(string)
	if !ok {
		return pkgErrors.NewError(fmt.Sprintf("owner claim '%v' is present, but it has an invalid type '%T'", ownerClaim, v), ErrOwnerClaimInvalid)
	}
	if owner != userID {
		return pkgErrors.NewError(fmt.Sprintf("owner identifier from the token '%v' doesn't match userID '%v' from the device", owner, userID), ErrOwnerClaimInvalid)
	}
	return nil
}

func (c Claims) Validate() error {
	return c.ValidTimes(time.Now())
}

func ParseToken(token string) (Claims, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var claims Claims
	_, _, err := parser.ParseUnverified(token, &claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
