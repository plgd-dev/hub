package config

import (
	"io/ioutil"

	"github.com/jessevdk/go-flags"
)

type ConfigPath struct {
	ConfigPath string `long:"config" description:"yaml config file path"`
}

type Validator interface {
	Validate() error
}

func LoadAndValidateConfig(v Validator) error {
	err := Load(v)
	if err != nil {
		return err
	}
	return v.Validate()
}

// Load loads config from ENV config or arguments config.
func Load(config interface{}) error {
	var c ConfigPath
	_, err := flags.NewParser(&c, flags.Default|flags.IgnoreUnknown).Parse()
	if err != nil {
		return err
	}

	return Read(c.ConfigPath, config)
}

// Read reads config from file.
func Read(filename string, config interface{}) error {
	cfg, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return Parse(cfg, config)
}
