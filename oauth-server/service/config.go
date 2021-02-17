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

var allowedGrantTypes = map[AllowedGrantType]bool{
	AllowedGrantType_AUTHORIZATION_CODE: true,
	AllowedGrantType_CLIENT_CREDENTIALS: true,
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
	AccessTokenType           AccessTokenType
	AllowedGrantTypes         AllowedGrantTypes
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
	_, ok := accessTokenTypes[c.AccessTokenType]
	if !ok {
		return fmt.Errorf("invalid AccessTokenType(%v)", c.AccessTokenType)
	}
	if len(c.AllowedGrantTypes) == 0 {
		return fmt.Errorf("invalid AllowedGrantTypes")
	}
	for i := range c.AllowedGrantTypes {
		_, ok := allowedGrantTypes[c.AllowedGrantTypes[i]]
		if !ok {
			return fmt.Errorf("AllowedGrantTypes[%v]: invalid grant type(%v)", i, c.AllowedGrantTypes[i])
		}
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

// Config represents application configuration
type Config struct {
	Address                      string
	Listen                       certManager.Config
	IDTokenPrivateKeyPath        string
	AccessTokenKeyPrivateKeyPath string
	Clients                      ClientsConfig
}

const ClientUI = "ui"
const ClientDevice = "device"
const ClientService = "service"

func (c *Config) SetDefaults() {
	if len(c.Clients) == 0 {
		c.Clients = ClientsConfig{
			{
				ID:                        ClientUI,
				AccessTokenType:           AccessTokenType_JWT,
				AllowedGrantTypes:         []AllowedGrantType{AllowedGrantType_AUTHORIZATION_CODE},
				AuthorizationCodeLifetime: time.Minute * 10,
				AccessTokenLifetime:       time.Hour,
			},
			{
				ID:                        ClientDevice,
				AccessTokenType:           AccessTokenType_REFERENCE,
				AllowedGrantTypes:         []AllowedGrantType{AllowedGrantType_AUTHORIZATION_CODE},
				AuthorizationCodeLifetime: time.Minute * 10,
				AccessTokenLifetime:       time.Hour,
			},
			{
				ID:                        ClientService,
				AccessTokenType:           AccessTokenType_JWT,
				AllowedGrantTypes:         []AllowedGrantType{AllowedGrantType_CLIENT_CREDENTIALS},
				AuthorizationCodeLifetime: time.Minute * 10,
				AccessTokenLifetime:       time.Hour,
			},
		}
	}

	for i := range c.Clients {
		c.Clients[i].SetDefaults()
	}
}

func (c *Config) Validate() error {
	if c.IDTokenPrivateKeyPath == "" {
		return fmt.Errorf("invalid IDTokenPrivateKeyPath")
	}
	if c.AccessTokenKeyPrivateKeyPath == "" {
		return fmt.Errorf("invalid AccessTokenKeyPrivateKeyPath")
	}
	if len(c.Clients) == 0 {
		return fmt.Errorf("invalid ClientsConfig")
	}
	for i := range c.Clients {
		err := c.Clients[i].Validate()
		if err != nil {
			return fmt.Errorf("ClientsConfig[%v]: %w", i, err)
		}
	}
	return nil
}

func (c Config) String() string {
	return config.ToString(c)
}
