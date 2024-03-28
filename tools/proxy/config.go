package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

type LivenessConfig struct {
	InitialDelay     time.Duration `json:"initialDelay" yaml:"initialDelay"`
	Period           time.Duration `json:"period" yaml:"period"`
	FailureThreshold int           `json:"failureThreshold" yaml:"failureThreshold"`
}

func (c *LivenessConfig) Validate() error {
	if c.Period <= 0 {
		return fmt.Errorf("period(%v) - invalid value", c.Period)
	}
	if c.FailureThreshold < 0 {
		return fmt.Errorf("failureThreshold(%v) - invalid value", c.FailureThreshold)
	}
	return nil
}

type TimeoutsConfig struct {
	Dial  time.Duration `json:"dial" yaml:"connect"`
	Read  time.Duration `json:"read" yaml:"read"`
	Write time.Duration `json:"write" yaml:"write"`
}

func (c *TimeoutsConfig) Validate() error {
	if c.Read <= 0 {
		return fmt.Errorf("read(%v) - invalid value", c.Read)
	}
	if c.Read <= 0 {
		return fmt.Errorf("read(%v) - invalid value", c.Read)
	}
	if c.Write <= 0 {
		return fmt.Errorf("write(%v) - invalid value", c.Write)
	}
	return nil
}

type TargetConfig struct {
	Addr       string         `json:"address" yaml:"address"`
	Liveness   LivenessConfig `json:"livenessProbe" yaml:"livenessProbe"`
	Timeouts   TimeoutsConfig `json:"timeouts" yaml:"timeouts"`
	BufferSize int            `json:"bufferSize" yaml:"bufferSize"`
}

func (c *TargetConfig) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address - cannot be empty")
	}
	if err := c.Timeouts.Validate(); err != nil {
		return fmt.Errorf("timeouts.%w", err)
	}
	if err := c.Liveness.Validate(); err != nil {
		return fmt.Errorf("livenessProbe.%w", err)
	}
	if c.BufferSize <= 0 {
		c.BufferSize = 1024
	}
	return nil
}

type TunnelConfig struct {
	Addr    string         `json:"address" yaml:"address"`
	Targets []TargetConfig `json:"targets" yaml:"targets"`
}

func (c *TunnelConfig) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address - cannot be empty")
	}
	l, err := net.Listen("tcp", c.Addr)
	if err != nil {
		return fmt.Errorf("address(%v) - %w", c.Addr, err)
	}
	_ = l.Close()

	if len(c.Targets) == 0 {
		return fmt.Errorf("targets - cannot be empty")
	}
	for i, target := range c.Targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf("targets[%v].%w", i, err)
		}
	}
	return nil
}

type TCPConfig struct {
	Tunnels []TunnelConfig `json:"tunnels" yaml:"tunnels"`
}

func (c *TCPConfig) Validate() error {
	if len(c.Tunnels) == 0 {
		return fmt.Errorf("tunnels - cannot be empty")
	}
	for i, tunnel := range c.Tunnels {
		if err := tunnel.Validate(); err != nil {
			return fmt.Errorf("tunnels[%v].%w", i, err)
		}
	}
	return nil
}

type ApisConfig struct {
	TCP TCPConfig `json:"tcp" yaml:"tcp"`
}

func (c *ApisConfig) Validate() error {
	if err := c.TCP.Validate(); err != nil {
		return fmt.Errorf("tcp.%w", err)
	}
	return nil
}

type DirectoryConfig struct {
	Path string `json:"path" yaml:"path"`
}

func (c *DirectoryConfig) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("path - cannot be empty")
	}
	p, err := os.Stat(c.Path)
	if err != nil {
		return fmt.Errorf("path(%v) - %w", c.Path, err)
	}
	if !p.IsDir() {
		return fmt.Errorf("path(%v) - not a directory", c.Path)
	}
	testFile := filepath.Clean(c.Path + string(filepath.Separator) + "test.plgd.file")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		return fmt.Errorf("path(%v) - %w", c.Path, err)
	}
	if err := os.Remove(testFile); err != nil {
		return fmt.Errorf("path(%v) - %w", c.Path, err)
	}
	return nil
}

type StorageConfig struct {
	Directory DirectoryConfig `json:"fileDirectory" yaml:"fileDirectory"`
}

func (c *StorageConfig) Validate() error {
	if err := c.Directory.Validate(); err != nil {
		return fmt.Errorf("fileDirectory.%w", err)
	}
	return nil
}

type ClientsConfig struct {
	Storage StorageConfig `json:"storage" yaml:"storage"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	return nil
}

type Config struct {
	Log     log.Config    `json:"log" yaml:"log"`
	Apis    ApisConfig    `json:"apis" yaml:"apis"`
	Clients ClientsConfig `json:"clients" yaml:"clients"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.Apis.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
