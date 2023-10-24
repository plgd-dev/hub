package uri

// Resource Service URIs.
const (
	Api   = "/api"
	ApiV1 = Api + "/v1"
	Base  = "/oic"

	Secure = Base + "/sec"

	SignUp            = Secure + "/account"
	RefreshToken      = Secure + "/tokenrefresh"
	SignIn            = Secure + "/session"
	ResourceDirectory = Base + "/rd"
	ResourceDiscovery = Base + "/res"
	ResourceRoute     = ApiV1 + "/devices"

	InterfaceQueryKey       = "if"
	InterfaceQueryKeyPrefix = InterfaceQueryKey + "="
)
