package oauthsigner

import (
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
)

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
	Owner               string                 `yaml:"owner"`
	AccessTokenLifetime time.Duration          `yaml:"accessTokenLifetime"`
	AllowedGrantTypes   []GrantType            `yaml:"allowedGrantTypes"`
	AllowedAudiences    []string               `yaml:"allowedAudiences"`
	AllowedScopes       []string               `yaml:"allowedScopes"`
	JWTPrivateKey       PrivateKeyJWTConfig    `yaml:"jwtPrivateKey"`
	InsertTokenClaims   map[string]interface{} `yaml:"insertTokenClaims"`

	// runtime
	Secret string `yaml:"-"`
}

func (c *Client) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("id('%v')", c.ID)
	}
	if !c.JWTPrivateKey.Enabled {
		if c.SecretFile == "" {
			return fmt.Errorf("secretFile('%v') - one of [secretFile, privateKeyJWT] need to be set", c.SecretFile)
		}
		if c.Owner == "" {
			return fmt.Errorf("owner('%v')", c.Owner)
		}
		data, err := c.SecretFile.Read()
		if err != nil {
			return fmt.Errorf("secretFile('%v') - %w", c.SecretFile, err)
		}
		c.Secret = string(data)
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

type Config struct {
	PrivateKeyFile urischeme.URIScheme `yaml:"privateKeyFile" json:"privateKeyFile"`
	Domain         string              `yaml:"domain" json:"domain"`
	OwnerClaim     string              `yaml:"ownerClaim" json:"ownerClaim"`
	DeviceIDClaim  string              `yaml:"deviceIDClaim" json:"deviceIDClaim"`
	Clients        OAuthClientsConfig  `yaml:"clients" json:"clients"`
}

func (c *Config) GetDomain() string {
	return "https://" + c.Domain
}

func (c *Config) GetAuthority() string {
	return c.GetDomain() + uri.Base
}

func (c *Config) Validate() error {
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
