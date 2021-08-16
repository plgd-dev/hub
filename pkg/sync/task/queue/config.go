package queue

import (
	"fmt"
	"time"
)

// Config configuration for task queue.
type Config struct {
	// GoroutinePoolSize maximum number of running goroutine instances.
	GoPoolSize int `yaml:"goPoolSize" json:"goPoolSize"`
	// Size size of queue. If it exhausted Submit returns error.
	Size int `yaml:"size" json:"size"`
	// MaxIdleTime sets up the interval time of cleaning up goroutines, 0 means never cleanup.
	MaxIdleTime time.Duration `yaml:"maxIdleTime" json:"maxIdleTime"`
}

// SetDefaults set zero values to defaults.
func (c *Config) Validate() error {
	if c.GoPoolSize <= 0 {
		return fmt.Errorf("goPoolSize('%v')", c.GoPoolSize)
	}
	if c.Size <= 0 {
		return fmt.Errorf("size('%v')", c.Size)
	}
	return nil
}
