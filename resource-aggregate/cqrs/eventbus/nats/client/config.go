package client

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
)

type PendingLimitsConfig struct {
	MsgLimit   int `yaml:"msgLimit" json:"msgLimit"`
	BytesLimit int `yaml:"bytesLimit" json:"bytesLimit"`
}

func (c *PendingLimitsConfig) Validate() error {
	if c.MsgLimit == 0 {
		return fmt.Errorf("msgLimit('%v')", c.MsgLimit)
	}
	if c.BytesLimit == 0 {
		return fmt.Errorf("bytesLimit('%v')", c.BytesLimit)
	}
	return nil
}

type Config struct {
	URL            string              `yaml:"url" json:"url"`
	FlusherTimeout time.Duration       `yaml:"flusherTimeout" json:"flusherTimeout"`
	PendingLimits  PendingLimitsConfig `yaml:"pendingLimits" json:"pendingLimits"`
	TLS            client.Config       `yaml:"tls" json:"tls"`
	Options        []nats.Option       `yaml:"-" json:"-"`
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url('%v')", c.URL)
	}
	if err := c.PendingLimits.Validate(); err != nil {
		return fmt.Errorf("pendingLimits.%w", err)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}

type LeadResourceTypeFilter string

const (
	LeadResourceTypeFilter_None  LeadResourceTypeFilter = ""
	LeadResourceTypeFilter_First LeadResourceTypeFilter = "first"
	LeadResourceTypeFilter_Last  LeadResourceTypeFilter = "last"
)

func CheckResourceTypeFilterString(s string) error {
	switch s {
	case string(LeadResourceTypeFilter_First), string(LeadResourceTypeFilter_Last), string(LeadResourceTypeFilter_None):
		return nil
	}
	return errors.New("unknown LeadResourceTypeFilter")
}

type LeadResourceTypePublisherConfig struct {
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	RegexFilter []string               `yaml:"regexFilter" json:"regexFilter"`
	Filter      LeadResourceTypeFilter `yaml:"filter" json:"filter"`
	UseUUID     bool                   `yaml:"useUUID" json:"useUUID"`

	compiledRegexFilter []*regexp.Regexp `yaml:"-" json:"-"`
}

func (c *LeadResourceTypePublisherConfig) IsEnabled() bool {
	return c != nil && c.Enabled
}

func (c *LeadResourceTypePublisherConfig) Validate() error {
	if err := CheckResourceTypeFilterString(string(c.Filter)); err != nil {
		return fmt.Errorf("filter(%v): %w", c.Filter, err)
	}
	if len(c.RegexFilter) > 0 {
		compiledRegexFilter := make([]*regexp.Regexp, 0, len(c.RegexFilter))
		for _, rf := range c.RegexFilter {
			compiledRf, err := regexp.Compile(rf)
			if err != nil {
				return fmt.Errorf("regexFilter('%v'): %w", rf, err)
			}
			compiledRegexFilter = append(compiledRegexFilter, compiledRf)
		}
		c.compiledRegexFilter = compiledRegexFilter
	}
	return nil
}

func (c *LeadResourceTypePublisherConfig) GetCompiledRegexFilter() []*regexp.Regexp {
	return c.compiledRegexFilter
}

type ConfigPublisher struct {
	Config           `yaml:",inline" json:",inline"`
	JetStream        bool                             `yaml:"jetstream" json:"jetstream"`
	LeadResourceType *LeadResourceTypePublisherConfig `yaml:"leadResourceType,omitempty" json:"leadResourceType,omitempty"`
}

func (c *ConfigPublisher) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url('%v')", c.URL)
	}
	if c.FlusherTimeout <= 0 {
		return fmt.Errorf("flusherTimeout('%v')", c.FlusherTimeout)
	}
	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	if c.LeadResourceType != nil {
		if err := c.LeadResourceType.Validate(); err != nil {
			return fmt.Errorf("leadResourceType.%w", err)
		}
	}
	return nil
}

type LeadResourceTypeSubscriberConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

func (c *LeadResourceTypeSubscriberConfig) IsEnabled() bool {
	return c != nil && c.Enabled
}

type ConfigSubscriber struct {
	Config           `yaml:",inline" json:",inline"`
	LeadResourceType *LeadResourceTypeSubscriberConfig `yaml:"leadResourceType,omitempty" json:"leadResourceType,omitempty"`
}
