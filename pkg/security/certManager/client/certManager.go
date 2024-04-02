package client

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/general"
	"github.com/plgd-dev/hub/v2/pkg/strings"
)

// Config provides configuration of a file based Server Certificate manager. CAPool can be a string or an array of strings.
type Config struct {
	CAPool          interface{}           `yaml:"caPool" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyFile         urischeme.URIScheme   `yaml:"keyFile" json:"keyFile" description:"file name of private key in PEM format"`
	CertFile        urischeme.URIScheme   `yaml:"certFile" json:"certFile" description:"file name of certificate in PEM format"`
	UseSystemCAPool bool                  `yaml:"useSystemCAPool" json:"useSystemCaPool" description:"use system certification pool"`
	caPoolArray     []urischeme.URIScheme `yaml:"-" json:"-"`
	validated       bool
}

func (c *Config) Validate() error {
	caPoolArray, ok := strings.ToStringArray(c.CAPool)
	if !ok {
		return fmt.Errorf("caPool('%v') - unsupported", c.CAPool)
	}
	c.caPoolArray = urischeme.ToURISchemeArray(caPoolArray)
	if !c.UseSystemCAPool && len(c.caPoolArray) == 0 {
		return fmt.Errorf("caPool('%v') - is empty", c.CAPool)
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

func (c *Config) CAPoolArray() ([]urischeme.URIScheme, error) {
	if !c.validated {
		return nil, errors.New("call Validate() first")
	}
	return c.caPoolArray, nil
}

func (c *Config) CAPoolFilePathArray() ([]string, error) {
	a, err := c.CAPoolArray()
	if err != nil {
		return nil, err
	}
	return urischeme.ToFilePathArray(a), nil
}

// CertManager holds certificates from filesystem watched for changes
type CertManager struct {
	c *general.CertManager
}

// GetTLSConfig returns tls configuration for clients
func (c *CertManager) GetTLSConfig() *tls.Config {
	return c.c.GetClientTLSConfig()
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
		CAPool:                    config.caPoolArray,
		KeyFile:                   config.KeyFile,
		CertFile:                  config.CertFile,
		ClientCertificateRequired: false,
		UseSystemCAPool:           config.UseSystemCAPool,
	}, fileWatcher, logger.With(log.CertManagerKey, "client"))
	if err != nil {
		return nil, err
	}
	return &CertManager{
		c: c,
	}, nil
}
