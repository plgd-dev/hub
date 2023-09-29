package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/service"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	raClient "github.com/plgd-dev/hub/v2/resource-aggregate/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const numberUpdatesInTimeToLive = 3

type serviceHeartbeat struct {
	timeToLive          time.Duration
	raClient            *raClient.Client
	done                chan struct{}
	instanceID          uuid.UUID
	logger              log.Logger
	heartbeatValidUntil time.Time
	service             *service.Service
}

// NewServiceHeartbeat creates new serviceHeartbeat instance. It will update service metadata in two times in timeToLive.
// If it fails to update service metadata in two times in row, it will kill the service. Because resource aggregate
// can't be sure if the service is still alive. If any other services updates service metadata, it can consider this
// service as dead. And all devices connected to this service will be marked as offline.
func newServiceHeartbeat(instanceID uuid.UUID, timeToLive time.Duration, raClient *raClient.Client, logger log.Logger, service *service.Service) (*serviceHeartbeat, error) {
	s := &serviceHeartbeat{
		instanceID: instanceID,
		timeToLive: timeToLive,
		raClient:   raClient,
		done:       make(chan struct{}, 1),
		logger:     logger.With("service-id", instanceID.String()),
		service:    service,
	}
	heartbeatValidUntil, err := s.updateServiceMetadata(true)
	if err != nil {
		return nil, fmt.Errorf("cannot update service metadata: %w", err)
	}
	s.heartbeatValidUntil = heartbeatValidUntil
	return s, nil
}

// updateServiceMetadata updates service metadata in resource aggregate.
func (s *serviceHeartbeat) updateServiceMetadata(register bool) (time.Time, error) {
	// set deadline to prevent blocking the service
	deadline := s.heartbeatValidUntil.Add(s.timeToLive)
	if s.heartbeatValidUntil.IsZero() {
		deadline = time.Now().Add(s.timeToLive)
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	resp, err := s.raClient.UpdateServiceMetadata(ctx, &commands.UpdateServiceMetadataRequest{
		Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
			Heartbeat: &commands.ServiceHeartbeat{
				ServiceId:  s.instanceID.String(),
				TimeToLive: s.timeToLive.Nanoseconds(),
				Timestamp:  time.Now().UnixNano(),
				Register:   register,
			},
		},
	})
	if err == nil {
		s.heartbeatValidUntil = pkgTime.Unix(0, resp.GetHeartbeatValidUntil())
		return s.heartbeatValidUntil, nil
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

func (s *serviceHeartbeat) tryUpdateServiceMetadata(now time.Time) error {
	var err error
	var isOffline bool
	switch {
	case now.After(s.heartbeatValidUntil):
		err = fmt.Errorf("service is offline from time %v", s.heartbeatValidUntil)
		isOffline = true
	default:
		var heartbeatValidUntil time.Time
		heartbeatValidUntil, err = s.updateServiceMetadata(false)
		if err == nil {
			s.logger.Debugf("service metadata updated, heartbeat valid until: %v", heartbeatValidUntil)
			s.heartbeatValidUntil = heartbeatValidUntil
			return nil
		}
	}
	err = fmt.Errorf("cannot update service metadata: %w", err)
	if isOffline || needToShutdownService(err) {
		// Kill the service to prevent inconsistent state propagation of connected devices through this service.
		s.logger.Infof("killing the service to prevent inconsistent state: %v", err)
		s.service.SigTerm()
		// to prevent too frequent killing the service
		time.Sleep(time.Second)
	} else {
		s.logger.Warnf("%v", err)
	}
	return nil
}

// Serve starts serviceHeartbeat. It will update service metadata in two times in timeToLive.
func (s *serviceHeartbeat) Serve() error {
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
		nextWake := time.Until(s.heartbeatValidUntil) / numberUpdatesInTimeToLive
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

// Close stops serviceHeartbeat.
func (s *serviceHeartbeat) Close() error {
	select {
	case s.done <- struct{}{}:
	default:
	}
	return nil
}
