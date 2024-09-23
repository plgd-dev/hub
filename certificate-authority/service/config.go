package service

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	storeConfig "github.com/plgd-dev/hub/v2/certificate-authority/store/config"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	httpServer "github.com/plgd-dev/hub/v2/pkg/net/http/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
)

type Config struct {
	HubID   string                   `yaml:"hubID" json:"hubId"`
	Log     log.Config               `yaml:"log" json:"log"`
	APIs    APIsConfig               `yaml:"apis" json:"apis"`
	Signer  grpcService.SignerConfig `yaml:"signer" json:"signer"`
	Clients ClientsConfig            `yaml:"clients" json:"clients"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Signer.Validate(); err != nil {
		return fmt.Errorf("signer.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if _, err := uuid.Parse(c.HubID); err != nil {
		return fmt.Errorf("hubID('%v') - %w", c.HubID, err)
	}

	_, err := grpcService.NewSigner(c.APIs.GRPC.Authorization.OwnerClaim, c.HubID, c.APIs.HTTP.ExternalAddress, c.Signer)
	if err != nil {
		return fmt.Errorf("signer('%v') - %w", c.Signer, err)
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

type HTTPConfig struct {
	ExternalAddress string            `yaml:"externalAddress" json:"externalAddress"`
	Addr            string            `yaml:"address" json:"address"`
	Server          httpServer.Config `yaml:",inline" json:",inline"`
}

func (c *HTTPConfig) Validate() error {
	if _, err := url.ParseRequestURI(c.ExternalAddress); err != nil {
		return fmt.Errorf("externalAddress('%v') invalid", c.ExternalAddress)
	}
	if _, err := net.ResolveTCPAddr("tcp", c.Addr); err != nil {
		return fmt.Errorf("address('%v') - %w", c.Addr, err)
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
}

func (c *ClientsConfig) Validate() error {
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
