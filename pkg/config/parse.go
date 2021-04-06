package config

import (
	"fmt"
	"reflect"

	yaml "gopkg.in/yaml.v2"
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

func FieldNameMarshaler(f reflect.StructField) string {
	return f.Name
}
