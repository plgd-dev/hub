package grpc

import (
	"github.com/plgd-dev/hub/v2/pkg/log"
	//"github.com/plgd-dev/hub/v2/integration-service/pb"
	"github.com/plgd-dev/hub/v2/integration-service/store"
)

// SnapshotServiceServer handles incoming requests.
type IntegrationServiceServer struct {
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
