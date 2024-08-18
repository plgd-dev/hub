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

func TestDeviceProvisionServiceServerCreateHub(t *testing.T) {
	h := test.NewHub(uuid.NewString(), test.DPSOwner)
	type args struct {
		req *pb.CreateHubRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.Hub
		wantErr bool
	}{
		{
			name: "invalidID",
			args: args{
				req: &pb.CreateHubRequest{},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.CreateHubRequest{
					HubId:                h.GetHubId(),
					Gateways:             h.GetGateways(),
					CertificateAuthority: h.GetCertificateAuthority(),
					Authorization:        h.GetAuthorization(),
					Name:                 h.GetName(),
				},
			},
			want: h,
		},
	}
	ch := new(inprocgrpc.Channel)

	store, closeStore := test.NewMongoStore(t)
	defer closeStore()
	pb.RegisterDeviceProvisionServiceServer(ch, grpc.NewDeviceProvisionServiceServer(store, test.MakeAuthorizationConfig().OwnerClaim))
	grpcClient := pb.NewDeviceProvisionServiceClient(ch)

	ctx := pkgGrpc.CtxWithToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": test.DPSOwner,
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grpcClient.CreateHub(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got.GetId())
			require.NotEqual(t, tt.want.GetId(), got.GetId())
			got.Id = tt.want.GetId()
			hubTest.CheckProtobufs(t, tt.want, got, hubTest.RequireToCheckFunc(require.Equal))
		})
	}
}
