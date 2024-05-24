package store

import (
	"context"
	"errors"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	IDKey      = "id"      // must match with Id field tag
	VersionKey = "version" // must match with Version field tag
	OwnerKey   = "owner"   // must match with Owner field tag
	// DeviceIDFilterKey     = "deviceIdFilter"     // must match with Condition.DeviceIdFilter tag
	// ResourceHrefFilterKey = "resourceHrefFilter" // must match with Condition.ResourceHrefFilter tag
	// ResourceTypeFilterKey = "resourceTypeFilter" // must match with Condition.ResourceTypeFilter tag

)

// type (
//	ConditionsQuery struct {
//		DeviceID           string
//		ResourceHref       string
//		ResourceTypeFilter []string
//	}
// )

type Iterator[T any] interface {
	Next(ctx context.Context, v *T) bool
	Err() error
}

type (
	// LoadConditionsFunc     = func(ctx context.Context, iter Iterator[Condition]) (err error)
	GetConfigurationsFunc = func(ctx context.Context, iter Iterator[Configuration]) (err error)
)

var (
	ErrNotSupported    = errors.New("not supported")
	ErrNotFound        = errors.New("not found")
	ErrInvalidArgument = errors.New("invalid argument")
)

type MongoIterator[T any] struct {
	Cursor *mongo.Cursor
}

func (i *MongoIterator[T]) Next(ctx context.Context, s *T) bool {
	if !i.Cursor.Next(ctx) {
		return false
	}
	err := i.Cursor.Decode(s)
	return err == nil
}

func (i *MongoIterator[T]) Err() error {
	return i.Cursor.Err()
}

type Store interface {
	// CreateCondition creates a new condition. If the condition already exists, it will throw an error.
	// CreateCondition(ctx context.Context, condition *Condition) error
	// UpdateSigningRecord updates an existing signing record. If the record does not exist, it will create a new one.
	// UpdateSigningRecord(ctx context.Context, record *SigningRecord) error
	// DeleteSigningRecords(ctx context.Context, ownerID string, query *DeleteSigningRecordsQuery) (int64, error)
	// LoadConditions(ctx context.Context, ownerID string, query *ConditionsQuery, h LoadConditionsFunc) error

	// CreateConfiguration creates a new configuration in the database.
	CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error)
	// UpdateConfiguration updates an existing configuration in the database.
	UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error)
	// GetConfigurations loads a configuration from the database.
	GetConfigurations(ctx context.Context, owner string, query *pb.GetConfigurationsRequest, h GetConfigurationsFunc) error
	// DeleteConfigurations deletes configurations from the database.
	DeleteConfigurations(ctx context.Context, owner string, query *pb.DeleteConfigurationsRequest) (int64, error)

	Close(ctx context.Context) error
}
