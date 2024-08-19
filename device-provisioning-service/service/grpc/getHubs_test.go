package grpc_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service/grpc"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestDeviceProvisionServiceServerGetHubs(t *testing.T) {
	h := test.NewHub(uuid.NewString(), test.DPSOwner)
	type args struct {
		req *pb.GetHubsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    pb.Hubs
		wantErr bool
	}{
		{
			name: "invalidID",
			args: args{
				req: &pb.GetHubsRequest{
					IdFilter: []string{"invalidID"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.GetHubsRequest{
					IdFilter: []string{h.GetId()},
				},
			},
			want: []*pb.Hub{h},
		},
	}

	store, closeStore := test.NewMongoStore(t)
	defer closeStore()

	err := store.CreateHub(context.Background(), h.GetOwner(), h)
	require.NoError(t, err)

	ch := new(inprocgrpc.Channel)
	pb.RegisterDeviceProvisionServiceServer(ch, grpc.NewDeviceProvisionServiceServer(store, test.MakeAuthorizationConfig().OwnerClaim))
	grpcClient := pb.NewDeviceProvisionServiceClient(ch)

	ctx := pkgGrpc.CtxWithToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": test.DPSOwner,
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := grpcClient.GetHubs(ctx, tt.args.req)
			require.NoError(t, err)
			var got pb.Hubs
			for {
				r, err := client.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				got = append(got, r)
			}
			require.Equal(t, len(tt.want), len(got))
			tt.want.Sort()
			got.Sort()
			for i := range got {
				hubTest.CheckProtobufs(t, tt.want[i], got[i], hubTest.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
