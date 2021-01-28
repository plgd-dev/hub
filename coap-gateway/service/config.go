package service

import (
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	oauthClient "github.com/plgd-dev/kit/security/oauth/service/client"
	"runtime"
	"time"

	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
)

// Config for application.
type Config struct {
	Log              log.Config     `yaml:"log" json:"log"`
	Service          APIsConfig	    `yaml:"apis" json:"apis"`
	Clients			 ClientsConfig  `yaml:"clients" json:"clients"`
}

func (c Config) CheckForDefaults() Config {
	if c.Service.CoapGW.NumTaskWorkers == 0 {
		c.Service.CoapGW.NumTaskWorkers = runtime.NumCPU() * 2
	}
	if c.Service.CoapGW.LimitTasks == 0 {
		c.Service.CoapGW.LimitTasks = 1024 * 1024 * 2
	}
	return c
}

type APIsConfig struct {
	CoapGW           CoapConfig     `yaml:"coap" json:"coap"`
}

type CoapConfig struct {
	Addr                            string             `yaml:"address" json:"address" default:"0.0.0.0:5684"`
	ExternalAddress                 string             `yaml:"externalAddress" json:"externalAddress" default:"5684"`
	ServerTLSConfig                 server.Config      `yaml:"tls" json:"tls"`
	RequestTimeout                  time.Duration      `yaml:"timeout" json:"timeout" default:"10s"`
	HeartBeat                       time.Duration      `yaml:"heartbeat" json:"heartbeat" default:"4s"`
	ReconnectInterval               time.Duration      `yaml:"reconnectInterval" json:"reconnectInterval" default:"10s"`
	Capabilities                    CapabilitiesConfig `yaml:"capabilities" json:"capabilities"`
	LogMessages                     bool           `yaml:"logMessageEnabled" json:"logMessageEnabled" default:"false"`
	NumTaskWorkers                  int            `yaml:"numTaskWorkers,omitempty" json:"numTaskWorkers"`
	LimitTasks                      int            `yaml:"limitTasks,omitempty" json:"limitTasks"`
}

type CapabilitiesConfig struct {
	MaxMessageSize                  int                `yaml:"maxMessageSize" json:"maxMessageSize" default:"262144"`
	KeepaliveEnable                 bool               `yaml:"keepaliveEnabled" json:"keepaliveEnabled" default:"true"`
	KeepaliveTimeoutConnection      time.Duration      `yaml:"keepaliveTimeout" json:"keepaliveTimeout" default:"20s"`
	BlockWiseTransferEnable         bool               `yaml:"blockwiseTransferEnabled" json:"blockwiseTransferEnabled" default:"true"`
	BlockWiseTransferSZX            string             `yaml:"blockwiseTransferBlockSize" json:"blockwiseTransferBlockSize" default:"1024"`
	DisableTCPSignalMessageCSM      bool               `yaml:"tcpSignalMessageCSMDisabled" json:"tcpSignalMessageCSMDisabled" default:"false"`
	DisablePeerTCPSignalMessageCSMs bool               `yaml:"peerTcpSignalMessageCSMDisabled" json:"peerTcpSignalMessageCSMDisabled" default:"false"`
	SendErrorTextInResponse         bool               `yaml:"errorResponseEnabled" json:"errorResponseEnabled" default:"true"`
}

type ClientsConfig struct {
	OAuthProvider         OAuthProvider           `yaml:"oauthProvider" json:"oauthProvider"`
	Authorization         AuthorizationConfig     `yaml:"authorizationServer" json:"authorizationServer"`
	ResourceDirectory     ResourceDirectoryConfig `yaml:"resourceDirectory" json:"resourceDirectory"`
	ResourceAggregate     ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate"`
}

type OAuthProvider struct {
	OAuthConfig        oauthClient.Config     `yaml:"oauth" json:"oauth"`
	OAuthTLSConfig     client.Config          `yaml:"tls" json:"tls"`
}

type AuthorizationConfig struct {
	AuthServerAddr                  string              `yaml:"address" json:"address" default:"127.0.0.1:9081"`
	AuthServerTLSConfig             client.Config       `yaml:"tls" json:"tls"`
}

type ResourceDirectoryConfig struct {
	ResourceDirectoryAddr           string              `yaml:"address" json:"address" default:"127.0.0.1:9082"`
	ResourceDirectoryTLSConfig      client.Config       `yaml:"tls" json:"tls"`
}

type ResourceAggregateConfig struct {
	ResourceAggregateAddr            string             `yaml:"address" json:"address" default:"127.0.0.1:9083"`
	ResourceAggregateTLSConfig       client.Config      `yaml:"tls" json:"tls"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}