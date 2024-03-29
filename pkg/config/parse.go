package config

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

// Parse parse bytes to config
func Parse(data []byte, config interface{}) error {
	err := yaml.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}

// ToString returns string representation of Config
func ToString(config interface{}) string {
	b, _ := yaml.Marshal(config)
	return string(b)
}
