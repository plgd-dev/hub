package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/listener"
)

type AsymmetricKey struct {
	PrivateFile string
	PublicFile  string
}

type AccessTokenType string

const AccessTokenType_JWT AccessTokenType = "jwt"
const AccessTokenType_REFERENCE AccessTokenType = "reference"

type AllowedGrantType string

const AllowedGrantType_AUTHORIZATION_CODE AllowedGrantType = "authorization_code"
const AllowedGrantType_CLIENT_CREDENTIALS AllowedGrantType = "client_credentials"
const AllowedGrantType_PASSWORD AllowedGrantType = "password"

type AllowedGrantTypes []AllowedGrantType

func (gt AllowedGrantTypes) IsAllowed(v AllowedGrantType) bool {
	for _, t := range gt {
		if v == t {
			return true
		}
	}
	return false
}

type Client struct {
	ID                        string
	AuthorizationCodeLifetime time.Duration
	AccessTokenLifetime       time.Duration
}

func (c *Client) SetDefaults() {
	if c.AuthorizationCodeLifetime <= 0 {
		c.AuthorizationCodeLifetime = time.Minute * 10
	}
}

func (c *Client) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("id('%v')", c.ID)
	}
	return nil
}

type ClientsConfig []*Client

func (c ClientsConfig) Find(id string) *Client {
	for _, client := range c {
		if client.ID == id {
			return client
		}
	}
	return nil
}

var clients = ClientsConfig{
	{
		ID:                        ClientTest,
		AuthorizationCodeLifetime: time.Minute * 10,
		AccessTokenLifetime:       time.Hour,
	},
}

// Config represents application configuration
type Config struct {
	Log         log.Config        `yaml:"log" json:"log"`
	APIs        APIsConfig        `yaml:"apis" json:"apis"`
	OAuthSigner OAuthSignerConfig `yaml:"oauthSigner" json:"oauthSigner"`
}

func (c *Config) Validate() error {
	err := c.APIs.Validate()
	if err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	err = c.OAuthSigner.Validate()
	if err != nil {
		return fmt.Errorf("oauthSigner.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	HTTP listener.Config `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	err := c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type OAuthSignerConfig struct {
	IDTokenKeyFile     string `yaml:"idTokenKeyFile" json:"idTokenKeyFile"`
	AccessTokenKeyFile string `yaml:"accessTokenKeyFile" json:"accessTokenKeyFile"`
	Domain             string `yaml:"domain" json:"domain"`
}

const ClientTest = "test"

func (c *OAuthSignerConfig) Validate() error {
	if c.IDTokenKeyFile == "" {
		return fmt.Errorf("idTokenKeyFile('%v')", c.IDTokenKeyFile)
	}
	if c.AccessTokenKeyFile == "" {
		return fmt.Errorf("accessTokenKeyFile('%v')", c.AccessTokenKeyFile)
	}
	if c.Domain == "" {
		return fmt.Errorf("domain('%v')", c.Domain)
	}
	return nil
}

func (c Config) String() string {
	return config.ToString(c)
}
