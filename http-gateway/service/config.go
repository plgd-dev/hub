package service

import (
	"fmt"
	"reflect"

	"github.com/go-ocf/kit/security/certManager"

	"gopkg.in/yaml.v2"
)

// Config represent application configuration
type Config struct {
	Address               string             `envconfig:"ADDRESS" default:"0.0.0.0:7000"`
	Listen                certManager.Config `envconfig:"LISTEN"`
	Dial                  certManager.Config `envconfig:"DIAL"`
	JwksURL               string             `envconfig:"JWKS_URL"`
	ResourceDirectoryAddr string             `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
}

func ParseConfig(s string) (Config, error) {
	var cfg Config
	err := yaml.Unmarshal([]byte(s), &cfg, yaml.DecoderWithFieldNameMarshaler(FieldNameMarshaler))
	if err != nil {
		return cfg, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func (c Config) String() string {
	b, _ := yaml.Marshal(c, yaml.EncoderWithFieldNameMarshaler(FieldNameMarshaler))
	return string(b)
}

func FieldNameMarshaler(f reflect.StructField) string {
	return f.Name
}
