package mongodb_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func makeLightResourceConfiguration(t *testing.T, id string, power int, ttl int64) *pb.Configuration_Resource {
	return &pb.Configuration_Resource{
		Href: hubTest.TestResourceLightInstanceHref(id),
		Content: &commands.Content{
			Data: hubTest.EncodeToCbor(t, map[string]interface{}{
				"power": power,
			}),
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: int32(message.AppOcfCbor),
		},
		TimeToLive: ttl,
	}
}

func TestStoreCreateConfiguration(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	confID := uuid.NewString()
	const owner = "owner1"
	resources := []*pb.Configuration_Resource{
		makeLightResourceConfiguration(t, "1", 1, 1337),
	}

	type args struct {
		create *pb.Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(*testing.T, *pb.Configuration)
	}{
		{
			name: "invalid ID",
			args: args{
				create: &pb.Configuration{
					Id:        "invalid",
					Name:      "invalid ID",
					Owner:     owner,
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			args: args{
				create: &pb.Configuration{
					Id:        confID,
					Name:      "missing owner",
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "missing resources",
			args: args{
				create: &pb.Configuration{
					Id:      confID,
					Name:    "missing resources",
					Owner:   owner,
					Version: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				create: &pb.Configuration{
					Id:        confID,
					Name:      "valid",
					Owner:     owner,
					Resources: resources,
				},
			},
			want: func(t *testing.T, got *pb.Configuration) {
				wantCfg := &pb.Configuration{
					Id:        confID,
					Name:      "valid",
					Owner:     owner,
					Resources: resources,
				}
				test.CmpConfiguration(t, wantCfg, got, true)
			},
		},
		{
			name: "valid - generated ID",
			args: args{
				create: &pb.Configuration{
					Name:      "valid",
					Owner:     owner,
					Version:   37,
					Resources: resources,
				},
			},
			want: func(t *testing.T, got *pb.Configuration) {
				wantCfg := &pb.Configuration{
					Id:        got.GetId(),
					Name:      "valid",
					Owner:     owner,
					Version:   37,
					Resources: resources,
				}
				test.CmpConfiguration(t, wantCfg, got, true)
			},
		},
		{
			name: "duplicit item (ID)",
			args: args{
				create: &pb.Configuration{
					Id:        confID,
					Name:      "duplicit ID",
					Owner:     owner,
					Resources: resources,
					Version:   42,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.CreateConfiguration(ctx, tt.args.create)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, got)
		})
	}
}
