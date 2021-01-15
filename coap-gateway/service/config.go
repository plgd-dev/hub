package service

import (
	"time"

	"github.com/plgd-dev/kit/security/certManager"
	"github.com/plgd-dev/kit/security/oauth/manager"
)

// Config for application.
type ServiceConfig struct {

	CoapGW CoapConfig `yaml:"coap" json:"coap"`
}

type CoapConfig struct {
	Addr            string             `yaml:"address" json:"address" default:"0.0.0.0:5684"`
	ExternalPort    uint16             `yaml:"external-port" json:"external-port" default:"5684"`
	FQDN            string             `yaml:"fqdn" json:"fqdn" default:"coapgw.ocf.cloud"`
	ServerTLSConfig certManager.Config `yaml:"tls" json:"tls"`

	RequestTimeout                  time.Duration `yaml:"request-timeout" json:"request-timeout" default:"10s"`
	HeartBeat                       time.Duration `yaml:"heartbeat" json:"heartbeat" default:"4s"`
	ReconnectInterval               time.Duration `yaml:"reconnect-interval" json:"reconnect-interval" default:"10s"`
	MaxMessageSize                  int           `yaml:"max-message-size" json:"max-message-size" default:"262144"`
	KeepaliveEnable                 bool          `yaml:"keepalive-enabled" json:"keepalive-enabled" default:"true"`
	KeepaliveTimeoutConnection      time.Duration `yaml:"keepalive-timeout" json:"keepalive-timeout" default:"20s"`
	BlockWiseTransferEnable         bool          `yaml:"blockwise-transfer-enabled" json:"blockwise-transfer-enabled" default:"true"`
	BlockWiseTransferSZX            string        `yaml:"blockwise-block-size" json:"blockwise-block-size" default:"1024"`
	DisableTCPSignalMessageCSM      bool          `yaml:"tcp-signal-message-csm-disabled" json:"tcp-signal-message-csm-disabled" default:"false"`
	DisablePeerTCPSignalMessageCSMs bool          `yaml:"peer-tcp-signal-message-csm-disabled" json:"peer-tcp-signal-message-csm-disabled" default:"false"`
	SendErrorTextInResponse         bool          `yaml:"error-response-enabled" json:"error-response-enabled" default:"true"`
}

type ClientsConfig struct {
	OAuthProvider     OAuthConfig             `yaml:"oauth-provider" json:"oauth-provider"`
	Authorization     AuthorizationConfig     `yaml:"authorization" json:"authorization"`
	ResourceDirectory ResourceDirectoryConfig `yaml:"resource-directory" json:"resource-directory"`
	ResourceAggregate ResourceAggregateConfig `yaml:"resource-aggregate" json:"resource-aggregate"`
}

type OAuthConfig struct {
	OAuthConfig    manager.Config     `yaml:"oauth" json:"oauth"`
	OAuthTLSConfig certManager.Config `yaml:"tls" json:"tls"`
}

type AuthorizationConfig struct {
	AuthServerAddr            string             `yaml:"address" json:"address" default:"127.0.0.1:9081"`
	AuthServerClientTLSConfig certManager.Config `yaml:"tls" json:"tls"`
}

type ResourceDirectoryConfig struct {
	ResourceDirectoryAddr            string             `yaml:"address" json:"address" default:"127.0.0.1:9082"`
	ResourceDirectoryClientTLSConfig certManager.Config `yaml:"tls" json:"tls"`
}

type ResourceAggregateConfig struct {
	ResourceAggregateAddr            string             `yaml:"address" json:"address" default:"127.0.0.1:9083"`
	ResourceAggregateClientTLSConfig certManager.Config `yaml:"tls" json:"tls"`
}
