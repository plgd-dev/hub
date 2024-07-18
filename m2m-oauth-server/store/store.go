package store

import (
	"context"
	"errors"
	"time"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"go.mongodb.org/mongo-driver/mongo"
)

type Iterator[T any] interface {
	Next(ctx context.Context, v *T) bool
	Err() error
}

type (
	Process[T any] func(v *T) error
	ProcessTokens  = Process[pb.Token]
)

var (
	ErrNotSupported    = errors.New("not supported")
	ErrNotFound        = errors.New("not found")
	ErrNotModified     = errors.New("not modified")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrPartialDelete   = errors.New("some errors occurred while deleting")
)

func IsDuplicateKeyError(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}

type BsonMapper interface {
	FromBsonMap(m map[string]interface{}) error
}

type MongoIterator[T any] struct {
	Cursor *mongo.Cursor
}

func (i *MongoIterator[T]) Next(ctx context.Context, s *T) bool {
	if !i.Cursor.Next(ctx) {
		return false
	}
	var tmp interface{} = s
	if tmp, ok := tmp.(BsonMapper); ok {
		var mapValue map[string]interface{}
		err := i.Cursor.Decode(&mapValue)
		if err == nil {
			err = tmp.FromBsonMap(mapValue)
		}
		return err == nil
	}
	err := i.Cursor.Decode(s)
	return err == nil
}

func (i *MongoIterator[T]) Err() error {
	return i.Cursor.Err()
}

type Store interface {
	// CreateToken creates a new token. If the token already exists, it will throw an error.
	CreateToken(ctx context.Context, owner string, token *pb.Token) (*pb.Token, error)
	// GetTokens loads tokens from the database.
	GetTokens(ctx context.Context, owner string, query *pb.GetTokensRequest, p ProcessTokens) error

	// DeleteTokens deletes blacklisted expired tokens from the database.
	DeleteTokens(ctx context.Context, now time.Time) error

	// Set tokens as blacklisted
	BlacklistTokens(ctx context.Context, owner string, req *pb.BlacklistTokensRequest) (*pb.BlacklistTokensResponse, error)

	Close(ctx context.Context) error
}
