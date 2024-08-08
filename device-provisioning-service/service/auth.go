package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"time"

	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
)

type AuthHandler interface {
	// tls.Config overrides
	VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error
	VerifyConnection(tls.ConnectionState) error

	GetChainsCache() *cache.Cache[uint64, [][]*x509.Certificate]
}

func verifyChain(certificates, certificatesAuthority []*x509.Certificate, currentTime time.Time) ([][]*x509.Certificate, error) {
	chains, err := pkgX509.Verify(certificates, certificatesAuthority, false, x509.VerifyOptions{
		CurrentTime: currentTime,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		return nil, err
	}
	certificate := certificates[0]
	// verify EKU manually
	ekuHasClient := false
	for _, eku := range certificate.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			ekuHasClient = true
		}
	}
	if !ekuHasClient {
		return nil, errors.New("not contains ExtKeyUsageClientAuth")
	}
	return chains, nil
}

type DefaultAuthHandler struct {
	config                Config
	chainsCache           *cache.Cache[uint64, [][]*x509.Certificate]
	enrollmentGroupsCache *EnrollmentGroupsCache
}

func MakeDefaultAuthHandler(config Config, enrollmentGroupsCache *EnrollmentGroupsCache) DefaultAuthHandler {
	chainsCache := cache.NewCache[uint64, [][]*x509.Certificate]()
	return DefaultAuthHandler{
		config:                config,
		chainsCache:           chainsCache,
		enrollmentGroupsCache: enrollmentGroupsCache,
	}
}

func (d DefaultAuthHandler) GetChainsCache() *cache.Cache[uint64, [][]*x509.Certificate] {
	return d.chainsCache
}

func (d DefaultAuthHandler) VerifyPeerCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	certs := make([]*x509.Certificate, 0, len(rawCerts))
	issuerNames := make([]string, 0, len(rawCerts))
	for _, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return pkgX509.NewError(nil, err)
		}
		certs = append(certs, cert)
		issuerNames = append(issuerNames, cert.Issuer.CommonName)
	}
	if len(certs) == 0 {
		return pkgX509.NewError(nil, errors.New("empty certificates"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), d.config.APIs.COAP.InactivityMonitor.Timeout)
	defer cancel()
	success := false
	err := d.enrollmentGroupsCache.GetEnrollmentGroupsByIssuerNames(ctx, issuerNames, func(g *EnrollmentGroup) bool {
		currentTime := time.Now()
		if g.GetAttestationMechanism().GetX509().GetExpiredCertificateEnabled() {
			currentTime = certs[0].NotBefore.Add(certs[0].NotAfter.Sub(certs[0].NotBefore) / 2)
		}
		chains, err := verifyChain(certs, g.AttestationMechanismX509CertificateChain, currentTime)
		if err == nil {
			el, loaded := d.chainsCache.LoadOrStore(toCRC64(certs[0].Raw), cache.NewElement(chains, time.Now().Add(d.config.APIs.COAP.InactivityMonitor.Timeout), nil))
			if loaded {
				el.ValidUntil.Store(time.Now().Add(d.config.APIs.COAP.InactivityMonitor.Timeout / 2))
			}
			success = true
			return false
		}
		return true
	})
	if err != nil {
		return pkgX509.NewError([][]*x509.Certificate{certs}, err)
	}
	if success {
		return nil
	}
	return pkgX509.NewError([][]*x509.Certificate{certs}, errors.New("untrusted certificate"))
}

func (d DefaultAuthHandler) VerifyConnection(tls.ConnectionState) error {
	return nil
}
