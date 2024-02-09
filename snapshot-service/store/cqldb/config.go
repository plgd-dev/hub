package cqldb

import (
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
)

// Config provides Mongo DB configuration options
type Config struct {
	Embedded cqldb.Config `yaml:",inline" json:",inline"`
	Table    string       `yaml:"table" json:"table"`
}

func (c *Config) Validate() error {
	return c.Embedded.Validate()
}
