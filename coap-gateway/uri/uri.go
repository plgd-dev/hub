package uri

// Resource Service URIs.
const (
	Base   = "/oic"
	Secure = Base + "/sec"

	SecureSignUp       = Secure + "/account"
	SignUp             = Base + "/account"
	SecureRefreshToken = Secure + "/tokenrefresh"
	RefreshToken       = Base + "/tokenrefresh"
	SecureSignIn       = Secure + "/session"
	SignIn             = Base + "/account/session"
	ResourceDirectory  = Base + "/rd"
	ResourceDiscovery  = Base + "/res"
	ResourcePing       = Base + "/ping"
)
