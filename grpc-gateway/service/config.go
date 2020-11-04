package service

import "github.com/plgd-dev/kit/config"

// Config represent application configuration
type Config struct {
	ResourceDirectoryAddr string `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
