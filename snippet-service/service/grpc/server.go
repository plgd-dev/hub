package grpc

import (
	"context"
	"errors"

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

func NewSnippetServiceServer(ownerClaim string, hubID string, store store.Store, logger log.Logger) (*SnippetServiceServer, error) {
	s := &SnippetServiceServer{
		logger:     logger,
		ownerClaim: ownerClaim,
		store:      store,
		hubID:      hubID,
	}

	return s, nil
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

func (s *SnippetServiceServer) CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	owner, err := s.checkOwner(ctx, conf.GetOwner())
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "cannot create configuration: %v", err))
	}

	conf.Owner = owner
	c, err := s.store.CreateConfiguration(ctx, conf)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "cannot create configuration: %v", err))
	}
	return c, nil
}

func (s *SnippetServiceServer) UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	owner, err := s.checkOwner(ctx, conf.GetOwner())
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "cannot update configuration: %v", err))
	}

	conf.Owner = owner
	c, err := s.store.UpdateConfiguration(ctx, conf)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "cannot update configuration: %v", err))
	}
	return c, nil
}

func (s *SnippetServiceServer) GetConfigurations(req *pb.GetConfigurationsRequest, srv pb.SnippetService_GetConfigurationsServer) error {
	owner, err := s.checkOwner(srv.Context(), "")
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "cannot get configurations: %v", err))
	}

	err = s.store.GetConfigurations(srv.Context(), owner, req, func(ctx context.Context, iter store.Iterator[store.Configuration]) error {
		storedCfg := store.Configuration{}
		for iter.Next(ctx, &storedCfg) {
			for _, version := range storedCfg.Versions {
				errS := srv.Send(&pb.Configuration{
					Id:        storedCfg.Id,
					Owner:     storedCfg.Owner,
					Name:      storedCfg.Name,
					Version:   version.Version,
					Resources: version.Resources,
				})
				if errS != nil {
					return errS
				}
			}
		}
		return nil
	})
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.Internal, "cannot get configurations: %v", err))
	}
	return nil
}

func (s *SnippetServiceServer) DeleteConfigurations(ctx context.Context, req *pb.DeleteConfigurationsRequest) (*pb.DeleteConfigurationsResponse, error) {
	owner, err := s.checkOwner(ctx, "")
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.PermissionDenied, "cannot delete configurations: %v", err))
	}
	count, err := s.store.DeleteConfigurations(ctx, owner, req)
	if err != nil {
		return nil, s.logger.LogAndReturnError(status.Errorf(codes.Internal, "cannot delete configurations: %v", err))
	}
	return &pb.DeleteConfigurationsResponse{
		Count: count,
	}, nil
}

func (s *SnippetServiceServer) Close(ctx context.Context) error {
	return s.store.Close(ctx)
}
