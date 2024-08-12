package service

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
)

func NewExpiredUpdatesChecker(cleanUpExpiredUpdates string, withSeconds bool, onCheckTimeout func()) (gocron.Scheduler, error) {
	s, err := gocron.NewScheduler(gocron.WithLocation(time.Local)) //nolint:gosmopolitan
	if err != nil {
		return nil, fmt.Errorf("cannot create cron job: %w", err)
	}
	_, err = s.NewJob(gocron.CronJob(cleanUpExpiredUpdates, withSeconds), gocron.NewTask(onCheckTimeout))
	if err != nil {
		return nil, fmt.Errorf("cannot create cron job: %w", err)
	}
	s.Start()
	return s, nil
}
