package uri

const (
	RedirectURIKey  = "redirect_uri"
	StateKey        = "state"
	ClientIDKey     = "client_id"
	NonceKey        = "nonce"
	CodeKey         = "code"
	ScopeKey        = "scope"
	ReturnToKey     = "returnTo"
	Auth0ClientKey  = "auth0Client"
	GrantTypeKey    = "grant_type"
	UsernameKey     = "username"
	PasswordKey     = "password"
	AudienceKey     = "audience"
	RefreshTokenKey = "refresh_token"
	ErrorMessageKey = "error"
	DeviceIDKey     = "deviceId"
	ResponseModeKey = "response_mode"
	ResponseTypeKey = "response_type"

	Token               = "/oauth/token"
	Authorize           = "/authorize"
	UserInfo            = Authorize + "/userinfo"
	JWKs                = "/.well-known/jwks.json"
	OpenIDConfiguration = "/.well-known/openid-configuration"
	LogOut              = "/v2/logout"
)
