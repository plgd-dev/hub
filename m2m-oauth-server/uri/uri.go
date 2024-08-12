package uri

import "github.com/lestrrat-go/jwx/v2/jwt"

const (
	ClientIDKey            = "client_id"
	ClientSecretKey        = "client_secret"
	TokenNameKey           = "token_name"
	ScopeKey               = "scope"
	GrantTypeKey           = "grant_type"
	ExpirationKey          = "expiration"
	AudienceKey            = jwt.AudienceKey
	AccessTokenKey         = "access_token"
	ClientAssertionTypeKey = "client_assertion_type"
	ClientAssertionTypeJWT = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	ClientAssertionKey     = "client_assertion"
	TokenTypeKey           = "token_type"
	ExpiresInKey           = "expires_in"
	IDFilterQuery          = "idFilter"

	OriginalTokenClaims = "originalTokenClaims"

	Base                = "/m2m-oauth-server"
	API                 = Base + "/api/v1"
	Token               = Base + "/oauth/token"
	JWKs                = Base + "/.well-known/jwks.json"
	OpenIDConfiguration = Base + "/.well-known/openid-configuration"
	Tokens              = API + "/tokens"
)
