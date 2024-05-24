package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
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

func (s *SnippetServiceServer) CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	return s.store.CreateConfiguration(ctx, conf)
}

func (s *SnippetServiceServer) UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	return s.store.UpdateConfiguration(ctx, conf)
}

func (s *SnippetServiceServer) Close(ctx context.Context) error {
	return s.store.Close(ctx)
}
