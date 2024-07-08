package updater

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/plgd-dev/hub/v2/pkg/log"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
)

type ResourceUpdaterConfig struct {
	Connection                grpcClient.Config `yaml:"grpc" json:"grpc"`
	CleanUpExpiredUpdates     string            `yaml:"cleanUpExpiredUpdates" json:"cleanUpExpiredUpdates"`
	ExtendCronParserBySeconds bool              `yaml:"-" json:"-"`
}

func (c *ResourceUpdaterConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	if c.CleanUpExpiredUpdates == "" {
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
	_, err = s.NewJob(gocron.CronJob(c.CleanUpExpiredUpdates, c.ExtendCronParserBySeconds),
		gocron.NewTask(func() {
			// do nothing
		}))
	if err != nil {
		return fmt.Errorf("cleanUpExpiredUpdates('%v') - %w", c.CleanUpExpiredUpdates, err)
	}
	return nil
}
