package service

import (
	"github.com/plgd-dev/kit/sync/task/queue"
	"time"

	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
)

type DeviceStatusExpirationConfig struct {
	Enabled   bool          `yaml:"enabled" json:"enabled" envconfig:"ENABLED"`
	ExpiresIn time.Duration `yaml:"expiresIn" json:"expiresIn" envconfig:"EXPIRES_IN"`
}

func (c *DeviceStatusExpirationConfig) SetDefaults() {
	if c.ExpiresIn == 0 {
		c.ExpiresIn = 24 * time.Hour
	}
}

func (c Config) SetDefaults() Config {
	c.Service.Coap.TaskQueue.SetDefaults()
	c.Service.Coap.DeviceStatusExpiration.SetDefaults()
	return c
}

// Config for application.
type Config struct {
	Log        log.Config       `yaml:"log" json:"log" envconfig:"LOG"`
	Service    APIsConfig	    `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients	   ClientsConfig    `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
}

type APIsConfig struct {
	Coap CoapConfig `yaml:"coap" json:"coap" envconfig:"COAP"`
}

type CoapConfig struct {
	Addr            string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"0.0.0.0:5684"`
	ExternalAddress string        `yaml:"externalAddress" json:"externalAddress" envconfig:"EXTERNAL_ADDRESS" default:"5684"`
	TLSConfig       server.Config `yaml:"tls" json:"tls" envconfig:"TLS"`

	RequestTimeout       time.Duration      `yaml:"timeout" json:"timeout" envconfig:"TIMEOUT" default:"10s"`
	HeartBeat            time.Duration      `yaml:"heartbeat" json:"heartbeat" envconfig:"HEARTBEAT" default:"4s"`
	ReconnectInterval    time.Duration      `yaml:"reconnectInterval" json:"reconnectInterval" envconfig:"RECONNECT_INTERVAL" default:"10s"`
	Capabilities         CapabilitiesConfig `yaml:"capabilities" json:"capabilities" envconfig:"CAPABILITIES"`
	LogMessages          bool               `yaml:"logMessageEnabled" json:"logMessageEnabled" envconfig:"LOG_MESSAGE_ENABLED" default:"false"`
	NumTaskWorkers       int                `yaml:"numTaskWorkers,omitempty" json:"numTaskWorkers" envconfig:"NUM_TASK_WORKERS"`
	LimitTasks           int                `yaml:"limitTasks,omitempty" json:"limitTasks" envconfig:"LIMIT_TASKS"`
	TaskQueue            queue.Config       `yaml:"taskQueue" json:"taskQueue" envconfig:"TASK_QUEUE"`
	DeviceStatusExpiration DeviceStatusExpirationConfig `yaml:"deviceStatusExpiration" envconfig:"DEVICE_STATUS_EXPIRATION"`
}

type CapabilitiesConfig struct {
	MaxMessageSize                  int           `yaml:"maxMessageSize" json:"maxMessageSize" envconfig:"MAX_MESSAGE_SIZE" default:"262144"`
	KeepaliveEnable                 bool          `yaml:"keepaliveEnabled" json:"keepaliveEnabled" envconfig:"KEEPALIVE_ENABLED" default:"true"`
	KeepaliveTimeoutConnection      time.Duration `yaml:"keepaliveTimeout" json:"keepaliveTimeout" envconfig:"KEEPALIVE_TIMEOUT" default:"20s"`
	BlockWiseTransferEnable         bool          `yaml:"blockwiseTransferEnabled" json:"blockwiseTransferEnabled" envconfig:"BLOCKWISE_TRANSFER_ENABLED" default:"true"`
	BlockWiseTransferSZX            string        `yaml:"blockwiseTransferBlockSize" json:"blockwiseTransferBlockSize" envconfig:"BLOCKWISE_TRANSFER_BLOCK_SIZE" default:"1024"`
	DisableTCPSignalMessageCSM      bool          `yaml:"tcpSignalMessageCSMDisabled" json:"tcpSignalMessageCSMDisabled" envconfig:"TCP_SIGNAL_MESSAGE_CSM_DISABLED" default:"false"`
	DisablePeerTCPSignalMessageCSMs bool          `yaml:"peerTcpSignalMessageCSMDisabled" json:"peerTcpSignalMessageCSMDisabled" envconfig:"PEER_TCP_SIGNAL_MESSAGE_CSM_DISABLED" default:"false"`
	SendErrorTextInResponse         bool          `yaml:"errorResponseEnabled" json:"errorResponseEnabled" envconfig:"ERROR_RESPONSE_ENABLED" default:"true"`
}

type ClientsConfig struct {
	Nats             	 nats.Config             `yaml:"nats" json:"nats" envconfig:"NATS"`
	OAuthProvider        OAuthProvider           `yaml:"oauthProvider" json:"oauthProvider" envconfig:"AUTH_PROVIDER"`
	Authorization        AuthorizationConfig     `yaml:"authorizationServer" json:"authorizationServer" envconfig:"AUTHORIZATION"`
	ResourceDirectory    ResourceDirectoryConfig `yaml:"resourceDirectory" json:"resourceDirectory" envconfig:"RESOURCE_DIRECTORY"`
	ResourceAggregate    ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate" envconfig:"RESOURCE_AGGREGATE"`
}

type OAuthProvider struct {
	OAuth        manager.Config `yaml:"oauth" json:"oauth" envconfig:"OAUTH"`
	TLSConfig     client.Config      `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type AuthorizationConfig struct {
	Addr         string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9081"`
	TLSConfig    client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceDirectoryConfig struct {
	Addr         string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9082"`
	TLSConfig    client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

type ResourceAggregateConfig struct {
	Addr         string        `yaml:"address" json:"address" envconfig:"ADDRESS" default:"127.0.0.1:9083"`
	TLSConfig    client.Config `yaml:"tls" json:"tls" envconfig:"TLS"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}