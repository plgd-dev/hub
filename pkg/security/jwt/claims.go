package jwt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
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
	ClaimName           = "name"
)

var ErrOwnerClaimInvalid = errors.New("owner claim is invalid")

// parseClaimsString tries to parse a key in the map claims type as a
// [ClaimsStrings] type, which can either be a string or an array of string.
func (c Claims) parseClaimStrings(claim string) (jwt.ClaimStrings, error) {
	return pkgStrings.ToSlice(c[claim])
}

// parseString tries to parse a key in the map claims type as a [string] type.
// If the key does not exist, an empty string is returned. If the key has the
// wrong type, an error is returned.
func (c Claims) parseString(claim string) (string, error) {
	cc, ok := c[claim]
	if !ok {
		return "", nil
	}
	s, ok := pkgStrings.ToString(cc)
	if !ok {
		return "", fmt.Errorf("%s is invalid: %w", claim, jwt.ErrInvalidType)
	}
	return s, nil
}

// GetExpirationTime returns the Expiration Time ("exp") claim. If the claim does not exist,
// nil is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.MapClaims(c).GetExpirationTime()
}

// GetNotBefore returns the Not Before ("nbf") claim. If the claim does not exist,
// nil is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetNotBefore() (*jwt.NumericDate, error) {
	return jwt.MapClaims(c).GetNotBefore()
}

// GetIssuedAt returns the Issued At ("iat") claim. If the claim does not exist,
// nil is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.MapClaims(c).GetIssuedAt()
}

// GetAudience returns the Audience ("aud") claim. If the claim does not exist,
// nil is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetAudience() (jwt.ClaimStrings, error) {
	return c.parseClaimStrings(ClaimAudience)
}

// GetIssuer returns the Issuer ("iss") claim. If the claim does not exist,
// nil is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetIssuer() (string, error) {
	return jwt.MapClaims(c).GetIssuer()
}

// GetSubject returns the Subject ("sub") claim. If the claim does not exist,
// nil is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetSubject() (string, error) {
	return jwt.MapClaims(c).GetSubject()
}

// GetScope returns the Scope ("scope") claim. If the claim does not exist,
// nil is returned. If the claim has the wrong type, an error is returned.
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

// GetID returns the ID ("jti") claim. If the claim does not exist,
// an empty string is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetID() (string, error) {
	return c.parseString(ClaimID)
}

// GetEmail returns the Email ("email") claim. If the claim does not exist,
// an empty string is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetEmail() (string, error) {
	return c.parseString(ClaimEmail)
}

// GetClientID returns the ClientID ("client_id") claim. If the claim does not exist,
// an empty string is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetClientID() (string, error) {
	return c.parseString(ClaimClientID)
}

// GetName returns the Name ("n") claim. If the claim does not exist,
// an empty string is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetName() (string, error) {
	return c.parseString(ClaimName)
}

// GetOwner returns the Owner ("owner") claim. If the claim does not exist,
// an empty string is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetOwner(ownerClaim string) (string, error) {
	return c.parseString(ownerClaim)
}

// GetDeviceID returns the DeviceID ("device_id") claim. If the claim does not exist,
// an empty string is returned. If the claim has the wrong type, an error is returned.
func (c Claims) GetDeviceID(deviceIDClaim string) (string, error) {
	return c.parseString(deviceIDClaim)
}

// ValidateOwnerClaim validates that ownerClaim is set and that it matches given user ID
func (c Claims) ValidateOwnerClaim(ownerClaim string, userID string) error {
	v, ok := c[ownerClaim]
	if !ok {
		return fmt.Errorf("%w: owner claim '%v' is not present", ErrOwnerClaimInvalid, ownerClaim)
	}
	owner, ok := v.(string)
	if !ok {
		return fmt.Errorf("%w: owner claim '%v' is present, but it has an invalid type '%T'", ErrOwnerClaimInvalid, ownerClaim, v)
	}
	if owner != userID {
		return fmt.Errorf("%w: owner identifier from the token '%v' doesn't match userID '%v' from the device", ErrOwnerClaimInvalid, owner, userID)
	}
	return nil
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

func getIssuer(token *jwt.Token) (string, error) {
	if token == nil {
		return "", ErrMissingToken
	}
	if token.Claims == nil {
		return "", ErrMissingClaims
	}

	switch claims := token.Claims.(type) {
	case interface{ GetIssuer() (string, error) }:
		issuer, err := claims.GetIssuer()
		if err != nil {
			return "", ErrMissingIssuer
		}
		return pkgHttpUri.CanonicalURI(issuer), nil
	default:
		return "", fmt.Errorf("unsupported type %T", token.Claims)
	}
}

func getID(claims jwt.Claims) (string, error) {
	switch c := claims.(type) {
	case interface{ GetID() (string, error) }:
		id, err := c.GetID()
		if err != nil {
			return "", ErrMissingID
		}
		return id, nil
	case jwt.MapClaims:
		id, ok := c[ClaimID].(string)
		if !ok {
			return "", ErrMissingID
		}
		return id, nil
	default:
		return "", fmt.Errorf("unsupported type %T", claims)
	}
}
