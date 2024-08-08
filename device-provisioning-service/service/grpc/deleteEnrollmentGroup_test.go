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

func TestDeviceProvisionServiceServerDeleteEnrollmentGroups(t *testing.T) {
	store, closeStore := test.NewMongoStore(t)
	defer closeStore()
	eg := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)
	err := store.CreateEnrollmentGroup(context.Background(), test.DPSOwner, eg)
	require.NoError(t, err)

	type args struct {
		req *pb.DeleteEnrollmentGroupsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				req: &pb.DeleteEnrollmentGroupsRequest{
					IdFilter: []string{eg.GetId()},
				},
			},
		},
	}

	ch := new(inprocgrpc.Channel)
	pb.RegisterDeviceProvisionServiceServer(ch, grpc.NewDeviceProvisionServiceServer(store, test.MakeAuthorizationConfig().OwnerClaim))
	grpcClient := pb.NewDeviceProvisionServiceClient(ch)

	ctx := pkgGrpc.CtxWithToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": test.DPSOwner,
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grpcClient.DeleteEnrollmentGroups(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}
