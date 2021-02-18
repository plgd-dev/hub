package uri

const (
	RedirectURIQueryKey = "redirect_uri"
	StateQueryKey       = "state"
	ClientIDQueryKey    = "client_id"
	NonceQueryKey       = "nonce"
	CodeQueryKey        = "code"
	ReturnToQueryKey    = "returnTo"
	Auth0ClientQueryKey = "auth0Client"

	Token     = "/oauth/token"
	Authorize = "/authorize"
	UserInfo  = Authorize + "/userinfo"
	JWKs      = "/.well-known/jwks.json"
	LogOut    = "/v2/logout"
)
