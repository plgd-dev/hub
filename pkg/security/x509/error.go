package x509

import (
	"crypto/x509"
	"time"
)

type Error struct {
	chains [][]*x509.Certificate
	err    error
}

func NewError(chains [][]*x509.Certificate, err error) *Error {
	return &Error{chains: chains, err: err}
}

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) Chains() [][]*x509.Certificate {
	return e.chains
}

func ParseCertificate(cert *x509.Certificate) *CertificateInfo {
	var serialNumber string
	if cert.SerialNumber != nil {
		serialNumber = cert.SerialNumber.String()
	}
	return &CertificateInfo{
		NotBefore:          cert.NotBefore,
		NotAfter:           cert.NotAfter,
		SubjectName:        cert.Subject.CommonName,
		Organization:       cert.Subject.Organization,
		IssuerName:         cert.Issuer.CommonName,
		IssuerOrganization: cert.Issuer.Organization,
		SerialNumber:       serialNumber,
	}
}

type CertificateInfo struct {
	SubjectName        string    `json:"subjectName,omitempty"`
	Organization       []string  `json:"organization,omitempty"`
	IssuerName         string    `json:"issuerName,omitempty"`
	IssuerOrganization []string  `json:"issuerOrganization,omitempty"`
	NotAfter           time.Time `json:"notAfter,omitempty"`
	NotBefore          time.Time `json:"notBefore,omitempty"`
	SerialNumber       string    `json:"serialNumber,omitempty"`
}

func (e *Error) LeafCertificateInfo() *CertificateInfo {
	if len(e.chains) == 0 || len(e.chains[0]) == 0 {
		return nil
	}
	return ParseCertificate(e.chains[0][0])
}
