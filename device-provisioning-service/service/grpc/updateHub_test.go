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

func TestDeviceProvisionServiceServerUpdateHub(t *testing.T) {
	store, closeStore := test.NewMongoStore(t)
	defer closeStore()
	h := test.NewHub(uuid.NewString(), test.DPSOwner)
	err := store.CreateHub(context.Background(), h.GetOwner(), h)
	require.NoError(t, err)
	hUpd := h
	hUpd.Gateways = []string{"coaps://123"}
	hUpd.Name = "new name"

	type args struct {
		req *pb.UpdateHubRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *pb.Hub
	}{
		{
			name: "valid",
			args: args{
				req: &pb.UpdateHubRequest{
					Id: hUpd.GetId(),
					Hub: &pb.UpdateHub{
						Gateways:             hUpd.GetGateways(),
						CertificateAuthority: hUpd.GetCertificateAuthority(),
						Authorization:        hUpd.GetAuthorization(),
						Name:                 hUpd.GetName(),
					},
				},
			},
			want: hUpd,
		},
		{
			name: "invalid",
			args: args{
				req: &pb.UpdateHubRequest{
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
			got, err := grpcClient.UpdateHub(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			hubTest.CheckProtobufs(t, tt.want, got, hubTest.RequireToCheckFunc(require.Equal))
		})
	}
}
