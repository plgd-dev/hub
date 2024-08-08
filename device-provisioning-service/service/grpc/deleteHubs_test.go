package grpc_test

import (
	"context"
	"testing"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service/grpc"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestDeviceProvisionServiceServerDeleteHubs(t *testing.T) {
	h := test.NewHub(uuid.NewString(), test.DPSOwner)
	ch := new(inprocgrpc.Channel)
	store, closeStore := test.NewMongoStore(t)
	defer closeStore()
	pb.RegisterDeviceProvisionServiceServer(ch, grpc.NewDeviceProvisionServiceServer(store, test.MakeAuthorizationConfig().OwnerClaim))
	grpcClient := pb.NewDeviceProvisionServiceClient(ch)

	err := store.CreateHub(context.Background(), h.GetOwner(), h)
	require.NoError(t, err)
	type args struct {
		req *pb.DeleteHubsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				req: &pb.DeleteHubsRequest{
					IdFilter: []string{h.GetId()},
				},
			},
		},
	}

	ctx := pkgGrpc.CtxWithToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": test.DPSOwner,
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grpcClient.DeleteHubs(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}
