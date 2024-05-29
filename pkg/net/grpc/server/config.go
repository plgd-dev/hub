package server

import (
	"fmt"
	"net"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"google.golang.org/grpc/keepalive"
)

const defaultMessageSize4MB = 4 * 1024 * 1024

// EnforcementPolicyConfig is used to set keepalive enforcement policy on the
// server-side. Server will close connection with a client that violates this
// policy.
type EnforcementPolicyConfig struct {
	// MinTime is the minimum amount of time a client should wait before sending
	// a keepalive ping.
	MinTime time.Duration `yaml:"minTime" json:"minTime"` // The current default value is 5 minutes.
	// If true, server allows keepalive pings even when there are no active
	// streams(RPCs). If false, and client sends ping when there are no active
	// streams, server will send GOAWAY and close the connection.
	PermitWithoutStream bool `yaml:"permitWithoutStream" json:"permitWithoutStream"` // false by default.
}

func (c EnforcementPolicyConfig) ToGrpc() keepalive.EnforcementPolicy {
	return keepalive.EnforcementPolicy{
		MinTime:             c.MinTime,
		PermitWithoutStream: c.PermitWithoutStream,
	}
}

type KeepAliveConfig struct {
	// MaxConnectionIdle is a duration for the amount of time after which an
	// idle connection would be closed by sending a GoAway. Idleness duration is
	// defined since the most recent time the number of outstanding RPCs became
	// zero or the connection establishment.
	MaxConnectionIdle time.Duration `yaml:"maxConnectionIdle" json:"maxConnectionIdle"` // The current default value is infinity.
	// MaxConnectionAge is a duration for the maximum amount of time a
	// connection may exist before it will be closed by sending a GoAway. A
	// random jitter of +/-10% will be added to MaxConnectionAge to spread out
	// connection storms.
	MaxConnectionAge time.Duration `yaml:"maxConnectionAge" json:"maxConnectionAge"` // The current default value is infinity.
	// MaxConnectionAgeGrace is an additive period after MaxConnectionAge after
	// which the connection will be forcibly closed.
	MaxConnectionAgeGrace time.Duration `yaml:"maxConnectionAgeGrace" json:"maxConnectionAgeGrace"` // The current default value is infinity.
	// After a duration of this time if the server doesn't see any activity it
	// pings the client to see if the transport is still alive.
	// If set below 1s, a minimum value of 1s will be used instead.
	Time time.Duration `yaml:"time" json:"time"` // The current default value is 2 hours.
	// After having pinged for keepalive check, the server waits for a duration
	// of Timeout and if no activity is seen even after that the connection is
	// closed.
	Timeout time.Duration `yaml:"timeout" json:"timeout"` // The current default value is 20 seconds.
}

func (c KeepAliveConfig) ToGrpc() keepalive.ServerParameters {
	return keepalive.ServerParameters{
		MaxConnectionIdle:     c.MaxConnectionIdle,
		MaxConnectionAge:      c.MaxConnectionAge,
		MaxConnectionAgeGrace: c.MaxConnectionAgeGrace,
		Time:                  c.Time,
		Timeout:               c.Timeout,
	}
}

type BaseConfig struct {
	Addr string `yaml:"address" json:"address"`
	// SendMsgSize is the maximum size of a message the server can send. If <=0, a default of 4MB will be used.
	SendMsgSize int `yaml:"sendMsgSize" json:"sendMsgSize"`
	// RecvMsgSize is the maximum size of a message the server can receive. If <=0, a default of 4MB will be used.
	RecvMsgSize       int                     `yaml:"recvMsgSize" json:"recvMsgSize"`
	EnforcementPolicy EnforcementPolicyConfig `yaml:"enforcementPolicy" json:"enforcementPolicy"`
	KeepAlive         KeepAliveConfig         `yaml:"keepAlive" json:"keepAlive"`
	TLS               server.Config           `yaml:"tls" json:"tls"`
}

type Config struct {
	BaseConfig    `yaml:",inline" json:",inline"`
	Authorization AuthorizationConfig `yaml:"authorization" json:"authorization"`
}

type AuthorizationConfig struct {
	OwnerClaim       string `yaml:"ownerClaim" json:"ownerClaim"`
	validator.Config `yaml:",inline" json:",inline"`
}

func (c *AuthorizationConfig) Validate() error {
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	return c.Config.Validate()
}

func (c *BaseConfig) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", c.Addr); err != nil {
		return fmt.Errorf("address('%v') - %w", c.Addr, err)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	if c.SendMsgSize <= 0 {
		c.SendMsgSize = defaultMessageSize4MB
	}
	if c.RecvMsgSize <= 0 {
		c.RecvMsgSize = defaultMessageSize4MB
	}
	return nil
}

func (c *Config) Validate() error {
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization.%w", err)
	}
	return c.BaseConfig.Validate()
}
