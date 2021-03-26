package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	grpcServer "github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	client2 "github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	eventbusConfig "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/config"
	eventstoreConfig "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/config"
)

//Config represent application configuration
type Config struct {
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
}

func (c *Config) Validate() error {
	err := c.APIs.Validate()
	if err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	err = c.Clients.Validate()
	if err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	return nil
}

type APIsConfig struct {
	GRPC GrpcConfig `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	err := c.GRPC.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type GrpcConfig struct {
	Server       grpcServer.Config  `yaml:",inline" json:",inline"`
	Capabilities CapabilitiesConfig `yaml:"capabilities" json:"capabilities"`
}

func (c *GrpcConfig) Validate() error {
	err := c.Server.Validate()
	if err != nil {
		return err
	}
	err = c.Capabilities.Validate()
	if err != nil {
		return fmt.Errorf("capabilities.%w", err)
	}
	return nil
}

type CapabilitiesConfig struct {
	SnapshotThreshold               int           `yaml:"snapshotThreshold" json:"snapshotThreshold" default:"16"`
	ConcurrencyExceptionMaxRetry    int           `yaml:"occMaxRetry" json:"occMaxRetry" default:"8"`
	UserDevicesManagerTickFrequency time.Duration `yaml:"userMgmtTickFrequency" json:"userMgmtTickFrequency" default:"15s"`
	UserDevicesManagerExpiration    time.Duration `yaml:"userMgmtExpiration" json:"userMgmtExpiration" default:"1m"`
}

func (c *CapabilitiesConfig) Validate() error {
	if c.SnapshotThreshold <= 0 {
		return fmt.Errorf("snapshotThreshold('%v')", c.SnapshotThreshold)
	}
	if c.ConcurrencyExceptionMaxRetry <= 0 {
		return fmt.Errorf("occMaxRetry('%v')", c.ConcurrencyExceptionMaxRetry)
	}
	if c.UserDevicesManagerTickFrequency <= 0 {
		return fmt.Errorf("userMgmtTickFrequency('%v')", c.UserDevicesManagerTickFrequency)
	}
	if c.UserDevicesManagerExpiration <= 0 {
		return fmt.Errorf("userMgmtExpiration('%v')", c.UserDevicesManagerExpiration)
	}
	return nil
}

type ClientsConfig struct {
	Eventbus      eventbusConfig.Config   `yaml:"eventBus" json:"eventBus"`
	Eventstore    eventstoreConfig.Config `yaml:"eventStore" json:"eventStore"`
	OAuthProvider OAuthProviderConfig     `yaml:"oauthProvider" json:"oauthProvider"`
	AuthServer    client.Config           `yaml:"authorizationServer" json:"authorizationServer"`
}

func (c *ClientsConfig) Validate() error {
	err := c.OAuthProvider.Validate()
	if err != nil {
		return fmt.Errorf("oauthProvider.%w", err)
	}
	err = c.AuthServer.Validate()
	if err != nil {
		return fmt.Errorf("authorizationServer.%w", err)
	}
	err = c.Eventbus.Validate()
	if err != nil {
		return fmt.Errorf("eventbus.%w", err)
	}
	err = c.Eventstore.Validate()
	if err != nil {
		return fmt.Errorf("eventstore.%w", err)
	}
	return nil
}

type OAuthProviderConfig struct {
	Jwks       validator.Config `yaml:"jwks" json:"jwks"`
	OAuth      client2.ConfigV2 `yaml:"oauth" json:"oauth"`
	OwnerClaim string           `yaml:"ownerClaim" json:"ownerClaim"`
}

func (c *OAuthProviderConfig) Validate() error {
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	err := c.Jwks.Validate()
	if err != nil {
		return fmt.Errorf("jwks.%w", err)
	}
	err = c.OAuth.Validate()
	if err != nil {
		return fmt.Errorf("oauth.%w", err)
	}
	return nil
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
