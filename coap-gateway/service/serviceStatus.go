package service

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/log"
	raClient "github.com/plgd-dev/hub/v2/resource-aggregate/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const numberUpdatesInTimeToLive = 2

type serviceStatus struct {
	timeToLive time.Duration
	raClient   *raClient.Client
	done       chan struct{}
	instanceID uuid.UUID
	logger     log.Logger
}

// NewServiceStatus creates new serviceStatus instance. It will update service metadata in two times in timeToLive.
// If it fails to update service metadata in two times in row, it will kill the service. Because resource aggregate
// can't be sure if the service is still alive. If any other services updates service metadata, it can consider this
// service as dead. And all devices connected to this service will be marked as offline.
func newServiceStatus(instanceID uuid.UUID, timeToLive time.Duration, raClient *raClient.Client, logger log.Logger) *serviceStatus {
	return &serviceStatus{
		instanceID: instanceID,
		timeToLive: timeToLive,
		raClient:   raClient,
		done:       make(chan struct{}, 1),
		logger:     logger.With("service-id", instanceID.String()),
	}
}

// updateServiceMetadata updates service metadata in resource aggregate with timeout = timeToLive/2.
func (s *serviceStatus) updateServiceMetadata() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeToLive/numberUpdatesInTimeToLive)
	defer cancel()

	_, err := s.raClient.UpdateServiceMetadata(ctx, &commands.UpdateServiceMetadataRequest{
		Update: &commands.UpdateServiceMetadataRequest_Status{
			Status: &commands.ServiceStatus{
				Id:         s.instanceID.String(),
				TimeToLive: s.timeToLive.Nanoseconds(),
			},
		},
	})
	return err
}

func needToShutdownService(err error) bool {
	s, ok := status.FromError(err)
	if ok {
		return s.Code() == codes.FailedPrecondition
	}
	return false
}

// Serve starts serviceStatus. It will update service metadata in two times in timeToLive.
func (s *serviceStatus) Serve() error {
	ticker := time.NewTicker(s.timeToLive / numberUpdatesInTimeToLive)
	defer ticker.Stop()
	failures := 0
	for {
		var err error
		if failures < numberUpdatesInTimeToLive {
			err = s.updateServiceMetadata()
			if err == nil {
				failures = 0
			}
		} else {
			err = fmt.Errorf("service is in inconsistent state")
		}
		if err != nil {
			err = fmt.Errorf("cannot update service metadata: %w", err)
			failures++
			if failures >= numberUpdatesInTimeToLive || needToShutdownService(err) {
				failures += numberUpdatesInTimeToLive
				// Kill the service to prevent inconsistent state propagation of connected devices through this service.
				s.logger.Infof("killing the service to prevent inconsistent state: %v", err)
				errKill := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
				if errKill == nil {
					return err
				}
				s.logger.Errorf("cannot kill service: %w", errKill)
			} else {
				s.logger.Warnf("%v", err)
			}
		}
		select {
		case <-s.done:
			return nil
		case <-ticker.C:
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
