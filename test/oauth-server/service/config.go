package service

import (
	"fmt"
	"github.com/plgd-dev/kit/log"
	"time"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certManager/server"
)

type AsymmetricKey struct {
	PrivatePath string
	PublicPath  string
}

type AccessTokenType string

const AccessTokenType_JWT AccessTokenType = "jwt"
const AccessTokenType_REFERENCE AccessTokenType = "reference"

var accessTokenTypes = map[AccessTokenType]bool{
	AccessTokenType_JWT:       true,
	AccessTokenType_REFERENCE: true,
}

type AllowedGrantType string

const AllowedGrantType_AUTHORIZATION_CODE AllowedGrantType = "authorization_code"
const AllowedGrantType_CLIENT_CREDENTIALS AllowedGrantType = "client_credentials"
const AllowedGrantType_PASSWORD AllowedGrantType = "password"

var allowedGrantTypes = map[AllowedGrantType]bool{
	AllowedGrantType_AUTHORIZATION_CODE: true,
	AllowedGrantType_CLIENT_CREDENTIALS: true,
	AllowedGrantType_PASSWORD:           true,
}

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
		return fmt.Errorf("invalid ID")
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
	Log         log.Config      `yaml:"log" json:"log" envconfig:"LOG"`
	Service	    APIsConfig      `yaml:"apis" json:"apis" envconfig:"SERVICE"`
}

type APIsConfig struct {
	Http HttpConfig `yaml:"http" json:"http" envconfig:"HTTP"`
}

type HttpConfig struct {
	Addr          string                `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:9100"`
	TLSConfig     server.Config         `yaml:"tls" json:"tls" envconfig:"TLS"`
	Domain                       string	`yaml:"domain" json:"domain" envconfig:"DOMAIN"`
	IDTokenPrivateKeyPath        string `yaml:"idTokenKeyPath" json:"idTokenKeyPath" envconfig:"ID_TOKEN_KEY_PATH"`
	AccessTokenKeyPrivateKeyPath string `yaml:"accessTokenKeyPath" json:"accessTokenKeyPath" envconfig:"ACCESS_TOKEN_KEY_PATH"`
}


const ClientTest = "test"

func (c *Config) SetDefaults() {
}

func (c *Config) Validate() error {
	if c.Service.Http.IDTokenPrivateKeyPath == "" {
		return fmt.Errorf("invalid IDTokenPrivateKeyPath")
	}
	if c.Service.Http.AccessTokenKeyPrivateKeyPath == "" {
		return fmt.Errorf("invalid AccessTokenKeyPrivateKeyPath")
	}
	if c.Service.Http.Domain == "" {
		return fmt.Errorf("invalid Domain")
	}
	return nil
}

func (c Config) String() string {
	return config.ToString(c)
}
