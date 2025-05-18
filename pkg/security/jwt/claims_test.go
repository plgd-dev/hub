package jwt_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

var now = time.Now()

func makeClaims() pkgJwt.Claims {
	return pkgJwt.Claims{
		pkgJwt.ClaimExpirationTime: float64(now.Add(time.Hour).Unix()),
		pkgJwt.ClaimNotBefore:      float64(now.Unix()),
		pkgJwt.ClaimIssuedAt:       float64(now.Unix()),
		pkgJwt.ClaimAudience:       []string{"testAudience", "testAudience2"},
		pkgJwt.ClaimIssuer:         "testIssuer",
		pkgJwt.ClaimSubject:        "testSubject",
		pkgJwt.ClaimScope:          []string{"testScope"},
		pkgJwt.ClaimID:             "testID",
		pkgJwt.ClaimEmail:          "testEmail",
		pkgJwt.ClaimClientID:       "testClientID",
	}
}

func toTime(t *testing.T, v interface{}) time.Time {
	switch val := v.(type) {
	case int64:
		return pkgTime.Unix(val, 0)
	case float64:
		return pkgTime.Unix(int64(val), 0)
	}
	t.Errorf("invalid type '%T'", v)
	return time.Time{}
}

func TestEmptyExpirationTime(t *testing.T) {
	c := pkgJwt.Claims{}
	exp, err := c.GetExpirationTime()
	require.NoError(t, err)
	require.Nil(t, exp)
}

func TestInvalidExpirationTime(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimExpirationTime: "string",
	}
	_, err := c.GetExpirationTime()
	require.Error(t, err)
}

func TestExpirationTime(t *testing.T) {
	c := makeClaims()
	exp, err := c.GetExpirationTime()
	require.NoError(t, err)
	require.Equal(t, exp.Time, toTime(t, c[pkgJwt.ClaimExpirationTime]))
}

func TestEmptyNotBefore(t *testing.T) {
	c := pkgJwt.Claims{}
	exp, err := c.GetNotBefore()
	require.NoError(t, err)
	require.Nil(t, exp)
}

func TestInvalidNotBefore(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimNotBefore: "string",
	}
	_, err := c.GetNotBefore()
	require.Error(t, err)
}

func TestNotBefore(t *testing.T) {
	c := makeClaims()
	nbf, err := c.GetNotBefore()
	require.NoError(t, err)
	require.Equal(t, nbf.Time, toTime(t, c[pkgJwt.ClaimNotBefore]))
}

func TestEmptyIssuedAt(t *testing.T) {
	c := pkgJwt.Claims{}
	exp, err := c.GetIssuedAt()
	require.NoError(t, err)
	require.Nil(t, exp)
}

func TestInvalidIssuedAt(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimIssuedAt: "string",
	}
	_, err := c.GetIssuedAt()
	require.Error(t, err)
}

func TestIssuedAt(t *testing.T) {
	c := makeClaims()
	iat, err := c.GetIssuedAt()
	require.NoError(t, err)
	require.Equal(t, iat.Time, toTime(t, c[pkgJwt.ClaimIssuedAt]))
}

func TestEmptyAudience(t *testing.T) {
	c := pkgJwt.Claims{}
	aud, err := c.GetAudience()
	require.NoError(t, err)
	require.Empty(t, aud)
}

func TestInvalidAudience(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimAudience: []interface{}{123},
	}
	_, err := c.GetAudience()
	require.Error(t, err)
}

func TestAudience(t *testing.T) {
	c := makeClaims()
	aud, err := c.GetAudience()
	require.NoError(t, err)
	require.Equal(t, []string(aud), c[pkgJwt.ClaimAudience])
}

func TestAudienceOfOne(t *testing.T) {
	c := makeClaims()
	c[pkgJwt.ClaimAudience] = "test"
	aud, err := c.GetAudience()
	require.NoError(t, err)
	require.Equal(t, []string{c[pkgJwt.ClaimAudience].(string)}, []string(aud))
}

func TestAudienceOfTwo(t *testing.T) {
	c := makeClaims()
	c[pkgJwt.ClaimAudience] = []string{"test1", "test2"}
	aud, err := c.GetAudience()
	require.NoError(t, err)
	require.Equal(t, []string(aud), c[pkgJwt.ClaimAudience])
}

func TestEmptyIssuer(t *testing.T) {
	c := pkgJwt.Claims{}
	issuer, err := c.GetIssuer()
	require.NoError(t, err)
	require.Empty(t, issuer)
}

func TestInvalidIssuer(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimIssuer: 123,
	}
	_, err := c.GetIssuer()
	require.Error(t, err)
}

func TestIssuer(t *testing.T) {
	c := makeClaims()
	issuer, err := c.GetIssuer()
	require.NoError(t, err)
	require.Equal(t, issuer, c[pkgJwt.ClaimIssuer])
}

func TestEmptySubject(t *testing.T) {
	c := pkgJwt.Claims{}
	sub, err := c.GetSubject()
	require.NoError(t, err)
	require.Empty(t, sub)
}

func TestInvalidSubject(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimSubject: 123,
	}
	_, err := c.GetSubject()
	require.Error(t, err)
}

func TestSubject(t *testing.T) {
	c := makeClaims()
	sub, err := c.GetSubject()
	require.NoError(t, err)
	require.Equal(t, sub, c[pkgJwt.ClaimSubject])
}

func TestEmptyScope(t *testing.T) {
	c := pkgJwt.Claims{}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Empty(t, scope)
}

func TestInvalidScope(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimScope: 123,
	}
	_, err := c.GetScope()
	require.Error(t, err)
}

func TestSingleScope(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimScope: "scope",
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"scope"}, scope)
}

func TestScopeSpaceSeparatedString(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimScope: "test1 test2",
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1", "test2"}, scope)
}

func TestScopeMultipleSpaceSeparatedStrings(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimScope: []string{"test1 test2", "test3 test4"},
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1 test2", "test3 test4"}, scope)
}

func TestScopeSpaceSeparatedArrayItem(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimScope: []string{"test1 test2 test3"},
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1", "test2", "test3"}, scope)
}

func TestScopeOneArrayItem(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimScope: []string{"test1"},
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1"}, scope)
}

func TestScopeTwoArrayItems(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimScope: []string{"test1", "test2"},
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1", "test2"}, scope)
}

func TestEmptyID(t *testing.T) {
	c := pkgJwt.Claims{}
	id, err := c.GetID()
	require.NoError(t, err)
	require.Empty(t, id)
}

func TestInvalidID(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimID: 123,
	}
	id, err := c.GetID()
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, id)
}

func TestValidID(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimID: "123",
	}
	id, err := c.GetID()
	require.NoError(t, err)
	require.Equal(t, "123", id)
}

func TestEmptyEmail(t *testing.T) {
	c := pkgJwt.Claims{}
	email, err := c.GetEmail()
	require.NoError(t, err)
	require.Empty(t, email)
}

func TestInvalidEmail(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimEmail: 123,
	}
	email, err := c.GetEmail()
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, email)
}

func TestValidEmail(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimEmail: "test@example.com",
	}
	email, err := c.GetEmail()
	require.NoError(t, err)
	require.Equal(t, "test@example.com", email)
}

func TestEmptyClientID(t *testing.T) {
	c := pkgJwt.Claims{}
	clientID, err := c.GetClientID()
	require.NoError(t, err)
	require.Empty(t, clientID)
}

func TestInvalidClientID(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimClientID: 42,
	}
	clientID, err := c.GetClientID()
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, clientID)
}

func TestValidClientID(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimClientID: "testClientID",
	}
	clientID, err := c.GetClientID()
	require.NoError(t, err)
	require.Equal(t, "testClientID", clientID)
}

func TestEmptyName(t *testing.T) {
	c := pkgJwt.Claims{}
	name, err := c.GetName()
	require.NoError(t, err)
	require.Empty(t, name)
}

func TestInvalidName(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimName: 42,
	}
	name, err := c.GetName()
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, name)
}

func TestValidName(t *testing.T) {
	c := pkgJwt.Claims{
		pkgJwt.ClaimName: "John Doe",
	}
	name, err := c.GetName()
	require.NoError(t, err)
	require.Equal(t, "John Doe", name)
}

func TestEmptyOwner(t *testing.T) {
	c := pkgJwt.Claims{}
	owner, err := c.GetOwner("owner")
	require.NoError(t, err)
	require.Empty(t, owner)
}

func TestInvalidOwner(t *testing.T) {
	c := pkgJwt.Claims{
		"owner": 42,
	}
	owner, err := c.GetOwner("owner")
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, owner)
}

func TestValidOwner(t *testing.T) {
	c := pkgJwt.Claims{
		"owner": "John Doe",
	}
	owner, err := c.GetOwner("owner")
	require.NoError(t, err)
	require.Equal(t, "John Doe", owner)
}

func TestEmptyDeviceID(t *testing.T) {
	c := pkgJwt.Claims{}
	deviceID, err := c.GetDeviceID("deviceID")
	require.NoError(t, err)
	require.Empty(t, deviceID)
}

func TestInvalidDeviceID(t *testing.T) {
	c := pkgJwt.Claims{
		"deviceID": 42,
	}
	deviceID, err := c.GetDeviceID("deviceID")
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, deviceID)
}

func TestValidDeviceID(t *testing.T) {
	c := pkgJwt.Claims{
		"deviceID": "testDeviceID",
	}
	owner, err := c.GetDeviceID("deviceID")
	require.NoError(t, err)
	require.Equal(t, "testDeviceID", owner)
}

func TestValidateOwnerClaim(t *testing.T) {
	c := pkgJwt.Claims{}
	require.ErrorIs(t, c.ValidateOwnerClaim("owner", "testOwner"), pkgJwt.ErrOwnerClaimInvalid)

	c = pkgJwt.Claims{
		"owner": "testOwner",
	}
	require.ErrorIs(t, c.ValidateOwnerClaim("sub", "testOwner"), pkgJwt.ErrOwnerClaimInvalid)
	require.ErrorIs(t, c.ValidateOwnerClaim("owner", "invalid"), pkgJwt.ErrOwnerClaimInvalid)
	require.NoError(t, c.ValidateOwnerClaim("owner", "testOwner"))

	c = pkgJwt.Claims{
		"owner": float64(now.Unix()),
	}
	require.ErrorIs(t, c.ValidateOwnerClaim("owner", "testOwner"), pkgJwt.ErrOwnerClaimInvalid)
}

func TestParseToken(t *testing.T) {
	_, err := pkgJwt.ParseToken("invalid")
	require.ErrorIs(t, err, jwt.ErrTokenMalformed)

	expClaims := makeClaims()
	token := config.CreateJwtToken(t, expClaims)
	claims, err := pkgJwt.ParseToken(token)
	require.NoError(t, err)
	require.Len(t, claims, len(expClaims))

	expTime1, _ := expClaims.GetExpirationTime()
	expTime2, _ := claims.GetExpirationTime()
	require.Equal(t, expTime1, expTime2)
	nbf1, _ := expClaims.GetNotBefore()
	nbf2, _ := claims.GetNotBefore()
	require.Equal(t, nbf1, nbf2)
	iat1, _ := expClaims.GetIssuedAt()
	iat2, _ := claims.GetIssuedAt()
	require.Equal(t, iat1, iat2)
	aud1, _ := expClaims.GetAudience()
	aud2, _ := claims.GetAudience()
	require.Equal(t, aud1, aud2)
	iss1, _ := expClaims.GetIssuer()
	iss2, _ := claims.GetIssuer()
	require.Equal(t, iss1, iss2)
	sub1, _ := expClaims.GetSubject()
	sub2, _ := claims.GetSubject()
	require.Equal(t, sub1, sub2)
	scope1, _ := expClaims.GetScope()
	scope2, _ := claims.GetScope()
	require.Equal(t, scope1, scope2)
	id1, _ := expClaims.GetID()
	id2, _ := claims.GetID()
	require.Equal(t, id1, id2)
	email1, _ := expClaims.GetEmail()
	email2, _ := claims.GetEmail()
	require.Equal(t, email1, email2)
	clientID1, _ := expClaims.GetClientID()
	clientID2, _ := claims.GetClientID()
	require.Equal(t, clientID1, clientID2)
}

func checkClaims(t *testing.T, tokenClaims jwt.Claims, expError error) {
	token := config.CreateJwtToken(t, tokenClaims)
	_, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) {
		return jwt.VerificationKeySet{
			Keys: []jwt.VerificationKey{
				[]uint8(config.JWTSecret),
			},
		}, nil
	}, jwt.WithIssuedAt())
	if expError == nil {
		require.NoError(t, err)
		return
	}
	require.ErrorIs(t, err, expError)
}

func TestValidate(t *testing.T) {
	expiredTime := now.Add(-time.Hour)
	futureTime := now.Add(time.Hour)

	claims := pkgJwt.Claims{
		pkgJwt.ClaimExpirationTime: float64(expiredTime.Unix()),
		pkgJwt.ClaimNotBefore:      float64(now.Unix()),
		pkgJwt.ClaimIssuedAt:       float64(now.Unix()),
	}
	checkClaims(t, claims, jwt.ErrTokenExpired)

	// Set that the token is not expired.
	claims[pkgJwt.ClaimExpirationTime] = float64(futureTime.Unix())
	// Check that a token used before issued returns an error.
	claims[pkgJwt.ClaimIssuedAt] = float64(futureTime.Unix())
	checkClaims(t, claims, jwt.ErrTokenUsedBeforeIssued)

	// Check that a token not yet valid returns an error.
	claims[pkgJwt.ClaimNotBefore] = float64(futureTime.Unix())
	claims[pkgJwt.ClaimIssuedAt] = float64(now.Unix())
	checkClaims(t, claims, jwt.ErrTokenNotValidYet)

	// Check that a valid token does not return an error.
	claims[pkgJwt.ClaimNotBefore] = float64(expiredTime.Unix())
	claims[pkgJwt.ClaimIssuedAt] = float64(expiredTime.Unix())
	checkClaims(t, claims, nil)
}
