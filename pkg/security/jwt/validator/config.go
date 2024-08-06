package validator

import (
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
)

type AuthorityConfig struct {
	Authority string        `yaml:"authority" json:"authority"`
	HTTP      client.Config `yaml:"http" json:"http"`
}

func (c *AuthorityConfig) Validate() error {
	if c.Authority == "" {
		return fmt.Errorf("authority('%v')", c.Authority)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type TokenTrustVerificationConfig struct {
	CacheExpiration time.Duration `yaml:"cacheExpiration,omitempty" json:"cacheExpiration,omitempty"`
}

func (c *TokenTrustVerificationConfig) Validate() error {
	if c.CacheExpiration == 0 {
		c.CacheExpiration = time.Second * 30
	}
	return nil
}

type Config struct {
	Audience          string                       `yaml:"audience" json:"audience"`
	Endpoints         []AuthorityConfig            `yaml:"endpoints" json:"endpoints"`
	TokenVerification TokenTrustVerificationConfig `yaml:"tokenTrustVerification,omitempty" json:"tokenTrustVerification,omitempty"`
	Authority         *string                      `yaml:"authority,omitempty" json:"authority,omitempty"` // deprecated
	HTTP              *client.Config               `yaml:"http,omitempty" json:"http,omitempty"`           // deprecated
}

func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		if c.Authority == nil || c.HTTP == nil {
			return fmt.Errorf("endpoints('%v') - are empty", c.Endpoints)
		}
		c.Endpoints = []AuthorityConfig{{
			Authority: *c.Authority,
			HTTP:      *c.HTTP,
		}}
		c.Authority = nil
		c.HTTP = nil
	}
	for i, v := range c.Endpoints {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("endpoints[%v].%w", i, err)
		}
	}
	return c.TokenVerification.Validate()
}
