package uri

import "github.com/lestrrat-go/jwx/v2/jwt"

const (
	ClientIDKey    = "client_id"
	ScopeKey       = "scope"
	GrantTypeKey   = "grant_type"
	UsernameKey    = "username"
	PasswordKey    = "password"
	AudienceKey    = jwt.AudienceKey
	SubjectKey     = jwt.SubjectKey
	AccessTokenKey = "access_token"
	DeviceIDKey    = "https://plgd.dev/deviceId"
	OwnerKey       = "https://plgd.dev/owner"

	Token               = "/oauth/token"
	JWKs                = "/.well-known/jwks.json"
	OpenIDConfiguration = "/.well-known/openid-configuration"
)
