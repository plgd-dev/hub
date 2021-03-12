package service

import (
	"encoding/json"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
)

// Config represent application configuration
type Config struct {
	ResourceDirectoryAddr string `envconfig:"RESOURCE_DIRECTORY_ADDRESS"  default:"127.0.0.1:9100"`
	ResourceAggregateAddr string `envconfig:"RESOURCE_AGGREGATE_ADDRESS"  default:"127.0.0.1:9100"`
	GoRoutinePoolSize     int    `envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	Nats                  nats.Config
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}
