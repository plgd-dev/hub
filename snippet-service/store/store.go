package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	GetLatestConditionsQuery struct {
		DeviceID           string
		ResourceHref       string
		ResourceTypeFilter []string
	}
)

type Iterator[T any] interface {
	Next(ctx context.Context, v *T) bool
	Err() error
}

type (
	Process[T any]                func(v *T) error
	ProccessAppliedConfigurations = Process[AppliedConfiguration]
	ProcessConfigurations         = Process[Configuration]
	ProcessConditions             = Process[Condition]
)

var (
	ErrNotSupported    = errors.New("not supported")
	ErrNotFound        = errors.New("not found")
	ErrNotModified     = errors.New("not modified")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrPartialDelete   = errors.New("some errors occurred while deleting")
)

func errInvalidArgument(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
}

func IsDuplicateKeyError(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}

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
	CreateCondition(ctx context.Context, condition *pb.Condition) (*pb.Condition, error)
	// UpdateCondition updates an existing condition.
	UpdateCondition(ctx context.Context, condition *pb.Condition) (*pb.Condition, error)
	// GetConditions loads conditions from the database.
	GetConditions(ctx context.Context, owner string, query *pb.GetConditionsRequest, p ProcessConditions) error
	// DeleteConditions deletes conditions from the database.
	DeleteConditions(ctx context.Context, owner string, query *pb.DeleteConditionsRequest) error
	// InsertConditions inserts conditions into the database.
	InsertConditions(ctx context.Context, conditions ...*Condition) error
	// GetLatestEnabledConditions finds latest conditions that match the query.
	GetLatestEnabledConditions(ctx context.Context, owner string, query *GetLatestConditionsQuery, p ProcessConditions) error

	// CreateConfiguration creates a new configuration in the database.
	CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error)
	// UpdateConfiguration updates an existing configuration in the database.
	UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error)
	// GetConfigurations loads a configurations from the database.
	GetConfigurations(ctx context.Context, owner string, query *pb.GetConfigurationsRequest, p ProcessConfigurations) error
	// DeleteConfigurations deletes configurations from the database.
	DeleteConfigurations(ctx context.Context, owner string, query *pb.DeleteConfigurationsRequest) error
	// InsertConditions inserts conditions into the database.
	InsertConfigurations(ctx context.Context, configurations ...*Configuration) error
	// GetLatestConfigurationsByID finds latest configurations by their IDs.
	GetLatestConfigurationsByID(ctx context.Context, owner string, ids []string, p ProcessConfigurations) error

	// GetAppliedConfigurations loads applied device configurations from the database.
	GetAppliedConfigurations(ctx context.Context, owner string, query *pb.GetAppliedConfigurationsRequest, p ProccessAppliedConfigurations) error
	// DeleteAppliedConfigurations deletes applied device configurations from the database.
	DeleteAppliedConfigurations(ctx context.Context, owner string, query *pb.DeleteAppliedConfigurationsRequest) error
	// CreateAppliedConfiguration creates a new applied device configuration in the database.
	//
	// If the configuration with given deviceID and configurationID already exists, it will throw an error, unless the force flag is set to true.
	//
	// The first return value is the created applied device configuration. The second return value is the applied device configuration that was replaced if the force flag was set to true.
	CreateAppliedConfiguration(ctx context.Context, conf *pb.AppliedConfiguration, force bool) (*pb.AppliedConfiguration, *pb.AppliedConfiguration, error)
	// InsertAppliedConditions inserts applied configurations into the database.
	InsertAppliedConfigurations(ctx context.Context, configurations ...*AppliedConfiguration) error
	// UpdateAppliedConfigurationResource updates an existing applied device configuration resource in the database.
	UpdateAppliedConfigurationResource(ctx context.Context, owner string, query UpdateAppliedConfigurationResourceRequest) (*pb.AppliedConfiguration, error)
	// GetPendingAppliedConfigurationResourceUpdates loads applied device configuration with expired (validUntil <= now) resource updates from the database.
	GetPendingAppliedConfigurationResourceUpdates(ctx context.Context, expiredOnly bool, p ProccessAppliedConfigurations) (int64, error)

	Close(ctx context.Context) error
}
