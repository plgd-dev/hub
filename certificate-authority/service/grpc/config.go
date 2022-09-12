package grpc

import (
	"fmt"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
)

type Config = server.Config

type SignerConfig struct {
	KeyFile   string        `yaml:"keyFile" json:"keyFile" description:"file name of CA private key in PEM format"`
	CertFile  string        `yaml:"certFile" json:"certFile" description:"file name of CA certificate in PEM format"`
	ValidFrom string        `yaml:"validFrom" json:"validFrom" description:"format https://github.com/karrick/tparse"`
	ExpiresIn time.Duration `yaml:"expiresIn" json:"expiresIn"`
	HubID     string        `yaml:"hubID" json:"hubId"`
}

func (c *SignerConfig) Validate() error {
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
	if c.HubID == "" {
		return fmt.Errorf("hubID('%v')", c.HubID)
	}
	return nil
}
