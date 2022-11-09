package server

import (
	"crypto/tls"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/general"
	"github.com/plgd-dev/hub/v2/pkg/strings"
)

// Config provides configuration of a file based Server Certificate manager. CAPool can be a string or an array of strings.
type Config struct {
	CAPool                    interface{} `yaml:"caPool" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyFile                   string      `yaml:"keyFile" json:"keyFile" description:"file name of private key in PEM format"`
	CertFile                  string      `yaml:"certFile" json:"certFile" description:"file name of certificate in PEM format"`
	ClientCertificateRequired bool        `yaml:"clientCertificateRequired" json:"clientCertificateRequired" description:"require client certificate"`
	CAPoolIsOptional          bool        `yaml:"-" json:"-"`
	CAPoolArray               []string    `yaml:"-" json:"-"`
	validated                 bool
}

func (c *Config) Validate() error {
	caPoolArray, ok := strings.ToStringArray(c.CAPool)
	if !ok {
		return fmt.Errorf("caPool('%v')", c.CAPool)
	}
	c.CAPoolArray = caPoolArray
	if !c.CAPoolIsOptional && len(caPoolArray) == 0 {
		return fmt.Errorf("caPool('%v')", c.CAPool)
	}
	if c.CertFile == "" {
		return fmt.Errorf("certFile('%v')", c.CertFile)
	}
	if c.KeyFile == "" {
		return fmt.Errorf("keyFile('%v')", c.KeyFile)
	}
	c.validated = true
	return nil
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
func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*CertManager, error) {
	if !config.validated {
		if err := config.Validate(); err != nil {
			return nil, err
		}
	}
	c, err := general.New(general.Config{
		CAPool:                    config.CAPoolArray,
		KeyFile:                   config.KeyFile,
		CertFile:                  config.CertFile,
		ClientCertificateRequired: config.ClientCertificateRequired,
		UseSystemCAPool:           false,
	}, fileWatcher, logger)
	if err != nil {
		return nil, err
	}
	return &CertManager{
		c: c,
	}, nil
}
