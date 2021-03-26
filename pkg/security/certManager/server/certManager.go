package server

import (
	"crypto/tls"
	"fmt"

	"github.com/plgd-dev/cloud/pkg/security/certManager/general"
	"go.uber.org/zap"
)

// Config provides configuration of a file based Server Certificate manager
type Config struct {
	CAPool                    string `yaml:"caPool" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyFile                   string `yaml:"keyFile" json:"keyFile" description:"file name of private key in PEM format"`
	CertFile                  string `yaml:"certFile" json:"certFile" description:"file name of certificate in PEM format"`
	ClientCertificateRequired bool   `yaml:"clientCertificateRequired" json:"clientCertificateRequired" description:"require client certificate"`
}

func (c Config) Validate() error {
	if c.CAPool == "" {
		return fmt.Errorf("caPool")
	}
	if c.CertFile == "" {
		return fmt.Errorf("certFile")
	}
	if c.KeyFile == "" {
		return fmt.Errorf("keyFile")
	}
	return nil
}

func (c *Config) SetDefaults() {
	c.ClientCertificateRequired = true
}

// CertManager holds certificates from filesystem watched for changes
type CertManager struct {
	c *general.CertManager
}

// GetTLSConfig returns tls configuration for clients
func (c *CertManager) GetTLSConfig() *tls.Config {
	return c.c.GetServerTLSConfig()
}

// Close ends watching certificates
func (c *CertManager) Close() {
	c.c.Close()
}

// New creates a new certificate manager which watches for certs in a filesystem
func New(config Config, logger *zap.Logger) (*CertManager, error) {

	c, err := general.New(general.Config{
		CAPool:                    config.CAPool,
		KeyFile:                   config.KeyFile,
		CertFile:                  config.CertFile,
		ClientCertificateRequired: config.ClientCertificateRequired,
		UseSystemCAPool:           false,
	}, logger)
	if err != nil {
		return nil, err
	}
	return &CertManager{
		c: c,
	}, nil
}
