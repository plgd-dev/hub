package uri

// Resource Service URIs.
const (
	Base   = "/oic"
	Secure = Base + "/sec"

	SignUp            = Secure + "/account"
	RefreshToken      = Secure + "/tokenrefresh"
	SignIn            = Secure + "/session"
	ResourceDirectory = Base + "/rd"
	ResourceDiscovery = Base + "/res"
	ResourceRoute     = Base + "/route"
)
