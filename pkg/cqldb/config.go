package cqldb

import (
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
)

type KeyspaceConfig struct {
	Name        string                 `yaml:"name" json:"name"`
	Create      bool                   `yaml:"create" json:"create"`
	Replication map[string]interface{} `yaml:"replication" json:"replication"`
}

func (c *KeyspaceConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name('%v')", c.Name)
	}
	if !c.Create {
		return nil
	}
	if len(c.Replication) == 0 {
		return fmt.Errorf("replication('%v')", c.Replication)
	}
	if _, ok := c.Replication["class"]; !ok {
		return fmt.Errorf("replication.class('%v')", c.Replication["class"])
	}
	if _, ok := c.Replication["class"].(string); !ok {
		return fmt.Errorf("replication.class('%v') - invalid type %T", c.Replication["class"], c.Replication["class"])
	}
	if _, ok := c.Replication["replication_factor"]; !ok {
		return fmt.Errorf("replication.replication_factor('%v')", c.Replication["replication_factor"])
	}
	if _, ok := c.Replication["replication_factor"].(int); !ok {
		return fmt.Errorf("replication.replication_factor('%v') - invalid type %T", c.Replication["replication_factor"], c.Replication["replication_factor"])
	}
	return nil
}

type ConstantReconnectionPolicyConfig struct {
	Interval   time.Duration `yaml:"interval" json:"interval"`
	MaxRetries int           `yaml:"maxRetries" json:"maxRetries"`
}

func (c *ConstantReconnectionPolicyConfig) Validate() error {
	if c.Interval < 0 {
		return fmt.Errorf("interval('%v')", c.Interval)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("maxRetries('%v')", c.MaxRetries)
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = int(^uint(0) >> 1)
	}
	if c.Interval == 0 {
		c.Interval = 3 * time.Second
	}
	return nil
}

type ReconnectionPolicyConfig struct {
	Constant ConstantReconnectionPolicyConfig `yaml:"constant" json:"constant"`
}

func (c *ReconnectionPolicyConfig) Validate() error {
	if err := c.Constant.Validate(); err != nil {
		return fmt.Errorf("constant.%w", err)
	}
	return nil
}

type Config struct {
	Hosts                 []string                 `yaml:"hosts" json:"hosts"`
	Port                  int                      `yaml:"port" json:"port"`
	NumConns              int                      `yaml:"numConnections" json:"numConnections"`
	ConnectTimeout        time.Duration            `yaml:"connectTimeout" json:"connectTimeout"`
	ReconnectionPolicy    ReconnectionPolicyConfig `yaml:"reconnectionPolicy" json:"reconnectionPolicy"`
	UseHostnameResolution bool                     `yaml:"useHostnameResolution" json:"useHostnameResolution"`
	Keyspace              KeyspaceConfig           `yaml:"keyspace" json:"keyspace"`
	TLS                   client.Config            `yaml:"tls" json:"tls"`
}

func (c *Config) Validate() error {
	if len(c.Hosts) == 0 {
		return fmt.Errorf("hosts('%v')", c.Hosts)
	}
	if c.NumConns <= 0 {
		return fmt.Errorf("numConnections('%v')", c.NumConns)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	if err := c.Keyspace.Validate(); err != nil {
		return fmt.Errorf("keyspace.%w", err)
	}
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("port('%v') - invalid port", c.Port)
	}
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if err := c.ReconnectionPolicy.Validate(); err != nil {
		return fmt.Errorf("reconnectionPolicy.%w", err)
	}
	return nil
}
