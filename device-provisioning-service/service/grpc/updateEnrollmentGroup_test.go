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
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestDeviceProvisionServiceServerUpdateEnrollmentGroup(t *testing.T) {
	store, closeStore := test.NewMongoStore(t)
	defer closeStore()
	eg := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)
	err := store.CreateEnrollmentGroup(context.Background(), eg.GetOwner(), eg)
	require.NoError(t, err)

	egUpd := eg
	egUpd.PreSharedKey = ""
	egUpd.Name = "newName"
	type args struct {
		req *pb.UpdateEnrollmentGroupRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *pb.EnrollmentGroup
	}{
		{
			name: "valid",
			args: args{
				req: &pb.UpdateEnrollmentGroupRequest{
					Id: eg.GetId(),
					EnrollmentGroup: &pb.UpdateEnrollmentGroup{
						AttestationMechanism: egUpd.GetAttestationMechanism(),
						HubIds:               eg.GetHubIds(),
						PreSharedKey:         egUpd.GetPreSharedKey(),
						Name:                 egUpd.GetName(),
					},
				},
			},
			want: egUpd,
		},
		{
			name: "invalid",
			args: args{
				req: &pb.UpdateEnrollmentGroupRequest{
					Id: "invalid",
				},
			},
			wantErr: true,
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
			got, err := grpcClient.UpdateEnrollmentGroup(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			hubTest.CheckProtobufs(t, tt.want, got, hubTest.RequireToCheckFunc(require.Equal))
		})
	}
}
