package validator

import (
	"fmt"

	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
)

type AuthorityConfig struct {
	Address string        `yaml:"address" json:"address"`
	HTTP    client.Config `yaml:"http" json:"http"`
}

type OldAuthorityConfig struct {
	Authority string        `yaml:"authority" json:"authority"`
	Audience  string        `yaml:"audience,omitempty" json:"audience,omitempty"` // deprecated
	HTTP      client.Config `yaml:"http" json:"http"`
}

func (c *OldAuthorityConfig) ToNew() AuthorityConfig {
	return AuthorityConfig{
		Address: c.Authority,
		HTTP:    c.HTTP,
	}
}

func (c *AuthorityConfig) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("address('%v')", c.Address)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type Config struct {
	Audience  string            `yaml:"audience" json:"audience"` // deprecated
	Endpoints []AuthorityConfig `yaml:"endpoints" json:"endpoints"`
}

func (c *Config) Validate() error {
	for i, v := range c.Endpoints {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("servers[%v].%w", i, err)
		}
	}
	return nil
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cfg Config
	err := unmarshal(&cfg)
	if err != nil || len(cfg.Endpoints) == 0 {
		var s OldAuthorityConfig
		err2 := unmarshal(&s)
		if err2 != nil {
			return err
		}
		if s.Audience != "" && cfg.Audience == "" {
			cfg.Audience = s.Audience
		}
		cfg.Endpoints = append(cfg.Endpoints, s.ToNew())
	}
	*c = cfg
	return nil
}

func (c *Config) MarshalYAML() (interface{}, error) {
	if c == nil {
		return nil, nil
	}
	return *c, nil
}

func (c *Config) UnmarshalJSON(data []byte) error {
	var cfg Config
	err := json.Decode(data, &cfg)
	if err != nil || len(cfg.Endpoints) == 0 {
		var s OldAuthorityConfig
		err2 := json.Decode(data, &s)
		if err2 != nil {
			return err
		}
		if s.Audience != "" && cfg.Audience == "" {
			cfg.Audience = s.Audience
		}
		cfg.Endpoints = append(cfg.Endpoints, s.ToNew())
	}
	*c = cfg
	return nil
}

func (c Config) MarshalJSON() ([]byte, error) {
	return json.Encode(c)
}
