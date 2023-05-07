package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/stretchr/testify/require"
)

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
	c := Claims{}
	exp, err := c.GetExpirationTime()
	require.NoError(t, err)
	require.Nil(t, exp)
}

func TestInvalidExpirationTime(t *testing.T) {
	c := Claims{
		ClaimExpirationTime: "string",
	}
	_, err := c.GetExpirationTime()
	require.Error(t, err)
}

func TestExpirationTime(t *testing.T) {
	c := testClaims()
	exp, err := c.GetExpirationTime()
	require.NoError(t, err)
	require.Equal(t, exp.Time, toTime(t, c[ClaimExpirationTime]))
}

func TestEmptyNotBefore(t *testing.T) {
	c := Claims{}
	exp, err := c.GetNotBefore()
	require.NoError(t, err)
	require.Nil(t, exp)
}

func TestInvalidNotBefore(t *testing.T) {
	c := Claims{
		ClaimNotBefore: "string",
	}
	_, err := c.GetNotBefore()
	require.Error(t, err)
}

func TestNotBefore(t *testing.T) {
	c := testClaims()
	nbf, err := c.GetNotBefore()
	require.NoError(t, err)
	require.Equal(t, nbf.Time, toTime(t, c[ClaimNotBefore]))
}

func TestEmptyIssuedAt(t *testing.T) {
	c := Claims{}
	exp, err := c.GetIssuedAt()
	require.NoError(t, err)
	require.Nil(t, exp)
}

func TestInvalidIssuedAt(t *testing.T) {
	c := Claims{
		ClaimIssuedAt: "string",
	}
	_, err := c.GetIssuedAt()
	require.Error(t, err)
}

func TestIssuedAt(t *testing.T) {
	c := testClaims()
	iat, err := c.GetIssuedAt()
	require.NoError(t, err)
	require.Equal(t, iat.Time, toTime(t, c[ClaimIssuedAt]))
}

func TestEmptyAudience(t *testing.T) {
	c := Claims{}
	aud, err := c.GetAudience()
	require.NoError(t, err)
	require.Empty(t, aud)
}

func TestInvalidAudience(t *testing.T) {
	c := Claims{
		ClaimAudience: []interface{}{123},
	}
	_, err := c.GetAudience()
	require.Error(t, err)
}

func TestAudience(t *testing.T) {
	c := testClaims()
	aud, err := c.GetAudience()
	require.NoError(t, err)
	require.Equal(t, []string(aud), c[ClaimAudience])
}

func TestAudienceOfOne(t *testing.T) {
	c := testClaims()
	c[ClaimAudience] = "test"
	aud, err := c.GetAudience()
	require.NoError(t, err)
	require.Equal(t, []string(aud), []string{c[ClaimAudience].(string)})
}

func TestAudienceOfTwo(t *testing.T) {
	c := testClaims()
	c[ClaimAudience] = []string{"test1", "test2"}
	aud, err := c.GetAudience()
	require.NoError(t, err)
	require.Equal(t, []string(aud), c[ClaimAudience])
}

func TestEmptyIssuer(t *testing.T) {
	c := Claims{}
	issuer, err := c.GetIssuer()
	require.NoError(t, err)
	require.Empty(t, issuer)
}

func TestInvalidIssuer(t *testing.T) {
	c := Claims{
		"iss": 123,
	}
	_, err := c.GetIssuer()
	require.Error(t, err)
}

func TestIssuer(t *testing.T) {
	c := testClaims()
	issuer, err := c.GetIssuer()
	require.NoError(t, err)
	require.Equal(t, issuer, c[ClaimIssuer])
}

func TestEmptySubject(t *testing.T) {
	c := Claims{}
	sub, err := c.GetSubject()
	require.NoError(t, err)
	require.Empty(t, sub)
}

func TestInvalidSubject(t *testing.T) {
	c := Claims{
		"sub": 123,
	}
	_, err := c.GetSubject()
	require.Error(t, err)
}

func TestSubject(t *testing.T) {
	c := testClaims()
	sub, err := c.GetSubject()
	require.NoError(t, err)
	require.Equal(t, sub, c[ClaimSubject])
}

func TestEmptyScope(t *testing.T) {
	c := Claims{}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Empty(t, scope)
}

func TestInvalidScope(t *testing.T) {
	c := Claims{
		ClaimScope: 123,
	}
	_, err := c.GetScope()
	require.Error(t, err)
}

func TestSingleScope(t *testing.T) {
	c := Claims{
		ClaimScope: "scope",
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"scope"}, scope)
}

func TestScopeSpaceSeparatedString(t *testing.T) {
	c := Claims{
		ClaimScope: "test1 test2",
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1", "test2"}, scope)
}

func TestScopeMultipleSpaceSeparatedStrings(t *testing.T) {
	c := Claims{
		ClaimScope: []string{"test1 test2", "test3 test4"},
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1 test2", "test3 test4"}, scope)
}

func TestScopeSpaceSeparatedArrayItem(t *testing.T) {
	c := Claims{
		ClaimScope: []string{"test1 test2 test3"},
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1", "test2", "test3"}, scope)
}

func TestScopeOneArrayItem(t *testing.T) {
	c := Claims{
		ClaimScope: []string{"test1"},
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1"}, scope)
}

func TestScopeTwoArrayItems(t *testing.T) {
	c := Claims{
		ClaimScope: []string{"test1", "test2"},
	}
	scope, err := c.GetScope()
	require.NoError(t, err)
	require.Equal(t, jwt.ClaimStrings{"test1", "test2"}, scope)
}

func TestEmptyID(t *testing.T) {
	c := Claims{}
	id, err := c.GetID()
	require.NoError(t, err)
	require.Empty(t, id)
}

func TestInvalidID(t *testing.T) {
	c := Claims{
		ClaimID: 123,
	}
	id, err := c.GetID()
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, id)
}

func TestValidID(t *testing.T) {
	c := Claims{
		ClaimID: "123",
	}
	id, err := c.GetID()
	require.NoError(t, err)
	require.Equal(t, "123", id)
}

func TestEmptyEmail(t *testing.T) {
	c := Claims{}
	email, err := c.GetEmail()
	require.NoError(t, err)
	require.Empty(t, email)
}

func TestInvalidEmail(t *testing.T) {
	c := Claims{
		ClaimEmail: 123,
	}
	email, err := c.GetEmail()
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, email)
}

func TestValidEmail(t *testing.T) {
	c := Claims{
		ClaimEmail: "test@example.com",
	}
	email, err := c.GetEmail()
	require.NoError(t, err)
	require.Equal(t, "test@example.com", email)
}

func TestEmptyClientID(t *testing.T) {
	c := Claims{}
	clientID, err := c.GetClientID()
	require.NoError(t, err)
	require.Empty(t, clientID)
}

func TestInvalidClientID(t *testing.T) {
	c := Claims{
		ClaimClientID: 42,
	}
	clientID, err := c.GetClientID()
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, clientID)
}

func TestValidClientID(t *testing.T) {
	c := Claims{
		ClaimClientID: "testClientID",
	}
	clientID, err := c.GetClientID()
	require.NoError(t, err)
	require.Equal(t, "testClientID", clientID)
}

func TestEmptyName(t *testing.T) {
	c := Claims{}
	name, err := c.GetName()
	require.NoError(t, err)
	require.Equal(t, "", name)
}

func TestInvalidName(t *testing.T) {
	c := Claims{
		ClaimName: 42,
	}
	name, err := c.GetName()
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Equal(t, "", name)
}

func TestValidName(t *testing.T) {
	c := Claims{
		ClaimName: "John Doe",
	}
	name, err := c.GetName()
	require.NoError(t, err)
	require.Equal(t, "John Doe", name)
}

func TestEmptyOwner(t *testing.T) {
	c := Claims{}
	owner, err := c.GetOwner("owner")
	require.NoError(t, err)
	require.Empty(t, owner)
}

func TestInvalidOwner(t *testing.T) {
	c := Claims{
		"owner": 42,
	}
	owner, err := c.GetOwner("owner")
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, owner)
}

func TestValidOwner(t *testing.T) {
	c := Claims{
		"owner": "John Doe",
	}
	owner, err := c.GetOwner("owner")
	require.NoError(t, err)
	require.Equal(t, "John Doe", owner)
}

func TestEmptyDeviceID(t *testing.T) {
	c := Claims{}
	deviceID, err := c.GetDeviceID("deviceID")
	require.NoError(t, err)
	require.Empty(t, deviceID)
}

func TestInvalidDeviceID(t *testing.T) {
	c := Claims{
		"deviceID": 42,
	}
	deviceID, err := c.GetDeviceID("deviceID")
	require.ErrorIs(t, err, jwt.ErrInvalidType)
	require.Empty(t, deviceID)
}

func TestValidDeviceID(t *testing.T) {
	c := Claims{
		"deviceID": "testDeviceID",
	}
	owner, err := c.GetDeviceID("deviceID")
	require.NoError(t, err)
	require.Equal(t, "testDeviceID", owner)
}

func TestValidate(t *testing.T) {
	now := time.Now()
	expiredTime := now.Add(-time.Hour)
	futureTime := now.Add(time.Hour)

	claims := Claims{
		ClaimExpirationTime: float64(expiredTime.Unix()),
		ClaimNotBefore:      float64(now.Unix()),
		ClaimIssuedAt:       float64(now.Unix()),
	}

	// Check that an expired token returns an error.
	err := claims.Validate()
	require.Error(t, err, "Expected an error for an expired token")

	// Set that the token is not expired.
	claims[ClaimExpirationTime] = float64(futureTime.Unix())
	// Check that a token used before issued returns an error.
	claims[ClaimIssuedAt] = float64(futureTime.Unix())
	err = claims.Validate()
	require.Error(t, err, "Expected an error for a token used before issued")

	// Check that a token not yet valid returns an error.
	claims[ClaimNotBefore] = float64(futureTime.Unix())
	claims[ClaimIssuedAt] = float64(now.Unix())
	err = claims.Validate()
	require.Error(t, err, "Expected an error for a token not yet valid")

	// Check that a valid token does not return an error.
	claims[ClaimNotBefore] = float64(expiredTime.Unix())
	claims[ClaimIssuedAt] = float64(expiredTime.Unix())
	err = claims.Validate()
	require.NoError(t, err, "Unexpected error")
}

var now = time.Now()

func testClaims() Claims {
	return Claims{
		ClaimExpirationTime: float64(now.Add(time.Hour).Unix()),
		ClaimNotBefore:      float64(now.Unix()),
		ClaimIssuedAt:       float64(now.Unix()),
		ClaimAudience:       []string{"testAudience", "testAudience2"},
		ClaimIssuer:         "testIssuer",
		ClaimSubject:        "testSubject",
		ClaimScope:          []string{"testScope"},
		ClaimID:             "testID",
		ClaimEmail:          "testEmail",
		ClaimClientID:       "testClientID",
	}
}
