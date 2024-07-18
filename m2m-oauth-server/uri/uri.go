package uri

import "github.com/lestrrat-go/jwx/v2/jwt"

const (
	ClientIDKey            = "client_id"
	ClientSecretKey        = "client_secret"
	ScopeKey               = "scope"
	GrantTypeKey           = "grant_type"
	AudienceKey            = jwt.AudienceKey
	AccessTokenKey         = "access_token"
	ClientAssertionTypeKey = "client_assertion_type"
	ClientAssertionTypeJWT = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	ClientAssertionKey     = "client_assertion"

	OriginalTokenClaims = "https://plgd.dev/originalClaims"

	Base                = "/m2m-oauth-server"
	Token               = Base + "/oauth/token"
	JWKs                = Base + "/.well-known/jwks.json"
	OpenIDConfiguration = Base + "/.well-known/openid-configuration"
)
