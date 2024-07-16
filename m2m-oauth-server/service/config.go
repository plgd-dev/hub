package service

import (
	"fmt"
	"time"

	storeConfig "github.com/plgd-dev/hub/v2/m2m-oauth-server/store/config"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
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

type PrivateKeyJWTConfig struct {
	Enabled       bool             `yaml:"enabled"`
	Authorization validator.Config `yaml:"authorization,omitempty"`
}

func (c *PrivateKeyJWTConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization.%w", err)
	}
	return nil
}

type Client struct {
	ID                  string                 `yaml:"id"`
	SecretFile          urischeme.URIScheme    `yaml:"secretFile"`
	AccessTokenLifetime time.Duration          `yaml:"accessTokenLifetime"`
	AllowedGrantTypes   []GrantType            `yaml:"allowedGrantTypes"`
	AllowedAudiences    []string               `yaml:"allowedAudiences"`
	AllowedScopes       []string               `yaml:"allowedScopes"`
	JWTPrivateKey       PrivateKeyJWTConfig    `yaml:"jwtPrivateKey"`
	InsertTokenClaims   map[string]interface{} `yaml:"insertTokenClaims"`

	// runtime
	secret string `yaml:"-"`
}

func (c *Client) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("id('%v')", c.ID)
	}
	if !c.JWTPrivateKey.Enabled {
		if c.SecretFile == "" {
			return fmt.Errorf("secretFile('%v') - one of [secretFile, privateKeyJWT] need to be set", c.SecretFile)
		}
		data, err := c.SecretFile.Read()
		if err != nil {
			return fmt.Errorf("secretFile('%v') - %w", c.SecretFile, err)
		}
		c.secret = string(data)
	}
	if len(c.AllowedGrantTypes) == 0 {
		return fmt.Errorf("allowedGrantTypes('%v') - is empty", c.AllowedGrantTypes)
	}
	for _, gt := range c.AllowedGrantTypes {
		switch gt {
		case GrantTypeClientCredentials:
		default:
			return fmt.Errorf("allowedGrantTypes('%v') - only %v is supported", c.AllowedGrantTypes, GrantTypeClientCredentials)
		}
	}
	if err := c.JWTPrivateKey.Validate(); err != nil {
		return fmt.Errorf("privateKeyJWT.%w", err)
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
	PrivateKeyFile urischeme.URIScheme `yaml:"privateKeyFile" json:"privateKeyFile"`
	Domain         string              `yaml:"domain" json:"domain"`
	OwnerClaim     string              `yaml:"ownerClaim" json:"ownerClaim"`
	DeviceIDClaim  string              `yaml:"deviceIDClaim" json:"deviceIDClaim"`
	Clients        OAuthClientsConfig  `yaml:"clients" json:"clients"`
}

func (c *OAuthSignerConfig) Validate() error {
	if c.PrivateKeyFile == "" {
		return fmt.Errorf("privateKeyFile('%v')", c.PrivateKeyFile)
	}
	if c.Domain == "" {
		return fmt.Errorf("domain('%v')", c.Domain)
	}
	if len(c.Clients) == 0 {
		return fmt.Errorf("clients('%v')", c.Clients)
	}
	for idx, client := range c.Clients {
		if err := client.Validate(); err != nil {
			return fmt.Errorf("clients[%v].%w", idx, err)
		}
	}
	return nil
}

func (c Config) String() string {
	return config.ToString(c)
}
