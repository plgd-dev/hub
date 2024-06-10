package service

import (
	"fmt"
	"net"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	grpcServer "github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	httpServer "github.com/plgd-dev/hub/v2/pkg/net/http/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	grpcService "github.com/plgd-dev/hub/v2/snippet-service/service/grpc"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
)

type HTTPConfig struct {
	Addr          string                         `yaml:"address" json:"address"`
	Server        httpServer.Config              `yaml:",inline" json:",inline"`
	Authorization grpcServer.AuthorizationConfig `yaml:"authorization" json:"authorization"`
}

func (c *HTTPConfig) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", c.Addr); err != nil {
		return fmt.Errorf("address('%v') - %w", c.Addr, err)
	}
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC grpcService.Config `yaml:"grpc" json:"grpc"`
	HTTP HTTPConfig         `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

type StorageConfig struct {
	Embedded                  storeConfig.Config `yaml:",inline" json:",inline"`
	ExtendCronParserBySeconds bool               `yaml:"-" json:"-"`
	CleanUpRecords            string             `yaml:"cleanUpRecords" json:"cleanUpRecords"`
}

func (c *StorageConfig) Validate() error {
	if err := c.Embedded.Validate(); err != nil {
		return err
	}
	if c.CleanUpRecords == "" {
		return nil
	}
	s, err := gocron.NewScheduler(gocron.WithLocation(time.Local)) //nolint:gosmopolitan
	if err != nil {
		return fmt.Errorf("cannot create cron job: %w", err)
	}
	defer func() {
		if errS := s.Shutdown(); errS != nil {
			log.Errorf("failed to shutdown cron job: %w", errS)
		}
	}()
	_, err = s.NewJob(gocron.CronJob(c.CleanUpRecords, c.ExtendCronParserBySeconds),
		gocron.NewTask(func() {
			// do nothing
		}))
	if err != nil {
		return fmt.Errorf("cleanUpRecords('%v') - %w", c.CleanUpRecords, err)
	}
	return nil
}

type ClientsConfig struct {
	Storage                StorageConfig     `yaml:"storage" json:"storage"`
	OpenTelemetryCollector otelClient.Config `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
	NATS                   natsClient.Config `yaml:"nats" json:"nats"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	if err := c.NATS.Validate(); err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
}

type Config struct {
	HubID   string        `yaml:"hubID" json:"hubId"`
	Log     log.Config    `yaml:"log" json:"log"`
	APIs    APIsConfig    `yaml:"apis" json:"apis"`
	Clients ClientsConfig `yaml:"clients" json:"clients"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if _, err := uuid.Parse(c.HubID); err != nil {
		return fmt.Errorf("hubID('%v') - %w", c.HubID, err)
	}

	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
