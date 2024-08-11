package mongodb

import (
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
)

type Config struct {
	Mongo pkgMongo.Config `yaml:",inline"`
}

func (c *Config) Validate() error {
	return c.Mongo.Validate()
}
