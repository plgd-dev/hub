package service

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/smallstep/certificates/authority/provisioner"
)

type Signer = interface {
	Sign(ctx context.Context, csr []byte) (certChain []byte, err error)
	ID() string
}

type authority struct {
	signers map[string]Signer
}

func loadPEM(pemCert []byte) ([]*x509.Certificate, error) {
	cert := make([]*x509.Certificate, 0, 4)
	for {
		if len(pemCert) == 0 {
			break
		}
		b, rest := pem.Decode(pemCert)
		if b == nil {
			return nil, fmt.Errorf("cannot decode pem of cert")
		}
		crt, err := x509.ParseCertificate(b.Bytes)
		if err != nil {
			return nil, err
		}
		cert = append(cert, crt)
		pemCert = rest
	}
	if len(cert) == 0 {
		return nil, fmt.Errorf("cert not found")
	}
	return cert, nil
}

// Sign creates a signed certificate from a certificate signing request.
func (s *authority) Sign(cr *x509.CertificateRequest, opts provisioner.Options, signOpts ...provisioner.SignOption) (*x509.Certificate, *x509.Certificate, error) {
	var ID string
	var ctx context.Context
	for _, s := range signOpts {
		switch v := s.(type) {
		case ProvisionerName:
			ID = string(v)
		case context.Context:
			ctx = v
		}
	}
	if ctx == nil {
		return nil, nil, fmt.Errorf("unknown context for sign")
	}

	signer, ok := s.signers[ID]
	if !ok {
		return nil, nil, fmt.Errorf("unknown ID %v of provisioner", ID)
	}

	pemCSR := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: cr.Raw,
	})
	pemCert, err := signer.Sign(ctx, pemCSR)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot sign cert for %v: %w", ID, err)
	}

	certs, err := loadPEM(pemCert)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot load cert for %v: %w", ID, err)
	}

	if len(certs) > 1 {
		return certs[0], certs[1], nil
	}
	return certs[0], nil, nil
}

// LoadProvisionerByID calls out to the SignAuthority interface to load a
// provisioner by ID.
func (s *authority) LoadProvisionerByID(ID string) (provisioner.Interface, error) {
	v := strings.Split(ID, "/")
	if len(v) < 2 {
		return nil, fmt.Errorf("invalid ID %v of provisioner", ID)
	}
	signer, ok := s.signers[v[1]]
	if !ok {
		return nil, fmt.Errorf("unknown ID %v of provisioner", ID)
	}

	return ACME{ProvisionerName: ProvisionerName(signer.ID())}, nil
}
