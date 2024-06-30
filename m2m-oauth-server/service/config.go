package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
)

type AsymmetricKey struct {
	PrivateFile string
	PublicFile  string
}

type AccessTokenType string

const AccessTokenType_JWT AccessTokenType = "jwt"

type GrantType string

const (
	GrantTypeClientCredentials GrantType = "client_credentials"
)

type Authority struct{}

type Client struct {
	ID                  string        `yaml:"id"`
	ClientSecret        string        `yaml:"secret"`
	RequireDeviceID     bool          `yaml:"requireDeviceID"`
	RequireOwner        bool          `yaml:"requireOwner"`
	AccessTokenLifetime time.Duration `yaml:"accessTokenLifetime"`
	AllowedGrantTypes   []GrantType   `yaml:"allowedGrantTypes"`
	AllowedAudiences    []string      `yaml:"allowedAudiences"`
	AllowedScopes       []string      `yaml:"allowedScopes"`
}

func (c *Client) Validate(ownerClaim, deviceIDClaim string) error {
	if c.ID == "" {
		return fmt.Errorf("id('%v')", c.ID)
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("secret('%v')", c.ClientSecret)
	}
	if len(c.AllowedGrantTypes) == 0 {
		return fmt.Errorf("allowedGrantTypes('%v')", c.AllowedGrantTypes)
	}
	for _, gt := range c.AllowedGrantTypes {
		switch gt {
		case GrantTypeClientCredentials:
		default:
			return fmt.Errorf("allowedGrantTypes('%v') - only %v is supported", c.AllowedGrantTypes, GrantTypeClientCredentials)
		}
	}
	if c.RequireDeviceID && deviceIDClaim == "" {
		return fmt.Errorf("requireDeviceID('%v') - oauthSigner.deviceIDClaim('%v') is empty", c.RequireDeviceID, deviceIDClaim)
	}
	if c.RequireOwner && ownerClaim == "" {
		return fmt.Errorf("requireOwner('%v') - oauthSigner.ownerClaim('%v') is empty", c.RequireOwner, ownerClaim)
	}
	return nil
}

type OAuthClientsConfig []*Client

func (c OAuthClientsConfig) Find(id string) *Client {
	for _, client := range c {
		if client.ID == id {
			return client
		}
	}
	return nil
}

type ClientsConfig struct {
	OpenTelemetryCollector http.OpenTelemetryCollectorConfig `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
}

// Config represents application configuration
type Config struct {
	Log         log.Config        `yaml:"log" json:"log"`
	APIs        APIsConfig        `yaml:"apis" json:"apis"`
	Clients     ClientsConfig     `yaml:"clients" json:"clients"`
	OAuthSigner OAuthSignerConfig `yaml:"oauthSigner" json:"oauthSigner"`
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
	HTTP HTTPConfig `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type HTTPConfig struct {
	Connection listener.Config `yaml:",inline" json:",inline"`
	Server     server.Config   `yaml:",inline" json:",inline"`
}

func (c *HTTPConfig) Validate() error {
	return c.Connection.Validate()
}

type OAuthSignerConfig struct {
	AccessTokenKeyFile urischeme.URIScheme `yaml:"accessTokenKeyFile" json:"accessTokenKeyFile"`
	Domain             string              `yaml:"domain" json:"domain"`
	OwnerClaim         string              `yaml:"ownerClaim" json:"ownerClaim"`
	DeviceIDClaim      string              `yaml:"deviceIDClaim" json:"deviceIDClaim"`
	Clients            OAuthClientsConfig  `yaml:"clients" json:"clients"`
}

func (c *OAuthSignerConfig) Validate() error {
	if c.AccessTokenKeyFile == "" {
		return fmt.Errorf("accessTokenKeyFile('%v')", c.AccessTokenKeyFile)
	}
	if c.Domain == "" {
		return fmt.Errorf("domain('%v')", c.Domain)
	}
	if len(c.Clients) == 0 {
		return fmt.Errorf("clients('%v')", c.Clients)
	}
	return nil
}

func (c Config) String() string {
	return config.ToString(c)
}
