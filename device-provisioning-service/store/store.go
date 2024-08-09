package store

import (
	"context"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
)

type (
	ProvisioningRecordsQuery = pb.GetProvisioningRecordsRequest
	EnrollmentGroupsQuery    = pb.GetEnrollmentGroupsRequest
	HubsQuery                = pb.GetHubsRequest
)

type ProvisioningRecordIter interface {
	Next(ctx context.Context, provisioningRecord *ProvisioningRecord) bool
	Err() error
}

type EnrollmentGroupIter interface {
	Next(ctx context.Context, enrollmentGroup *EnrollmentGroup) bool
	Err() error
}

type WatchEnrollmentGroupIter interface {
	Next(ctx context.Context) (event Event, id string, ok bool)
	Err() error
	Close() error
}

type WatchHubIter interface {
	Next(ctx context.Context) (event Event, id string, ok bool)
	Err() error
	Close() error
}

type HubIter interface {
	Next(ctx context.Context, hub *Hub) bool
	Err() error
}

type (
	LoadProvisioningRecordsFunc = func(ctx context.Context, iter ProvisioningRecordIter) (err error)
	LoadEnrollmentGroupsFunc    = func(ctx context.Context, iter EnrollmentGroupIter) (err error)
	LoadHubsFunc                = func(ctx context.Context, iter HubIter) (err error)
)

type Event string

const (
	EventDelete Event = "delete"
	EventUpdate Event = "update"
)

type Store interface {
	UpdateProvisioningRecord(ctx context.Context, owner string, sub *ProvisioningRecord) error
	DeleteProvisioningRecords(ctx context.Context, owner string, query *ProvisioningRecordsQuery) (int64, error)
	LoadProvisioningRecords(ctx context.Context, owner string, query *ProvisioningRecordsQuery, h LoadProvisioningRecordsFunc) error

	CreateEnrollmentGroup(ctx context.Context, owner string, enrollmentGroup *EnrollmentGroup) error
	UpdateEnrollmentGroup(ctx context.Context, owner string, enrollmentGroup *EnrollmentGroup) error
	DeleteEnrollmentGroups(ctx context.Context, owner string, query *EnrollmentGroupsQuery) (int64, error)
	LoadEnrollmentGroups(ctx context.Context, owner string, query *EnrollmentGroupsQuery, h LoadEnrollmentGroupsFunc) error
	// returned iterator need to be close after use.
	WatchEnrollmentGroups(ctx context.Context) (WatchEnrollmentGroupIter, error)

	CreateHub(ctx context.Context, owner string, hub *Hub) error
	UpdateHub(ctx context.Context, owner string, hub *Hub) error
	DeleteHubs(ctx context.Context, owner string, query *HubsQuery) (int64, error)
	LoadHubs(ctx context.Context, owner string, query *HubsQuery, h LoadHubsFunc) error
	// returned iterator need to be close after use.
	WatchHubs(ctx context.Context) (WatchHubIter, error)

	Close(ctx context.Context) error
}
