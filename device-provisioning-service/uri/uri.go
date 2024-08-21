package uri

// COAP Service URIs.
const (
	APIV1 = "/api/v1"

	CoAPsTCPSchemePrefix = "coaps+tcp://"

	Provisioning       = APIV1 + "/provisioning"
	Resources          = Provisioning + "/resources"
	ACLs               = Provisioning + "/acls"
	Credentials        = Provisioning + "/credentials"
	CloudConfiguration = Provisioning + "/cloud-configuration"
	Ownership          = Provisioning + "/ownership"
)
