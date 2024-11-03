package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/general"
	pkgTls "github.com/plgd-dev/hub/v2/pkg/security/tls"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"go.opentelemetry.io/otel/trace"
)

// Config provides configuration of a file based Server Certificate manager. CAPool can be a string or an array of strings.
type Config struct {
	CAPool                    interface{}         `yaml:"caPool" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyFile                   urischeme.URIScheme `yaml:"keyFile" json:"keyFile" description:"file name of private key in PEM format"`
	CertFile                  urischeme.URIScheme `yaml:"certFile" json:"certFile" description:"file name of certificate in PEM format"`
	ClientCertificateRequired bool                `yaml:"clientCertificateRequired" json:"clientCertificateRequired" description:"require client certificate"`
	CRL                       pkgTls.CRLConfig    `yaml:"crl" json:"crl"`

	CAPoolIsOptional bool                  `yaml:"-" json:"-"`
	caPoolArray      []urischeme.URIScheme `yaml:"-" json:"-"`
	validated        bool
}

func (c *Config) Validate() error {
	if c.validated {
		return nil
	}
	caPoolArray, ok := strings.ToStringArray(c.CAPool)
	if !ok {
		return fmt.Errorf("caPool('%v') - unsupported", c.CAPool)
	}
	c.caPoolArray = urischeme.ToURISchemeArray(caPoolArray)
	if !c.CAPoolIsOptional && len(caPoolArray) == 0 {
		return fmt.Errorf("caPool('%v') - is empty", c.CAPool)
	}
	if c.CertFile == "" {
		return fmt.Errorf("certFile('%v')", c.CertFile)
	}
	if c.KeyFile == "" {
		return fmt.Errorf("keyFile('%v')", c.KeyFile)
	}
	if err := c.CRL.Validate(); err != nil {
		return fmt.Errorf("CRL configuration is invalid: %w", err)
	}
	c.validated = true
	return nil
}

func (c *Config) CAPoolArray() ([]urischeme.URIScheme, error) {
	if !c.validated {
		return nil, errors.New("call Validate() first")
	}
	return c.caPoolArray, nil
}

// CertManager holds certificates from filesystem watched for changes
type CertManager struct {
	c *general.CertManager
}

// GetTLSConfig returns tls configuration for clients
func (c *CertManager) GetTLSConfig() *tls.Config {
	return c.c.GetServerTLSConfig()
}

func (c *CertManager) VerifyByCRL(ctx context.Context, certificate *x509.Certificate, cdp []string) error {
	return c.c.VerifyByCRL(ctx, certificate, cdp)
}

// Close ends watching certificates
func (c *CertManager) Close() {
	c.c.Close()
}

// New creates a new certificate manager which watches for certs in a filesystem
func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...general.SetOption) (*CertManager, error) {
	if !config.validated {
		if err := config.Validate(); err != nil {
			return nil, err
		}
	}
	cfg := general.Config{
		CAPool:                    config.caPoolArray,
		CAPoolIsOptional:          config.CAPoolIsOptional,
		KeyFile:                   config.KeyFile,
		CertFile:                  config.CertFile,
		ClientCertificateRequired: config.ClientCertificateRequired,
		UseSystemCAPool:           false,
		CRL:                       config.CRL,
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	c, err := general.New(cfg, fileWatcher, logger.With(log.CertManagerKey, "server"), tracerProvider, opts...)
	if err != nil {
		return nil, err
	}
	return &CertManager{
		c: c,
	}, nil
}
