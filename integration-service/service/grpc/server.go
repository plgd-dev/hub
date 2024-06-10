package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/integration-service/pb"
	"github.com/plgd-dev/hub/v2/integration-service/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

// IntegrationServiceServer handles incoming requests.
type IntegrationServiceServer struct {
	pb.UnimplementedIntegrationServiceServer

	logger     log.Logger
	ownerClaim string
	store      store.Store
	hubID      string
}

func NewIntegrationServiceServer(ownerClaim string, hubID string, store store.Store, logger log.Logger) (*IntegrationServiceServer, error) {
	s := &IntegrationServiceServer{
		logger:     logger,
		ownerClaim: ownerClaim,
		store:      store,
		hubID:      hubID,
	}

	return s, nil
}

func (s *IntegrationServiceServer) Close() {
}

func (s *IntegrationServiceServer) GetConfiguration(req *pb.GetConfigurationRequest, srv pb.IntegrationService_GetConfigurationServer) error {

	var sub pb.Configuration

	s.store.GetRecord(context.Background(), req.GetId(), &store.GetConfigurationRequest{
		Id: req.GetId(),
	}, &sub)

	if err := srv.Send(&sub); err != nil {
		return err
	}

	return nil
}
