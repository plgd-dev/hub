package service

import (
	"fmt"

	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	httpClient "github.com/plgd-dev/cloud/pkg/net/http/client"
	"github.com/plgd-dev/cloud/pkg/net/listener"
	"github.com/plgd-dev/kit/config"
)

// Config provides defaults and enables configuring via env variables.
type Config struct {
	Log          log.Config         `yaml:"log" json:"log"`
	APIs         APIsConfig         `yaml:"apis" json:"apis"`
	Clients      ClientsConfig      `yaml:"clients" json:"clients"`
	OAuthClients OAuthClientsConfig `yaml:"oauthClients" json:"oauthClients"`
}

func (c *Config) Validate() error {
	err := c.Clients.Validate()
	if err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	err = c.APIs.Validate()
	if err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	err = c.OAuthClients.Validate()
	if err != nil {
		return fmt.Errorf("oauthClients.%w", err)
	}
	return nil
}

type APIsConfig struct {
	GRPC server.Config   `yaml:"grpc" json:"grpc"`
	HTTP listener.Config `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	err := c.GRPC.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	err = c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type ClientsConfig struct {
	Storage StorageConfig `yaml:"storage" json:"storage"`
}

func (c *ClientsConfig) Validate() error {
	err := c.Storage.Validate()
	if err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	return nil
}

type OAuthClientsConfig struct {
	Device provider.Config `yaml:"device" json:"device"`
	SDK    SDKOAuthConfig  `yaml:"client" json:"client"`
}

func (c *OAuthClientsConfig) Validate() error {
	err := c.Device.Validate()
	if err != nil {
		return fmt.Errorf("device.%w", err)
	}
	err = c.SDK.Validate()
	if err != nil {
		return fmt.Errorf("client.%w", err)
	}
	return nil
}

type SDKOAuthConfig struct {
	oauth.Config `yaml:",inline" json:",inline"`
	HTTP         httpClient.Config `yaml:"http" json:"http"`
}

func (c *SDKOAuthConfig) Validate() error {
	err := c.Config.Validate()
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	err = c.HTTP.Validate()
	if err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type StorageConfig struct {
	MongoDB mongodb.Config `yaml:"mongoDB" json:"mongoDB"`
}

func (c *StorageConfig) Validate() error {
	err := c.MongoDB.Validate()
	if err != nil {
		return fmt.Errorf("mongoDB.%w", err)
	}
	return nil
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
