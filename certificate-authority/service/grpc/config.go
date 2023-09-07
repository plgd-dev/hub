package grpc

import (
	"fmt"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"gopkg.in/yaml.v3"
)

type Config = server.Config

type SignerConfig struct {
	CAPool    interface{}   `yaml:"caPool" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyFile   string        `yaml:"keyFile" json:"keyFile" description:"file name of CA private key in PEM format"`
	CertFile  string        `yaml:"certFile" json:"certFile" description:"file name of CA certificate in PEM format"`
	ValidFrom string        `yaml:"validFrom" json:"validFrom" description:"format https://github.com/karrick/tparse"`
	ExpiresIn time.Duration `yaml:"expiresIn" json:"expiresIn"`

	caPoolArray []string `yaml:"-" json:"-"`
}

func (c *SignerConfig) Validate() error {
	caPoolArray, ok := strings.ToStringArray(c.CAPool)
	if !ok {
		return fmt.Errorf("caPool('%v')", c.CAPool)
	}
	c.caPoolArray = caPoolArray
	if c.CertFile == "" {
		return fmt.Errorf("certFile('%v')", c.CertFile)
	}
	if c.KeyFile == "" {
		return fmt.Errorf("keyFile('%v')", c.KeyFile)
	}
	if c.ExpiresIn <= 0 {
		return fmt.Errorf("expiresIn('%v')", c.KeyFile)
	}
	_, err := tparse.ParseNow(time.RFC3339, c.ValidFrom)
	if err != nil {
		return fmt.Errorf("validFrom('%v')", c.ValidFrom)
	}

	return nil
}

func (c *SignerConfig) String() string {
	d, err := yaml.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(d)
}
