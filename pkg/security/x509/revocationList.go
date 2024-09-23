package x509

import "crypto/x509"

func IsRevoked(certificate *x509.Certificate, crl *x509.RevocationList) bool {
	for _, entry := range crl.RevokedCertificateEntries {
		if certificate.SerialNumber.Cmp(entry.SerialNumber) == 0 {
			return true
		}
	}
	return false
}
