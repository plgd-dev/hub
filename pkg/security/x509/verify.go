package x509

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
)

type (
	VerifyByCRL                         = func(context.Context, *x509.Certificate, []string) (bool, error)
	VerifyForDistributionPoint          = func(context.Context, *x509.Certificate, string) (bool, error)
	CustomDistributionPointVerification = map[string]VerifyForDistributionPoint
	Options                             struct {
		CustomDistributionPointVerification CustomDistributionPointVerification
	}

	SetOption = func(cfg *Options)
)

func WithCustomDistributionPointVerification(customDistributionPointVerification CustomDistributionPointVerification) SetOption {
	return func(o *Options) {
		o.CustomDistributionPointVerification = customDistributionPointVerification
	}
}

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
		return nil, errors.New("at least one certificate need to be set")
	}
	if len(certificateAuthorities) == 0 {
		return nil, errors.New("at least one certificate authority need to be set")
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

type CRLVerification struct {
	Enabled bool
	Ctx     context.Context
	Verify  VerifyByCRL
}

func VerifyChain(chain []*x509.Certificate, capool *x509.CertPool, crlVerify CRLVerification) error {
	if len(chain) == 0 {
		return errors.New("certificate chain is empty")
	}
	certificate := chain[0]
	intermediateCAPool := x509.NewCertPool()
	for i := 1; i < len(chain); i++ {
		intermediateCAPool.AddCert(chain[i])
	}
	_, err := certificate.Verify(x509.VerifyOptions{
		Roots:         capool,
		Intermediates: intermediateCAPool,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		return err
	}
	if crlVerify.Enabled && len(certificate.CRLDistributionPoints) > 0 {
		if crlVerify.Verify == nil {
			return errors.New("cannot verify certificate validity by CRL: verification function not provided")
		}
		ctx := crlVerify.Ctx
		if ctx == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
		}
		revoked, err := crlVerify.Verify(ctx, certificate, certificate.CRLDistributionPoints)
		if err != nil {
			return fmt.Errorf("cannot verify certificate validity by CRL: %w", err)
		}
		if revoked {
			return errors.New("certificate revoked by CRL")
		}
	}
	return nil
}

func VerifyChains(capool *x509.CertPool, crlVerify CRLVerification) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(_ [][]byte, chains [][]*x509.Certificate) error {
		var errs *multierror.Error
		for _, chain := range chains {
			err := VerifyChain(chain, capool, crlVerify)
			if err == nil {
				return nil
			}
			errs = multierror.Append(errs, err)
		}
		err := errors.New("empty chains")
		if errs.ErrorOrNil() != nil {
			err = errs
		}
		return NewError(chains, err)
	}
}
