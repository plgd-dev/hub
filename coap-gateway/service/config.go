package service

import (
	"time"

	"github.com/plgd-dev/kit/security/certManager"

	"github.com/plgd-dev/kit/security/oauth/manager"
)

// Config for application.
type Config struct {
	Addr            string             `envconfig:"ADDRESS" long:"coap-address" default:"0.0.0.0:5684"`
	ExternalPort    uint16             `envconfig:"EXTERNAL_PORT" long:"coap-external-port" default:"5684"`
	FQDN            string             `envconfig:"FQDN" long:"coap-fqdn" default:"coapgw.ocf.cloud"`
	ServerTLSConfig certManager.Config `envconfig:"TLS" long:"coap-tls"`

	RequestTimeout                  time.Duration `envconfig:"REQUEST_TIMEOUT" long:"coap-request-timeout" default:"10s"`
	HeartBeat                       time.Duration `envconfig:"HEARTBEAT" long:"coap-heartbeat" default:"4s"`
	ReconnectInterval               time.Duration `envconfig:"RECONNECT_TIMEOUT" long:"coap-reconnect-interval" default:"10s"`
	MaxMessageSize                  int           `envconfig:"MAX_MESSAGE_SIZE" long:"coap-max-message-size" default:"262144"`
	KeepaliveEnable                 bool          `envconfig:"KEEPALIVE_ENABLE" long:"coap-keepalive-enabled" default:"true"`
	KeepaliveTimeoutConnection      time.Duration `envconfig:"KEEPALIVE_TIMEOUT_CONNECTION" long:"coap-keepalive-timeout" default:"20s"`
	EnableBlockWiseTransfer         bool          `envconfig:"ENABLE_BLOCKWISE_TRANSFER" long:"coap-blockwise-transfer-enabled" default:"true"`
	BlockWiseTransferSZX            string        `envconfig:"BLOCKWISE_TRANSFER_SZX" long:"coap-blockwise-block-size" default:"1024"`
	DisableTCPSignalMessageCSM      bool          `envconfig:"DISABLE_TCP_SIGNAL_MESSAGE_CSM" long:"coap-tcp-signal-message-csm-disabled" default:"false"`
	DisablePeerTCPSignalMessageCSMs bool          `envconfig:"DISABLE_PEER_TCP_SIGNAL_MESSAGE_CSMS" long:"coap-peer-tcp-signal-message-csm-disabled"  default:"false"`
	SendErrorTextInResponse         bool          `envconfig:"ERROR_IN_RESPONSE" long:"coap-error-response-enabled" default:"true"`

	OAuth          manager.Config     `envconfig:"OAUTH" long:"coap-oauth"`
	OAuthTLSConfig certManager.Config `envconfig:"TLS" long:"coap-oauth-tls"`
}

type ClientsConfig struct {
	AuthServerAddr                   string             `envconfig:"AUTH_SERVER_ADDRESS" long:"authorization-address" default:"127.0.0.1:9081"`
	AuthServerClientTLSConfig        certManager.Config `envconfig:"AUTH_SERVER_TLS" long:"authorization-tls"`
	ResourceDirectoryAddr            string             `envconfig:"RESOURCE_DIRECTORY_ADDRESS" long:"resource-directory-address" default:"127.0.0.1:9082"`
	ResourceDirectoryClientTLSConfig certManager.Config `envconfig:"RESOURCE_DIRECTORY_TLS" long:"resource-directory-tls"`
	ResourceAggregateAddr            string             `envconfig:"RESOURCE_AGGREGATE_ADDRESS" long:"resource-aggregate-address" default:"127.0.0.1:9083"`
	ResourceAggregateClientTLSConfig certManager.Config `envconfig:"RESOURCE_AGGREGATE_TLS" long:"resource-aggregate-tls"`
}
