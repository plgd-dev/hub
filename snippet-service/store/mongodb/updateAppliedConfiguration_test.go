package mongodb_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestStoreUpdateAppliedConfigurationResource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	id := uuid.NewString()
	confID := uuid.NewString()
	condID := uuid.NewString()
	resources := []*pb.AppliedConfiguration_Resource{
		{
			Href:          "/test/1",
			CorrelationId: "corID1",
			Status:        pb.AppliedConfiguration_Resource_PENDING,
		},
		{
			Href:          "/test/2",
			CorrelationId: "corID2",
			Status:        pb.AppliedConfiguration_Resource_PENDING,
			ResourceUpdated: &events.ResourceUpdated{
				ResourceId: &commands.ResourceId{DeviceId: "deviceID", Href: "/test/2"},
			},
		},
		{
			Href:          "/test/3",
			CorrelationId: "corID3",
			Status:        pb.AppliedConfiguration_Resource_QUEUED,
		},
	}
	owner := "owner1"
	appliedConf, _, err := s.CreateAppliedConfiguration(ctx, &pb.AppliedConfiguration{
		Id:       id,
		DeviceId: "dev1",
		ConfigurationId: &pb.AppliedConfiguration_LinkedTo{
			Id: confID,
		},
		ExecutedBy: pb.MakeExecutedByConditionId(condID, 0),
		Resources:  resources,
		Owner:      owner,
	}, false)
	require.NoError(t, err)

	// error - missing applied configuration ID
	_, err = s.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		Resource: &pb.AppliedConfiguration_Resource{
			Href:          "/test/1",
			CorrelationId: "corID1",
			Status:        pb.AppliedConfiguration_Resource_DONE,
		},
	})
	require.Error(t, err)

	// error - missing resource
	_, err = s.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: id,
	})
	require.Error(t, err)

	// error - missing resource href
	_, err = s.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: id,
		Resource: &pb.AppliedConfiguration_Resource{
			CorrelationId: "corID1",
			Status:        pb.AppliedConfiguration_Resource_DONE,
		},
	})
	require.Error(t, err)

	// error - missing resource correlationID
	_, err = s.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: id,
		Resource: &pb.AppliedConfiguration_Resource{
			Href:   "/test/1",
			Status: pb.AppliedConfiguration_Resource_DONE,
		},
	})
	require.Error(t, err)

	// error - invalid resource status
	_, err = s.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: id,
		Resource: &pb.AppliedConfiguration_Resource{
			Href:          "/test/1",
			CorrelationId: "corID1",
		},
	})
	require.Error(t, err)
	_, err = s.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: id,
		Resource: &pb.AppliedConfiguration_Resource{
			Href:          "/test/1",
			CorrelationId: "corID1",
			Status:        pb.AppliedConfiguration_Resource_UNSPECIFIED,
		},
	})
	require.Error(t, err)

	updatedAppliedConf, err := s.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: id,
		StatusFilter:           []pb.AppliedConfiguration_Resource_Status{pb.AppliedConfiguration_Resource_PENDING},
		Resource: &pb.AppliedConfiguration_Resource{
			Href:          "/test/1",
			CorrelationId: "corID1",
			Status:        pb.AppliedConfiguration_Resource_DONE,
		},
	})
	require.NoError(t, err)
	wantAppliedConf := appliedConf.Clone()
	wantAppliedConf.Resources[0].Status = pb.AppliedConfiguration_Resource_DONE
	test.CmpAppliedDeviceConfiguration(t, wantAppliedConf, updatedAppliedConf, true)

	// /test/1 is no longer in pending state, so additional update should fail
	_, err = s.UpdateAppliedConfigurationResource(ctx, owner, store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: id,
		StatusFilter:           []pb.AppliedConfiguration_Resource_Status{pb.AppliedConfiguration_Resource_PENDING},
		Resource: &pb.AppliedConfiguration_Resource{
			Href:          "/test/1",
			CorrelationId: "corID1",
			Status:        pb.AppliedConfiguration_Resource_TIMEOUT,
		},
	})
	require.ErrorIs(t, err, store.ErrNotModified)

	// mismatched owner
	_, err = s.UpdateAppliedConfigurationResource(ctx, "mismatch", store.UpdateAppliedConfigurationResourceRequest{
		AppliedConfigurationID: id,
		StatusFilter:           []pb.AppliedConfiguration_Resource_Status{pb.AppliedConfiguration_Resource_PENDING},
		Resource: &pb.AppliedConfiguration_Resource{
			Href:          "/test/2",
			CorrelationId: "corID2",
			Status:        pb.AppliedConfiguration_Resource_DONE,
		},
	})
	require.Error(t, err)
}
