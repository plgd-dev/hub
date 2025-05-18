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

func TestDeviceProvisionServiceServerGetEnrollmentGroups(t *testing.T) {
	eg := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)
	type args struct {
		req *pb.GetEnrollmentGroupsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    pb.EnrollmentGroups
		wantErr bool
	}{
		{
			name: "invalidID",
			args: args{
				req: &pb.GetEnrollmentGroupsRequest{
					IdFilter: []string{"invalidID"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.GetEnrollmentGroupsRequest{
					IdFilter: []string{eg.GetId()},
				},
			},
			want: []*pb.EnrollmentGroup{eg},
		},
	}

	store, closeStore := test.NewMongoStore(t)
	defer closeStore()

	err := store.CreateEnrollmentGroup(context.Background(), eg.GetOwner(), eg)
	require.NoError(t, err)

	ch := new(inprocgrpc.Channel)
	pb.RegisterDeviceProvisionServiceServer(ch, grpc.NewDeviceProvisionServiceServer(store, test.MakeAuthorizationConfig().OwnerClaim))
	grpcClient := pb.NewDeviceProvisionServiceClient(ch)

	ctx := pkgGrpc.CtxWithToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": test.DPSOwner,
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := grpcClient.GetEnrollmentGroups(ctx, tt.args.req)
			require.NoError(t, err)
			var got pb.EnrollmentGroups
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
			require.Len(t, got, len(tt.want))
			tt.want.Sort()
			got.Sort()
			for i := range got {
				hubTest.CheckProtobufs(t, tt.want[i], got[i], hubTest.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
