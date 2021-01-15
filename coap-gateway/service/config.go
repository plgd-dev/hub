package service

import (
	"time"

	"github.com/plgd-dev/kit/security/certManager/server"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/oauth/manager"
)

// Config for application.
type ServiceConfig struct {

	CoapGW                CoapConfig       `yaml:"coap" json:"coap"`
}

type CoapConfig struct {
	Addr                            string             `yaml:"address" json:"address" default:"0.0.0.0:5684"`
	ExternalAddress                 string             `yaml:"externalAddress" json:"externalAddress" default:"5684"`
	ServerTLSConfig                 server.ServerConfig `yaml:"tls" json:"tls"`
	RequestTimeout                  time.Duration      `yaml:"timeout" json:"timeout" default:"10s"`
	HeartBeat                       time.Duration      `yaml:"heartbeat" json:"heartbeat" default:"4s"`
	ReconnectInterval               time.Duration      `yaml:"reconnectInterval" json:"reconnectInterval" default:"10s"`
	Capabilities                    CapabilitiesConfig `yaml:"capabilities" json:"capabilities"`
}

type CapabilitiesConfig struct {
	MaxMessageSize                  int                `yaml:"maxMessageSize" json:"maxMessageSize" default:"262144"`
	KeepaliveEnable                 bool               `yaml:"keepaliveEnabled" json:"keepaliveEnabled" default:"true"`
	KeepaliveTimeoutConnection      time.Duration      `yaml:"keepaliveTimeout" json:"keepaliveTimeout" default:"20s"`
	BlockWiseTransferEnable         bool               `yaml:"blockwiseTransferEenabled" json:"blockwiseTransferEenabled" default:"true"`
	BlockWiseTransferSZX            string             `yaml:"blockwiseBlockSize" json:"blockwiseBlockSize" default:"1024"`
	DisableTCPSignalMessageCSM      bool               `yaml:"tcpSignalMessageCSMDisabled" json:"tcpSignalMessageCSMDisabled" default:"false"`
	DisablePeerTCPSignalMessageCSMs bool               `yaml:"peerTcpSignalMessageCSMDisabled" json:"peerTcpSignalMessageCSMDisabled" default:"false"`
	SendErrorTextInResponse         bool               `yaml:"errorResponseEnabled" json:"errorResponseEnabled" default:"true"`
}

type ClientsConfig struct {
	OAuthProvider                   OAuthConfig             `yaml:"oAuthProvider" json:"oAuthProvider"`
	Authorization                   AuthorizationConfig     `yaml:"authorizationServer" json:"authorizationServer"`
	ResourceDirectory               ResourceDirectoryConfig `yaml:"resourceDirectory" json:"resourceDirectory"`
	ResourceAggregate               ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate"`
}

type OAuthConfig struct {
	OAuthConfig                     manager.Config           `yaml:"oauth" json:"oauth"`
	OAuthTLSConfig                  client.ClientConfig `yaml:"tls" json:"tls"`
}

type AuthorizationConfig struct {
	AuthServerAddr                  string                   `yaml:"address" json:"address" default:"127.0.0.1:9081"`
	AuthServerTLSConfig             client.ClientConfig `yaml:"tls" json:"tls"`
}

type ResourceDirectoryConfig struct {
	ResourceDirectoryAddr           string                   `yaml:"address" json:"address" default:"127.0.0.1:9082"`
	ResourceDirectoryTLSConfig      client.ClientConfig `yaml:"tls" json:"tls"`
}

type ResourceAggregateConfig struct {
	ResourceAggregateAddr            string                   `yaml:"address" json:"address" default:"127.0.0.1:9083"`
	ResourceAggregateTLSConfig       client.ClientConfig `yaml:"tls" json:"tls"`
}
