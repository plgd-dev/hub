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

	appliedConfID := uuid.NewString()
	owner := "owner"
	deviceID := "deviceID"
	confID := uuid.NewString()
	condID := uuid.NewString()
	executedBy := &pb.AppliedDeviceConfiguration_ConditionId{
		ConditionId: &pb.AppliedDeviceConfiguration_RelationTo{
			Id: condID,
		},
	}
	resource := &pb.AppliedDeviceConfiguration_Resource{
		Href:          "href",
		CorrelationId: "correlationID",
		Status:        pb.AppliedDeviceConfiguration_Resource_DONE,
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

	type args struct {
		adc *pb.AppliedDeviceConfiguration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *pb.AppliedDeviceConfiguration
	}{
		{
			name: "invalid ID",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id: appliedConfID,
				},
			},
			wantErr: true,
		},
		{
			name: "missing deviceID",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id:    appliedConfID,
					Owner: owner,
				},
			},
			wantErr: true,
		},
		{
			name: "missing configuration",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id:       appliedConfID,
					Owner:    owner,
					DeviceId: deviceID,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid configurationID",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id:              appliedConfID,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{},
				},
			},
			wantErr: true,
		},
		{
			name: "missing executedBy",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id:              appliedConfID,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{Id: confID},
				},
			},
			wantErr: true,
		},
		{
			name: "missing resources",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id:              appliedConfID,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{Id: confID},
					ExecutedBy:      executedBy,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid resource - missing href",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id:              appliedConfID,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{Id: confID},
					ExecutedBy:      executedBy,
					Resources: []*pb.AppliedDeviceConfiguration_Resource{
						{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid resource - missing correlationID",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id:              appliedConfID,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{Id: confID},
					ExecutedBy:      executedBy,
					Resources: []*pb.AppliedDeviceConfiguration_Resource{
						{
							Href: "href",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				adc: &pb.AppliedDeviceConfiguration{
					Id:              appliedConfID,
					Owner:           owner,
					DeviceId:        deviceID,
					ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{Id: confID},
					ExecutedBy:      executedBy,
					Resources:       []*pb.AppliedDeviceConfiguration_Resource{resource},
				},
			},
			want: &pb.AppliedDeviceConfiguration{
				Id:              appliedConfID,
				Owner:           owner,
				DeviceId:        deviceID,
				ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{Id: confID},
				ExecutedBy:      executedBy,
				Resources:       []*pb.AppliedDeviceConfiguration_Resource{resource},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.CreateAppliedDeviceConfiguration(ctx, tt.args.adc)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			test.CmpAppliedDeviceConfiguration(t, tt.want, got, true)
		})
	}
}
