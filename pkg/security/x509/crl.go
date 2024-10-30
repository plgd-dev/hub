package x509

import (
	"crypto/x509"
	"fmt"
	"net/url"
)

func IsRevoked(certificate *x509.Certificate, crl *x509.RevocationList) bool {
	for _, entry := range crl.RevokedCertificateEntries {
		if certificate.SerialNumber.Cmp(entry.SerialNumber) == 0 {
			return true
		}
	}
	return false
}

func ValidateCRLDistributionPointAddress(s string) error {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return fmt.Errorf("invalid address(%s)", s)
	}
	if !u.IsAbs() {
		return fmt.Errorf("invalid relative URL address(%s)", s)
	}
	return nil
}
