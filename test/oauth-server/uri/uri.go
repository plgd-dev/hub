package uri

const (
	RedirectURIKey = "redirect_uri"
	StateKey       = "state"
	ClientIDKey    = "client_id"
	NonceKey       = "nonce"
	CodeKey        = "code"
	ReturnToKey    = "returnTo"
	Auth0ClientKey = "auth0Client"
	GrantTypeKey   = "grant_type"
	UsernameKey    = "username"
	PasswordKey    = "password"
	AudienceKey    = "audience"
	ResponseMode   = "response_mode"

	Token     = "/oauth/token"
	Authorize = "/authorize"
	UserInfo  = Authorize + "/userinfo"
	JWKs      = "/.well-known/jwks.json"
	LogOut    = "/v2/logout"
)
