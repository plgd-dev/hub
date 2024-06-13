package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SnippetServiceServer handles incoming requests.
type SnippetServiceServer struct {
	pb.UnimplementedSnippetServiceServer

	logger     log.Logger
	ownerClaim string
	store      store.Store
	hubID      string
}

func NewSnippetServiceServer(ownerClaim string, hubID string, store store.Store, logger log.Logger) *SnippetServiceServer {
	return &SnippetServiceServer{
		logger:     logger,
		ownerClaim: ownerClaim,
		store:      store,
		hubID:      hubID,
	}
}

func (s *SnippetServiceServer) checkOwner(ctx context.Context, owner string) (string, error) {
	ownerFromToken, err := grpc.OwnerFromTokenMD(ctx, s.ownerClaim)
	if err != nil {
		return "", err
	}
	if owner != "" && ownerFromToken != owner {
		return "", errors.New("owner mismatch")
	}
	return ownerFromToken, nil
}

func getGRPCErrorCode(err error) codes.Code {
	if errors.Is(err, store.ErrInvalidArgument) {
		return codes.InvalidArgument
	}
	return codes.Internal
}

func errCannotCreateConfiguration(err error) error {
	return fmt.Errorf("cannot get configuration: %w", err)
}

func (s *SnippetServiceServer) CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	owner, err := s.checkOwner(ctx, conf.GetOwner())
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotCreateConfiguration(err)))
	}

	conf.Owner = owner
	c, err := s.store.CreateConfiguration(ctx, conf)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(getGRPCErrorCode(err), "%v", errCannotCreateConfiguration(err)))
	}
	return c, nil
}

func errCannotUpdateConfiguration(err error) error {
	return fmt.Errorf("cannot update configuration: %w", err)
}

func (s *SnippetServiceServer) UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	owner, err := s.checkOwner(ctx, conf.GetOwner())
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotUpdateConfiguration(err)))
	}

	conf.Owner = owner
	c, err := s.store.UpdateConfiguration(ctx, conf)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(getGRPCErrorCode(err), "%v", errCannotUpdateConfiguration(err)))
	}
	return c, nil
}

func errCannotGetConfigurations(err error) error {
	return fmt.Errorf("cannot get configurations: %w", err)
}

func sendConfiguration(srv pb.SnippetService_GetConfigurationsServer, c *store.Configuration) error {
	var lastVersion *store.ConfigurationVersion
	for i := range c.Versions {
		err := srv.Send(c.GetConfiguration(i))
		if err != nil {
			return err
		}
		lastVersion = &c.Versions[i]
	}
	if c.Latest == nil {
		return nil
	}
	latest, err := c.GetLatest()
	if err != nil {
		return err
	}
	if lastVersion != nil && lastVersion.Version == latest.GetVersion() {
		// already sent when iterating over versions array
		return nil
	}
	return srv.Send(latest)
}

func (s *SnippetServiceServer) GetConfigurations(req *pb.GetConfigurationsRequest, srv pb.SnippetService_GetConfigurationsServer) error {
	owner, err := s.checkOwner(srv.Context(), "")
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotGetConfigurations(err)))
	}

	err = s.store.GetConfigurations(srv.Context(), owner, req, func(c *store.Configuration) error {
		return sendConfiguration(srv, c)
	})
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotGetConfigurations(err)))
	}
	return nil
}

func errCannotDeleteConfigurations(err error) error {
	return fmt.Errorf("cannot delete configurations: %w", err)
}

func (s *SnippetServiceServer) DeleteConfigurations(ctx context.Context, req *pb.DeleteConfigurationsRequest) (*pb.DeleteConfigurationsResponse, error) {
	owner, err := s.checkOwner(ctx, "")
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotDeleteConfigurations(err)))
	}
	count, err := s.store.DeleteConfigurations(ctx, owner, req)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotDeleteConfigurations(err)))
	}
	return &pb.DeleteConfigurationsResponse{
		Count: count,
	}, nil
}

func errCannotCreateCondition(err error) error {
	return fmt.Errorf("cannot create condition: %w", err)
}

func (s *SnippetServiceServer) CreateCondition(ctx context.Context, condition *pb.Condition) (*pb.Condition, error) {
	owner, err := s.checkOwner(ctx, condition.GetOwner())
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotCreateCondition(err)))
	}

	condition.Owner = owner
	c, err := s.store.CreateCondition(ctx, condition)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(getGRPCErrorCode(err), "%v", errCannotCreateCondition(err)))
	}
	return c, nil
}

func errCannotUpdateCondition(err error) error {
	return fmt.Errorf("cannot update condition: %w", err)
}

func (s *SnippetServiceServer) UpdateCondition(ctx context.Context, condition *pb.Condition) (*pb.Condition, error) {
	owner, err := s.checkOwner(ctx, condition.GetOwner())
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotUpdateCondition(err)))
	}

	condition.Owner = owner
	c, err := s.store.UpdateCondition(ctx, condition)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(getGRPCErrorCode(err), "%v", errCannotUpdateCondition(err)))
	}
	return c, nil
}

func errCannotGetConditions(err error) error {
	return fmt.Errorf("cannot get conditions: %w", err)
}

func sendCondition(srv pb.SnippetService_GetConditionsServer, c *store.Condition) error {
	var lastVersion *store.ConditionVersion
	for i := range c.Versions {
		err := srv.Send(c.GetCondition(i))
		if err != nil {
			return err
		}
		lastVersion = &c.Versions[i]
	}
	if c.Latest == nil {
		return nil
	}
	latest, err := c.GetLatest()
	if err != nil {
		return err
	}
	if lastVersion != nil && lastVersion.Version == latest.GetVersion() {
		// already sent when iterating over versions array
		return nil
	}
	return srv.Send(latest)
}

func (s *SnippetServiceServer) GetConditions(req *pb.GetConditionsRequest, srv pb.SnippetService_GetConditionsServer) error {
	owner, err := s.checkOwner(srv.Context(), "")
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotGetConditions(err)))
	}

	err = s.store.GetConditions(srv.Context(), owner, req, func(c *store.Condition) error {
		return sendCondition(srv, c)
	})
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotGetConditions(err)))
	}
	return nil
}

func (s *SnippetServiceServer) GetLatestEnabledConditions(req *store.GetLatestConditionsQuery, srv pb.SnippetService_GetConditionsServer) error {
	owner, err := s.checkOwner(srv.Context(), "")
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotGetConditions(err)))
	}

	err = s.store.GetLatestEnabledConditions(srv.Context(), owner, req, func(c *store.Condition) error {
		return sendCondition(srv, c)
	})
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotGetConditions(err)))
	}
	return nil
}

func errCannotDeleteConditions(err error) error {
	return fmt.Errorf("cannot delete conditions: %w", err)
}

func (s *SnippetServiceServer) DeleteConditions(ctx context.Context, req *pb.DeleteConditionsRequest) (*pb.DeleteConditionsResponse, error) {
	owner, err := s.checkOwner(ctx, "")
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotDeleteConditions(err)))
	}
	count, err := s.store.DeleteConditions(ctx, owner, req)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotDeleteConditions(err)))
	}
	return &pb.DeleteConditionsResponse{
		Count: count,
	}, nil
}

func (s *SnippetServiceServer) Close(ctx context.Context) error {
	return s.store.Close(ctx)
}
