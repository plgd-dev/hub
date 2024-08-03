package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/store"
)

func (s *Store) CreateToken(context.Context, *pb.Token) (*pb.Token, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) GetTokens(context.Context, string, *pb.GetTokensRequest, store.ProcessTokens) error {
	return store.ErrNotSupported
}

func (s *Store) DeleteBlacklistedTokens(context.Context) error {
	return store.ErrNotSupported
}

func (s *Store) DeleteTokens(context.Context, string, *pb.DeleteTokensRequest) (*pb.DeleteTokensResponse, error) {
	return nil, store.ErrNotSupported
}
