package uri

const (
	HrefKey     string = "Href"
	DeviceIDKey string = "deviceId"

	InterfaceQueryKey      string = "interface"
	SkipShadowQueryKey     string = "skipShadow"
	DeviceIDFilterQueryKey string = "deviceId"
	TypeFilterQueryKey     string = "type"

	CorrelationIDHeader string = "CorrelationID"

	API   string = "/api/v1"
	APIWS string = API + "/ws"

	// ocfcloud configuration
	ClientConfiguration = "/.well-known/ocfcloud-configuration"

	// oauth configuration for ui
	OAuthConfiguration = "/auth_config.json"

	//certificate-authority
	CertificaAuthority     = API + "/certificate-authority"
	CertificaAuthoritySign = CertificaAuthority + "/sign"
)
