package uri

import "github.com/lestrrat-go/jwx/v2/jwt"

const (
	ClientIDKey            = "client_id"
	ScopeKey               = "scope"
	GrantTypeKey           = "grant_type"
	UsernameKey            = "username"
	PasswordKey            = "password"
	AudienceKey            = jwt.AudienceKey
	SubjectKey             = jwt.SubjectKey
	AccessTokenKey         = "access_token"
	DeviceIDKey            = "https://plgd.dev/deviceId"
	OwnerKey               = "https://plgd.dev/owner"
	ClientAssertionTypeKey = "client_assertion_type"
	ClientAssertionTypeJWT = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	ClientAssertionKey     = "client_assertion"

	Token               = "/m2m-oauth-server/oauth/token"
	JWKs                = "/.well-known/m2m-oauth-server/jwks.json"
	OpenIDConfiguration = "/.well-known/m2m-oauth-server/openid-configuration"
)
