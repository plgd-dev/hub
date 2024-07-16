package config

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/store/cqldb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

type Config struct {
	CleanUpDeletedTokens                            string `yaml:"cleanUpDeletedTokens" json:"cleanUpDeletedTokens"`
	ExtendCronParserBySeconds                       bool   `yaml:"-" json:"-"`
	database.Config[*mongodb.Config, *cqldb.Config] `yaml:",inline" json:",inline"`
}

func (c *Config) Validate() error {
	if err := c.Config.Validate(); err != nil {
		return err
	}
	if c.CleanUpDeletedTokens == "" {
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
	_, err = s.NewJob(gocron.CronJob(c.CleanUpDeletedTokens, c.ExtendCronParserBySeconds),
		gocron.NewTask(func() {
			// do nothing
		}))
	if err != nil {
		return fmt.Errorf("cleanUpExpiredUpdates('%v') - %w", c.CleanUpDeletedTokens, err)
	}
	return nil
}
