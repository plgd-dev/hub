package grpc

import (
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/snapshot-service/pb"
	"github.com/plgd-dev/hub/v2/snapshot-service/store"
)

// SnapshotServiceServer handles incoming requests.
type SnapshotServiceServer struct {
	pb.UnimplementedSnapshotServiceServer

	logger     log.Logger
	ownerClaim string
	store      store.Store
	hubID      string
}

func NewSnapshotServiceServer(ownerClaim string, hubID string, store store.Store, logger log.Logger) (*SnapshotServiceServer, error) {
	s := &SnapshotServiceServer{
		logger:     logger,
		ownerClaim: ownerClaim,
		store:      store,
		hubID:      hubID,
	}

	return s, nil
}

func (s *SnapshotServiceServer) Close() {
}
