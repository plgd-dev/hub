package x509

import (
	"bytes"
	"crypto/x509"
	"fmt"
)

func IsRootCA(cert *x509.Certificate) bool {
	return cert.IsCA && bytes.Equal(cert.RawIssuer, cert.RawSubject) && cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature) == nil
}

func setCAPools(roots *x509.CertPool, intermediates *x509.CertPool, certs []*x509.Certificate) {
	for _, cert := range certs {
		if !cert.IsCA {
			continue
		}
		if IsRootCA(cert) {
			if roots == nil {
				continue
			}
			roots.AddCert(cert)
			continue
		}
		intermediates.AddCert(cert)
	}
}

// Verify verifies certificate against certificate authorities.
func Verify(certificates []*x509.Certificate, certificateAuthorities []*x509.Certificate, useSystemRoots bool, opts x509.VerifyOptions) ([][]*x509.Certificate, error) {
	if len(certificates) == 0 {
		return nil, fmt.Errorf("at least one certificate need to be set")
	}
	if len(certificateAuthorities) == 0 {
		return nil, fmt.Errorf("at least one certificate authority need to be set")
	}
	intermediateCA := x509.NewCertPool()
	rootCA := x509.NewCertPool()
	if useSystemRoots {
		var err error
		rootCA, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
	}

	setCAPools(rootCA, intermediateCA, certificateAuthorities)
	// skip root CA, root need to be added to certificateAuthorities argument
	if len(certificates) > 1 {
		setCAPools(nil, intermediateCA, certificates[1:])
	}

	if opts.Roots == nil {
		opts.Roots = rootCA
	}
	if opts.Intermediates == nil {
		opts.Intermediates = intermediateCA
	}
	return certificates[0].Verify(opts)
}
