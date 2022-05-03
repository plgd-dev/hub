package service

import (
	"fmt"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
)

type Config struct {
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Signer  SignerConfig  `yaml:"signer" json:"signer"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Signer.Validate(); err != nil {
		return fmt.Errorf("signer.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC server.Config `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

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

type ClientsConfig struct {
	OpenTelemetryCollector otelClient.Config `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
