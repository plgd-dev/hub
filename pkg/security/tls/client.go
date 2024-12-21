package tls

import (
	"errors"
	"fmt"
	"slices"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/strings"
)

// ClientConfig provides configuration of a file based Server Certificate manager. CAPool can be a string or an array of strings.
type ClientConfig struct {
	CAPool          interface{}         `yaml:"caPool" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyFile         urischeme.URIScheme `yaml:"keyFile" json:"keyFile" description:"file name of private key in PEM format"`
	CertFile        urischeme.URIScheme `yaml:"certFile" json:"certFile" description:"file name of certificate in PEM format"`
	UseSystemCAPool bool                `yaml:"useSystemCAPool" json:"useSystemCaPool" description:"use system certification pool"`
	CRL             CRLConfig           `yaml:"crl" json:"crl"`

	caPoolArray []urischeme.URIScheme `yaml:"-" json:"-"`
	validated   bool
}

func (c *ClientConfig) Validate() error {
	if c.validated {
		return nil
	}
	caPoolArray, ok := strings.ToStringArray(c.CAPool)
	if !ok {
		return fmt.Errorf("caPool('%v') - unsupported", c.CAPool)
	}
	c.caPoolArray = urischeme.ToURISchemeArray(caPoolArray)
	if !c.UseSystemCAPool && len(c.caPoolArray) == 0 {
		return fmt.Errorf("caPool('%v') - is empty", c.CAPool)
	}
	if err := c.CRL.Validate(); err != nil {
		return fmt.Errorf("CRL configuration is invalid: %w", err)
	}
	c.validated = true
	return nil
}

func (c *ClientConfig) CAPoolArray() ([]urischeme.URIScheme, error) {
	if !c.validated {
		return nil, errors.New("call Validate() first")
	}
	return c.caPoolArray, nil
}

func (c *ClientConfig) CAPoolFilePathArray() ([]string, error) {
	a, err := c.CAPoolArray()
	if err != nil {
		return nil, err
	}
	return urischeme.ToFilePathArray(a), nil
}

func (c *ClientConfig) Equals(c2 ClientConfig) bool {
	caPool1, ok1 := strings.ToStringArray(c.CAPool)
	if !ok1 {
		return false
	}
	caPool2, ok2 := strings.ToStringArray(c2.CAPool)
	if !ok2 {
		return false
	}
	return slices.Equal(caPool1, caPool2) &&
		c.KeyFile == c2.KeyFile &&
		c.CertFile == c2.CertFile &&
		c.UseSystemCAPool == c2.UseSystemCAPool &&
		c.CRL.Equals(c2.CRL)
}
