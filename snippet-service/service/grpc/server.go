package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/updater"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SnippetServiceServer handles incoming requests.
type SnippetServiceServer struct {
	pb.UnimplementedSnippetServiceServer

	store           store.Store
	resourceUpdater *updater.ResourceUpdater
	ownerClaim      string
	hubID           string
	logger          log.Logger
}

func NewSnippetServiceServer(store store.Store, resourceUpdater *updater.ResourceUpdater, ownerClaim string, hubID string, logger log.Logger) *SnippetServiceServer {
	return &SnippetServiceServer{
		store:           store,
		resourceUpdater: resourceUpdater,
		logger:          logger,
		ownerClaim:      ownerClaim,
		hubID:           hubID,
	}
}

func (s *SnippetServiceServer) checkOwner(ctx context.Context, owner string) (string, error) {
	ownerFromToken, err := pkgGrpc.OwnerFromTokenMD(ctx, s.ownerClaim)
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
	// increment version automatically by mongo
	conf.Version = 0
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

func getAllLatest() []*pb.IDFilter {
	return []*pb.IDFilter{
		{
			Version: &pb.IDFilter_Latest{
				Latest: true,
			},
		},
	}
}

func (s *SnippetServiceServer) GetConfigurations(req *pb.GetConfigurationsRequest, srv pb.SnippetService_GetConfigurationsServer) error {
	owner, err := s.checkOwner(srv.Context(), "")
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotGetConfigurations(err)))
	}
	req.IdFilter = append(req.GetIdFilter(), req.ConvertHTTPIDFilter()...)
	if len(req.GetIdFilter()) == 0 {
		// get all latest conditions by default
		req.IdFilter = getAllLatest()
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
	req.IdFilter = append(req.GetIdFilter(), req.ConvertHTTPIDFilter()...)

	count, err := s.store.DeleteConfigurations(ctx, owner, req)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotDeleteConfigurations(err)))
	}
	return &pb.DeleteConfigurationsResponse{
		Count: count,
	}, nil
}

func errCannotInvokeConfiguration(err error) error {
	return fmt.Errorf("cannot invoke configuration: %w", err)
}

func (s *SnippetServiceServer) InvokeConfiguration(ctx context.Context, req *pb.InvokeConfigurationRequest) (*pb.InvokeConfigurationResponse, error) {
	owner, err := s.checkOwner(ctx, "")
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotInvokeConfiguration(err)))
	}
	token, errT := pkgGrpc.TokenFromMD(ctx)
	// we must have token for communication by raClient
	if errT != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotInvokeConfiguration(errT)))
	}
	// TODO: the query parameter must match the req.ConfigurationId
	appliedConf, err := s.resourceUpdater.InvokeConfiguration(ctx, token, owner, req)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotInvokeConfiguration(err)))
	}
	return &pb.InvokeConfigurationResponse{
		AppliedConfigurationId: appliedConf.GetId(),
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
	// increment version automatically by mongo
	condition.Version = 0
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
	req.IdFilter = append(req.GetIdFilter(), req.ConvertHTTPIDFilter()...)
	if len(req.GetIdFilter()) == 0 && len(req.GetConfigurationIdFilter()) == 0 {
		// get all latest conditions by default
		req.IdFilter = getAllLatest()
	}

	err = s.store.GetConditions(srv.Context(), owner, req, func(c *store.Condition) error {
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
	req.IdFilter = append(req.GetIdFilter(), req.ConvertHTTPIDFilter()...)

	count, err := s.store.DeleteConditions(ctx, owner, req)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotDeleteConditions(err)))
	}
	return &pb.DeleteConditionsResponse{
		Count: count,
	}, nil
}

func errCannotCreateAppliedConfiguration(err error) error {
	return fmt.Errorf("cannot create applied configuration: %w", err)
}

func (s *SnippetServiceServer) CreateAppliedConfiguration(ctx context.Context, configuration *pb.AppliedConfiguration) (*pb.AppliedConfiguration, error) {
	owner, err := s.checkOwner(ctx, configuration.GetOwner())
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotCreateAppliedConfiguration(err)))
	}

	configuration.Owner = owner
	c, _, err := s.store.CreateAppliedConfiguration(ctx, configuration, false)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(getGRPCErrorCode(err), "%v", errCannotCreateAppliedConfiguration(err)))
	}
	return c, nil
}

func errCannotGetAppliedConfigurations(err error) error {
	return fmt.Errorf("cannot get applied configurations: %w", err)
}

func (s *SnippetServiceServer) GetAppliedConfigurations(req *pb.GetAppliedConfigurationsRequest, srv pb.SnippetService_GetAppliedConfigurationsServer) error {
	owner, err := s.checkOwner(srv.Context(), "")
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotGetAppliedConfigurations(err)))
	}

	req.ConditionIdFilter = append(req.GetConditionIdFilter(), req.ConvertHTTPConditionIdFilter()...)
	req.ConfigurationIdFilter = append(req.GetConfigurationIdFilter(), req.ConvertHTTPConfigurationIdFilter()...)

	err = s.store.GetAppliedConfigurations(srv.Context(), owner, req, func(c *store.AppliedConfiguration) error {
		return srv.Send(c.GetAppliedConfiguration())
	})
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotGetAppliedConfigurations(err)))
	}
	return nil
}

func errCannotDeleteAppliedConfigurations(err error) error {
	return fmt.Errorf("cannot delete applied configurations: %w", err)
}

func (s *SnippetServiceServer) DeleteAppliedConfigurations(ctx context.Context, req *pb.DeleteAppliedConfigurationsRequest) (*pb.DeleteAppliedConfigurationsResponse, error) {
	owner, err := s.checkOwner(ctx, "")
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "%v", errCannotDeleteAppliedConfigurations(err)))
	}
	count, err := s.store.DeleteAppliedConfigurations(ctx, owner, req)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "%v", errCannotDeleteAppliedConfigurations(err)))
	}
	return &pb.DeleteAppliedConfigurationsResponse{
		Count: count,
	}, nil
}
