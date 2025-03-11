package certificateSigner

import (
	"crypto"
	"crypto/x509"
	"encoding/asn1"
)

var ExtendedKeyUsage_IDENTITY_CERTIFICATE = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 44924, 1, 6}

func NewIdentityCertificateSigner(caCert []*x509.Certificate, caKey crypto.PrivateKey, opts ...Opt) (*CertificateSigner, error) {
	var cfg SignerConfig
	for _, o := range opts {
		o(&cfg)
	}
	overrideCertTemplate := func(template *x509.Certificate) error {
		template.UnknownExtKeyUsage = []asn1.ObjectIdentifier{ExtendedKeyUsage_IDENTITY_CERTIFICATE}
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
		if cfg.OverrideCertTemplate != nil {
			return cfg.OverrideCertTemplate(template)
		}
		return nil
	}
	opts = append(opts, WithOverrideCertTemplate(overrideCertTemplate))
	return New(caCert, caKey, opts...)
}
