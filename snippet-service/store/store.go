package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	IDKey                 = "_id"                // must match with Id field tag
	DeviceIDKey           = "deviceId"           // must match with DeviceId field tag
	OwnerKey              = "owner"              // must match with Owner field tag
	LatestKey             = "latest"             // must match with Latest field tag
	NameKey               = "name"               // must match with Name field tag
	VersionKey            = "version"            // must match with Version field tag
	VersionsKey           = "versions"           // must match with Versions field tag
	ResourcesKey          = "resources"          // must match with Resources field tag
	ConfigurationIDKey    = "configurationId"    // must match with ConfigurationId field tag
	EnabledKey            = "enabled"            // must match with Enabled field tag
	TimestampKey          = "timestamp"          // must match with Timestamp field tag
	ApiAccessTokenKey     = "apiAccessToken"     // must match with ApiAccessToken field tag
	DeviceIDFilterKey     = "deviceIdFilter"     // must match with Condition.DeviceIdFilter tag
	ResourceHrefFilterKey = "resourceHrefFilter" // must match with Condition.ResourceHrefFilter tag
	JqExpressionFilterKey = "jqExpressionFilter" // must match with Condition.JqExpressionFilter tag
	ResourceTypeFilterKey = "resourceTypeFilter" // must match with Condition.ResourceTypeFilter tag
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
	Process[T any]        func(v *T) error
	ProcessConfigurations = Process[Configuration]
	ProcessConditions     = Process[Condition]
)

var (
	ErrNotSupported    = errors.New("not supported")
	ErrNotFound        = errors.New("not found")
	ErrInvalidArgument = errors.New("invalid argument")
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
	DeleteConditions(ctx context.Context, owner string, query *pb.DeleteConditionsRequest) (int64, error)
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
	DeleteConfigurations(ctx context.Context, owner string, query *pb.DeleteConfigurationsRequest) (int64, error)

	// CreateAppliedDeviceConfiguration creates a new applied device configuration in the database.
	CreateAppliedDeviceConfiguration(ctx context.Context, conf *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error)

	Close(ctx context.Context) error
}
