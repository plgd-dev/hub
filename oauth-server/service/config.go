package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certManager"
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
	Address                      string
	Listen                       certManager.Config
	IDTokenPrivateKeyPath        string
	AccessTokenKeyPrivateKeyPath string
}

const ClientTest = "test"

func (c *Config) SetDefaults() {
}

func (c *Config) Validate() error {
	if c.IDTokenPrivateKeyPath == "" {
		return fmt.Errorf("invalid IDTokenPrivateKeyPath")
	}
	if c.AccessTokenKeyPrivateKeyPath == "" {
		return fmt.Errorf("invalid AccessTokenKeyPrivateKeyPath")
	}
	return nil
}

func (c Config) String() string {
	return config.ToString(c)
}
