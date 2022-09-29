package service

import (
	"fmt"
	"net"

	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	httpServer "github.com/plgd-dev/hub/v2/pkg/net/http/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
)

type Config struct {
	Log     log.Config               `yaml:"log" json:"log"`
	APIs    APIsConfig               `yaml:"apis" json:"apis"`
	Signer  grpcService.SignerConfig `yaml:"signer" json:"signer"`
	Clients ClientsConfig            `yaml:"clients" json:"clients"`
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
	GRPC grpcService.Config `yaml:"grpc" json:"grpc"`
	HTTP HTTPConfig         `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type HTTPConfig struct {
	Addr   string            `yaml:"address" json:"address"`
	Server httpServer.Config `yaml:",inline" json:",inline"`
}

func (c *HTTPConfig) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", c.Addr); err != nil {
		return fmt.Errorf("address('%v') - %w", c.Addr, err)
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

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
