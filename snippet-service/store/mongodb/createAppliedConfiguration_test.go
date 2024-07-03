package mongodb_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	"github.com/stretchr/testify/require"
)

func TestStoreCreateAppliedConfiguration(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	appliedConfID1 := uuid.NewString()
	owner := "owner"
	deviceID := "deviceID1"
	confID1 := uuid.NewString()
	condID1 := uuid.NewString()
	executedBy1 := &pb.AppliedConfiguration_ConditionId{
		ConditionId: &pb.AppliedConfiguration_RelationTo{
			Id: condID1,
		},
	}
	resource1 := &pb.AppliedConfiguration_Resource{
		Href:          "/test/1",
		CorrelationId: "correlationID1",
		Status:        pb.AppliedConfiguration_Resource_DONE,
		ResourceUpdated: &events.ResourceUpdated{
			ResourceId: commands.NewResourceID(deviceID, hubTest.TestResourceLightInstanceHref("1")),
			Content: &commands.Content{
				CoapContentFormat: -1,
			},
			Status:        commands.Status_OK,
			AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
			ResourceTypes: hubTest.TestResourceLightInstanceResourceTypes,
		},
	}

	appliedConf1, _, err := s.CreateAppliedConfiguration(ctx, &pb.AppliedConfiguration{
		Id:              appliedConfID1,
		Owner:           owner,
		DeviceId:        deviceID,
		ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID1},
		ExecutedBy:      executedBy1,
		Resources:       []*pb.AppliedConfiguration_Resource{resource1},
	}, false)
	require.NoError(t, err)

	appliedConfID2 := uuid.NewString()
	confID2 := uuid.NewString()
	executedBy2 := &pb.AppliedConfiguration_OnDemand{
		OnDemand: true,
	}
	resource2 := &pb.AppliedConfiguration_Resource{
		Href:          "/test/2",
		CorrelationId: "correlationID2",
		Status:        pb.AppliedConfiguration_Resource_TIMEOUT,
	}
	appliedConfID3 := uuid.NewString()
	appliedConfID4 := uuid.NewString()
	confID3 := uuid.NewString()

	type args struct {
		adc   *pb.AppliedConfiguration
		force bool
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		want         *pb.AppliedConfiguration
		wantReplaced *pb.AppliedConfiguration
	}{
		{
			name: "invalid ID",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id: appliedConfID2,
				},
			},
			wantErr: true,
		},
		{
			name: "missing deviceID",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:    appliedConfID2,
					Owner: owner,
				},
			},
			wantErr: true,
		},
		{
			name: "missing configuration",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:       appliedConfID2,
					Owner:    owner,
					DeviceId: deviceID,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid configurationID",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID2,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{},
				},
			},
			wantErr: true,
		},
		{
			name: "missing executedBy",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID2,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID2},
				},
			},
			wantErr: true,
		},
		{
			name: "missing resources",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID2,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID2},
					ExecutedBy:      executedBy2,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid resource - missing href",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID2,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID2},
					ExecutedBy:      executedBy2,
					Resources: []*pb.AppliedConfiguration_Resource{
						{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid resource - missing correlationID",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID2,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID2},
					ExecutedBy:      executedBy2,
					Resources: []*pb.AppliedConfiguration_Resource{
						{
							Href: "/href",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate ID",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID1,
					Owner:           owner,
					DeviceId:        "deviceID2",
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: uuid.NewString()},
					ExecutedBy:      executedBy2,
					Resources:       []*pb.AppliedConfiguration_Resource{resource1},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate deviceID + configurationID",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              uuid.NewString(),
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID1},
					ExecutedBy:      executedBy1,
					Resources:       []*pb.AppliedConfiguration_Resource{resource1},
				},
			},
			wantErr: true,
		},
		{
			name: "new",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID2,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID2},
					ExecutedBy:      executedBy2,
					Resources:       []*pb.AppliedConfiguration_Resource{resource2},
				},
			},
			want: &pb.AppliedConfiguration{
				Id:              appliedConfID2,
				Owner:           owner,
				DeviceId:        deviceID,
				ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID2},
				ExecutedBy:      executedBy2,
				Resources:       []*pb.AppliedConfiguration_Resource{resource2},
			},
		},
		{
			name: "force duplicate deviceID + configurationID",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID3,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID1},
					ExecutedBy:      executedBy2,
					Resources:       []*pb.AppliedConfiguration_Resource{resource2},
				},
				force: true,
			},
			want: &pb.AppliedConfiguration{
				Id:              appliedConfID3,
				Owner:           owner,
				DeviceId:        deviceID,
				ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID1},
				ExecutedBy:      executedBy2,
				Resources:       []*pb.AppliedConfiguration_Resource{resource2},
			},
			wantReplaced: appliedConf1,
		},
		{
			name: "new (force)",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              appliedConfID4,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID3},
					ExecutedBy:      executedBy2,
					Resources:       []*pb.AppliedConfiguration_Resource{resource2},
				},
				force: true,
			},
			want: &pb.AppliedConfiguration{
				Id:              appliedConfID4,
				Owner:           owner,
				DeviceId:        deviceID,
				ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID3},
				ExecutedBy:      executedBy2,
				Resources:       []*pb.AppliedConfiguration_Resource{resource2},
			},
		},
		{
			// force allows to create a new applied configuration with the same deviceID and configurationID
			// however, the owner must match
			name: "fail force duplicate deviceID + configurationID - mismatched owner",
			args: args{
				adc: &pb.AppliedConfiguration{
					Id:              uuid.NewString(),
					Owner:           "mismatched",
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedConfiguration_RelationTo{Id: confID1},
					ExecutedBy:      executedBy2,
					Resources:       []*pb.AppliedConfiguration_Resource{resource2},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotReplaced, err := s.CreateAppliedConfiguration(ctx, tt.args.adc, tt.args.force)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantReplaced != nil {
				require.NotNil(t, gotReplaced)
				test.CmpAppliedDeviceConfiguration(t, tt.wantReplaced, gotReplaced, true)
			} else {
				require.Nil(t, gotReplaced)
			}
			test.CmpAppliedDeviceConfiguration(t, tt.want, got, true)
		})
	}
}
