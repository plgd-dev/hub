package service

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	raClient "github.com/plgd-dev/hub/v2/resource-aggregate/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const numberUpdatesInTimeToLive = 3

type serviceStatus struct {
	timeToLive       time.Duration
	raClient         *raClient.Client
	done             chan struct{}
	instanceID       uuid.UUID
	logger           log.Logger
	onlineValidUntil time.Time
}

// NewServiceStatus creates new serviceStatus instance. It will update service metadata in two times in timeToLive.
// If it fails to update service metadata in two times in row, it will kill the service. Because resource aggregate
// can't be sure if the service is still alive. If any other services updates service metadata, it can consider this
// service as dead. And all devices connected to this service will be marked as offline.
func newServiceStatus(instanceID uuid.UUID, timeToLive time.Duration, raClient *raClient.Client, logger log.Logger) (*serviceStatus, error) {
	s := &serviceStatus{
		instanceID: instanceID,
		timeToLive: timeToLive,
		raClient:   raClient,
		done:       make(chan struct{}, 1),
		logger:     logger.With("service-id", instanceID.String()),
	}
	onlineValidUntil, err := s.updateServiceMetadata()
	if err != nil {
		return nil, fmt.Errorf("cannot update service metadata: %w", err)
	}
	s.onlineValidUntil = onlineValidUntil
	return s, nil
}

// updateServiceMetadata updates service metadata in resource aggregate.
func (s *serviceStatus) updateServiceMetadata() (time.Time, error) {
	// set deadline to prevent blocking the service
	deadline := s.onlineValidUntil.Add(s.timeToLive)
	if s.onlineValidUntil.IsZero() {
		deadline = time.Now().Add(s.timeToLive)
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	resp, err := s.raClient.UpdateServiceMetadata(ctx, &commands.UpdateServiceMetadataRequest{
		Update: &commands.UpdateServiceMetadataRequest_Status{
			Status: &commands.ServiceStatus{
				Id:         s.instanceID.String(),
				TimeToLive: s.timeToLive.Nanoseconds(),
				Timestamp:  time.Now().UnixNano(),
			},
		},
	})
	if err == nil {
		s.onlineValidUntil = pkgTime.Unix(0, resp.GetOnlineValidUntil())
		return s.onlineValidUntil, nil
	}
	return time.Time{}, err
}

func needToShutdownService(err error) bool {
	s, ok := status.FromError(err)
	if ok {
		return s.Code() == codes.FailedPrecondition
	}
	return false
}

func (s *serviceStatus) tryUpdateServiceMetadata(now time.Time) error {
	var err error
	var isOffline bool
	switch {
	case now.After(s.onlineValidUntil):
		err = fmt.Errorf("service is offline from time %v", s.onlineValidUntil)
		isOffline = true
	default:
		var onlineValidUntil time.Time
		onlineValidUntil, err = s.updateServiceMetadata()
		if err == nil {
			s.logger.Debugf("service metadata updated, online valid until: %v", onlineValidUntil)
			s.onlineValidUntil = onlineValidUntil
			return nil
		}
	}
	err = fmt.Errorf("cannot update service metadata: %w", err)
	if isOffline || needToShutdownService(err) {
		// Kill the service to prevent inconsistent state propagation of connected devices through this service.
		s.logger.Infof("killing the service to prevent inconsistent state: %v", err)
		errKill := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		if errKill == nil {
			return err
		}
		s.logger.Errorf("cannot kill service: %w", errKill)
		// to prevent too frequent killing the service
		time.Sleep(time.Second)
	} else {
		s.logger.Warnf("%v", err)
	}
	return nil
}

// Serve starts serviceStatus. It will update service metadata in two times in timeToLive.
func (s *serviceStatus) Serve() error {
	now := time.Now()
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	for {
		err := s.tryUpdateServiceMetadata(now)
		if err != nil {
			// the error is set only when sigkill has been sent
			return err
		}
		nextWake := time.Until(s.onlineValidUntil) / numberUpdatesInTimeToLive
		if nextWake < 0 {
			nextWake = 0
		}
		timer.Reset(nextWake)
		select {
		case <-s.done:
			timer.Stop()
			return nil
		case now = <-timer.C:
		}
	}
}

// Close stops serviceStatus.
func (s *serviceStatus) Close() error {
	select {
	case s.done <- struct{}{}:
	default:
	}
	return nil
}
