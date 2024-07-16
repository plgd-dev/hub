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
func (s *Store) DeleteTokens(context.Context) error {
	return store.ErrNotSupported
}
func (s *Store) GetBlacklistedTokens(context.Context, string, *pb.GetBlacklistedTokensRequest, store.ProcessTokens) error {
	return store.ErrNotSupported
}
func (s *Store) BlacklistTokens(context.Context, string, *pb.BlacklistTokensRequest) (*pb.BlacklistTokensResponse, error) {
	return nil, store.ErrNotSupported
}
