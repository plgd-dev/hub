package uri

import "strings"

const (
	API  string = "/api/v1"
	Sign string = API + "/sign"

	SignIdentityCertificate string = Sign + "/identity-csr"
	SignCertificate         string = Sign + "/csr"

	IssuerIDKey string = "issuerId"

	SigningRevocationListBase string = API + "/signing/crl"
	SigningRevocationList     string = SigningRevocationListBase + "/{" + IssuerIDKey + "}"
)

var QueryCaseInsensitive = map[string]string{
	strings.ToLower(IssuerIDKey): IssuerIDKey,
}
