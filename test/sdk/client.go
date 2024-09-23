package sdk

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certificateSigner"
)

type testSetupSecureClient struct {
	ca      []*x509.Certificate
	mfgCA   []*x509.Certificate
	mfgCert tls.Certificate
}

var errNotSet = errors.New("not set")

func (c *testSetupSecureClient) GetManufacturerCertificate() (tls.Certificate, error) {
	if c.mfgCert.PrivateKey == nil {
		return c.mfgCert, errNotSet
	}
	return c.mfgCert, nil
}

func (c *testSetupSecureClient) GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.mfgCA) == 0 {
		return nil, errNotSet
	}
	return c.mfgCA, nil
}

func (c *testSetupSecureClient) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.ca) == 0 {
		return nil, errNotSet
	}
	return c.ca, nil
}

// sdkConfig represents the configuration options available for the SDK client.
type sdkConfig struct {
	id     string
	rootCA struct {
		certificate []byte
		key         []byte
	}
	// TODO: replace by notBefore and notAfter
	validFrom             string // RFC3339, or relative time such as now-1m
	validFor              string // string parsable by time.ParseDuration
	crlDistributionPoints []string

	useDeviceIDInQuery bool
}

// Option interface used for setting optional sdkConfig properties.
type Option interface {
	apply(*sdkConfig)
}

type optionFunc func(*sdkConfig)

func (o optionFunc) apply(c *sdkConfig) {
	o(c)
}

// WithID creates Option that overrides the default device ID used by the SDK client.
func WithID(id string) Option {
	return optionFunc(func(cfg *sdkConfig) {
		cfg.id = id
	})
}

// WithID creates Option that overrides the certificate authority used by the SDK client.
func WithRootCA(certificate, key []byte) Option {
	return optionFunc(func(cfg *sdkConfig) {
		cfg.rootCA.certificate = certificate
		cfg.rootCA.key = key
	})
}

// WithValidity creates Option that overrides the ValidFrom timestamp and CertExpiry
// interval used by the SDK client when generating certificates.
func WithValidity(validFrom, validFor string) Option {
	return optionFunc(func(cfg *sdkConfig) {
		cfg.validFrom = validFrom
		cfg.validFor = validFor
	})
}

func WithUseDeviceIDInQuery(useDeviceIDInQuery bool) Option {
	return optionFunc(func(cfg *sdkConfig) {
		cfg.useDeviceIDInQuery = useDeviceIDInQuery
	})
}

func WithCRLDistributionPoints(crlDistributionPoints []string) Option {
	return optionFunc(func(cfg *sdkConfig) {
		cfg.crlDistributionPoints = crlDistributionPoints
	})
}

func getSDKConfig(opts ...Option) (*sdkConfig, error) {
	c := &sdkConfig{
		id: CertIdentity,
	}
	for _, opt := range opts {
		opt.apply(c)
	}

	var err error
	if c.rootCA.certificate == nil {
		c.rootCA.certificate, err = os.ReadFile(os.Getenv("TEST_ROOT_CA_CERT"))
		if err != nil {
			return nil, err
		}
	}
	if c.rootCA.key == nil {
		c.rootCA.key, err = os.ReadFile(os.Getenv("TEST_ROOT_CA_KEY"))
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func NewClient(opts ...Option) (*client.Client, error) {
	c, err := getSDKConfig(opts...)
	if err != nil {
		return nil, err
	}

	mfgTrustedCABlock, _ := pem.Decode(MfgTrustedCA)
	if mfgTrustedCABlock == nil {
		return nil, errors.New("mfgTrustedCABlock is empty")
	}
	mfgCA, err := x509.ParseCertificates(mfgTrustedCABlock.Bytes)
	if err != nil {
		return nil, err
	}

	identityIntermediateCA := c.rootCA.certificate
	identityIntermediateCAKey := c.rootCA.key
	identityTrustedCA := c.rootCA.certificate

	identityIntermediateCABlock, _ := pem.Decode(identityIntermediateCA)
	if identityIntermediateCABlock == nil {
		return nil, errors.New("identityIntermediateCABlock is empty")
	}
	identityIntermediateCAKeyBlock, _ := pem.Decode(identityIntermediateCAKey)
	if identityIntermediateCAKeyBlock == nil {
		return nil, errors.New("identityIntermediateCAKeyBlock is empty")
	}

	identityTrustedCABlock, _ := pem.Decode(identityTrustedCA)
	if identityTrustedCABlock == nil {
		return nil, errors.New("identityTrustedCABlock is empty")
	}
	identityTrustedCACert, err := x509.ParseCertificates(identityTrustedCABlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("cannot parse cert: %w", err)
	}
	mfgCert, err := tls.X509KeyPair(MfgCert, MfgKey)
	if err != nil {
		return nil, fmt.Errorf("cannot X509KeyPair: %w", err)
	}

	devCfg := &client.DeviceOwnershipSDKConfig{
		ID:                    c.id,
		Cert:                  string(identityIntermediateCA),
		CertKey:               string(identityIntermediateCAKey),
		ValidFrom:             c.validFrom,
		CRLDistributionPoints: c.crlDistributionPoints,
		CreateSignerFunc: func(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore, validNotAfter time.Time, crlDistributionPoints []string) core.CertificateSigner {
			return certificateSigner.NewIdentityCertificateSigner(caCert, caKey, certificateSigner.WithNotBefore(validNotBefore), certificateSigner.WithNotAfter(validNotAfter),
				certificateSigner.WithCRLDistributionPoints(crlDistributionPoints))
		},
	}
	if c.validFor != "" {
		devCfg.CertExpiry = &c.validFor
	}
	cfg := client.Config{
		DisablePeerTCPSignalMessageCSMs: true,
		DeviceOwnershipSDK:              devCfg,
		UseDeviceIDInQuery:              c.useDeviceIDInQuery,
	}

	client, err := client.NewClientFromConfig(&cfg, &testSetupSecureClient{
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
		ca:      identityTrustedCACert,
	}, log.Get().DTLSLoggerFactory().NewLogger("test"),
	)
	if err != nil {
		return nil, err
	}
	err = client.Initialization(context.Background())
	if err != nil {
		return nil, err
	}

	return client, nil
}

var (
	CertIdentity = "00000000-0000-0000-0000-000000000001"

	MfgCert = []byte(`-----BEGIN CERTIFICATE-----
MIIB9zCCAZygAwIBAgIRAOwIWPAt19w7DswoszkVIEIwCgYIKoZIzj0EAwIwEzER
MA8GA1UEChMIVGVzdCBPUkcwHhcNMTkwNTAyMjAwNjQ4WhcNMjkwMzEwMjAwNjQ4
WjBHMREwDwYDVQQKEwhUZXN0IE9SRzEyMDAGA1UEAxMpdXVpZDpiNWEyYTQyZS1i
Mjg1LTQyZjEtYTM2Yi0wMzRjOGZjOGVmZDUwWTATBgcqhkjOPQIBBggqhkjOPQMB
BwNCAAQS4eiM0HNPROaiAknAOW08mpCKDQmpMUkywdcNKoJv1qnEedBhWne7Z0jq
zSYQbyqyIVGujnI3K7C63NRbQOXQo4GcMIGZMA4GA1UdDwEB/wQEAwIDiDAzBgNV
HSUELDAqBggrBgEFBQcDAQYIKwYBBQUHAwIGCCsGAQUFBwMBBgorBgEEAYLefAEG
MAwGA1UdEwEB/wQCMAAwRAYDVR0RBD0wO4IJbG9jYWxob3N0hwQAAAAAhwR/AAAB
hxAAAAAAAAAAAAAAAAAAAAAAhxAAAAAAAAAAAAAAAAAAAAABMAoGCCqGSM49BAMC
A0kAMEYCIQDuhl6zj6gl2YZbBzh7Th0uu5izdISuU/ESG+vHrEp7xwIhANCA7tSt
aBlce+W76mTIhwMFXQfyF3awWIGjOcfTV8pU
-----END CERTIFICATE-----
`)

	MfgKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIMPeADszZajrkEy4YvACwcbR0pSdlKG+m8ALJ6lj/ykdoAoGCCqGSM49
AwEHoUQDQgAEEuHojNBzT0TmogJJwDltPJqQig0JqTFJMsHXDSqCb9apxHnQYVp3
u2dI6s0mEG8qsiFRro5yNyuwutzUW0Dl0A==
-----END EC PRIVATE KEY-----
`)

	MfgTrustedCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBaTCCAQ+gAwIBAgIQR33gIB75I7Vi/QnMnmiWvzAKBggqhkjOPQQDAjATMREw
DwYDVQQKEwhUZXN0IE9SRzAeFw0xOTA1MDIyMDA1MTVaFw0yOTAzMTAyMDA1MTVa
MBMxETAPBgNVBAoTCFRlc3QgT1JHMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
xbwMaS8jcuibSYJkCmuVHfeV3xfYVyUq8Iroz7YlXaTayspW3K4hVdwIsy/5U+3U
vM/vdK5wn2+NrWy45vFAJqNFMEMwDgYDVR0PAQH/BAQDAgEGMBMGA1UdJQQMMAoG
CCsGAQUFBwMBMA8GA1UdEwEB/wQFMAMBAf8wCwYDVR0RBAQwAoIAMAoGCCqGSM49
BAMCA0gAMEUCIBWkxuHKgLSp6OXDJoztPP7/P5VBZiwLbfjTCVRxBvwWAiEAnzNu
6gKPwtKmY0pBxwCo3NNmzNpA6KrEOXE56PkiQYQ=
-----END CERTIFICATE-----
`)
	MfgTrustedCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICzfC16AqtSv3wt+qIbrgM8dTqBhHANJhZS5xCpH6P2roAoGCCqGSM49
AwEHoUQDQgAExbwMaS8jcuibSYJkCmuVHfeV3xfYVyUq8Iroz7YlXaTayspW3K4h
VdwIsy/5U+3UvM/vdK5wn2+NrWy45vFAJg==
-----END EC PRIVATE KEY-----
`)
)
