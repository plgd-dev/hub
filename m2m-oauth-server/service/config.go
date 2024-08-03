package service

import (
	"fmt"
	"net"

	oauthsigner "github.com/plgd-dev/hub/v2/m2m-oauth-server/oauthSigner"
	grpcService "github.com/plgd-dev/hub/v2/m2m-oauth-server/service/grpc"
	storeConfig "github.com/plgd-dev/hub/v2/m2m-oauth-server/store/config"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
)

type ClientsConfig struct {
	Storage                storeConfig.Config                `yaml:"storage" json:"storage"`
	OpenTelemetryCollector http.OpenTelemetryCollectorConfig `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	return nil
}

// Config represents application configuration
type Config struct {
	Log         log.Config         `yaml:"log" json:"log"`
	APIs        APIsConfig         `yaml:"apis" json:"apis"`
	Clients     ClientsConfig      `yaml:"clients" json:"clients"`
	OAuthSigner oauthsigner.Config `yaml:"oauthSigner" json:"oauthSigner"`
}

func (c *Config) String() string {
	return config.ToString(c)
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if err := c.OAuthSigner.Validate(); err != nil {
		return fmt.Errorf("oauthSigner.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	HTTP HTTPConfig         `yaml:"http" json:"http"`
	GRPC grpcService.Config `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type HTTPConfig struct {
	Addr   string        `yaml:"address" json:"address"`
	Server server.Config `yaml:",inline" json:",inline"`
}

func (c *HTTPConfig) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", c.Addr); err != nil {
		return fmt.Errorf("address('%v') - %w", c.Addr, err)
	}
	return nil
}
