package service

import (
	"fmt"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
)

type Config struct {
	Log    log.Config   `yaml:"log" json:"log"`
	APIs   APIsConfig   `yaml:"apis" json:"apis"`
	Signer SignerConfig `yaml:"signer" json:"signer"`
}

func (c *Config) Validate() error {
	err := c.APIs.Validate()
	if err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	err = c.Signer.Validate()
	if err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC server.Config `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	err := c.GRPC.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type SignerConfig struct {
	KeyFile   string        `yaml:"keyFile" json:"keyFile" description:"file name of CA private key in PEM format"`
	CertFile  string        `yaml:"certFile" json:"certFile" description:"file name of CA certificate in PEM format"`
	ValidFrom string        `yaml:"validFrom" json:"validFrom" description:"format https://github.com/karrick/tparse"`
	ExpiresIn time.Duration `yaml:"expiresIn" json:"expiresIn"`
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
	return nil
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
