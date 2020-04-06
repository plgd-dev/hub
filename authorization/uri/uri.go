package uri

const (
	Base string = "/api/authz"

	AuthorizationCode string = Base + "/code"
	AccessToken       string = Base + "/token"
	OAuthCallback     string = Base + "/callback"
	JWKs              string = "/.well-known/jwks.json"

	Healthcheck string = "/"
)
