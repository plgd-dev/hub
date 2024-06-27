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

func TestStoreUpdateAppliedConfiguration(t *testing.T) {
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
			CorrelationId: "corID",
		},
	}
	owner := "owner1"
	appliedConf, _, err := s.CreateAppliedConfiguration(ctx, &pb.AppliedConfiguration{
		Id:       id,
		DeviceId: "dev1",
		ConfigurationId: &pb.AppliedConfiguration_RelationTo{
			Id: confID,
		},
		ExecutedBy: pb.MakeExecutedByConditionId(condID, 0),
		Resources:  resources,
		Owner:      owner,
	}, false)
	require.NoError(t, err)

	appliedConf2, _, err := s.CreateAppliedConfiguration(ctx, &pb.AppliedConfiguration{
		Id:       uuid.NewString(),
		DeviceId: "dev2",
		ConfigurationId: &pb.AppliedConfiguration_RelationTo{
			Id: confID,
		},
		ExecutedBy: pb.MakeExecutedByConditionId(condID, 0),
		Resources:  resources,
		Owner:      owner,
	}, false)
	require.NoError(t, err)

	type args struct {
		update *pb.AppliedConfiguration
	}
	tests := []struct {
		name    string
		args    args
		want    func(*pb.AppliedConfiguration)
		wantErr bool
	}{
		{
			name: "missing Id",
			args: args{
				update: func() *pb.AppliedConfiguration {
					c := appliedConf.Clone()
					c.Id = ""
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "non-existing Id",
			args: args{
				update: func() *pb.AppliedConfiguration {
					c := appliedConf.Clone()
					c.Id = uuid.NewString()
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "non-matching owner",
			args: args{
				update: func() *pb.AppliedConfiguration {
					c := appliedConf.Clone()
					c.Owner = "owner2"
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "duplicit deviceID+configurationID",
			args: args{
				update: func() *pb.AppliedConfiguration {
					c := appliedConf.Clone()
					c.Id = appliedConf2.GetId()
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				update: &pb.AppliedConfiguration{
					Id:       appliedConf.GetId(),
					Owner:    appliedConf.GetOwner(),
					DeviceId: "dev2",
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{
						Id:      uuid.NewString(),
						Version: appliedConf.GetConfigurationId().GetVersion() + 1,
					},
					ExecutedBy: pb.MakeExecutedByOnDemand(),
					Resources: []*pb.AppliedConfiguration_Resource{
						{
							Href:          "/test/2",
							CorrelationId: "corID2",
							Status:        pb.AppliedConfiguration_Resource_PENDING,
						},
					},
				},
			},
			want: func(updated *pb.AppliedConfiguration) {
				require.Equal(t, appliedConf.GetId(), updated.GetId())
				require.Equal(t, "dev2", updated.GetDeviceId())
				require.Equal(t, appliedConf.GetOwner(), updated.GetOwner())
				require.Equal(t, pb.MakeExecutedByOnDemand(), updated.GetExecutedBy())
				require.Len(t, updated.GetResources(), 1)
				require.Equal(t, "/test/2", updated.GetResources()[0].GetHref())
				require.Equal(t, "corID2", updated.GetResources()[0].GetCorrelationId())
				require.Equal(t, pb.AppliedConfiguration_Resource_PENDING, updated.GetResources()[0].GetStatus())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := s.UpdateAppliedConfiguration(ctx, tt.args.update)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(updated)
		})
	}
}

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
		ConfigurationId: &pb.AppliedConfiguration_RelationTo{
			Id: confID,
		},
		ExecutedBy: pb.MakeExecutedByConditionId(condID, 0),
		Resources:  resources,
		Owner:      owner,
	}, false)
	require.NoError(t, err)

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
