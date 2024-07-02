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

	Base                = "/m2m-oauth-server"
	Token               = Base + "/oauth/token"
	JWKs                = Base + "/.well-known/jwks.json"
	OpenIDConfiguration = Base + "/.well-known/openid-configuration"
)
