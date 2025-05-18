package grpc_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service/grpc"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestDeviceProvisionServiceServerGetProvisioningRecords(t *testing.T) {
	type args struct {
		req *pb.GetProvisioningRecordsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    pb.ProvisioningRecords
		wantErr bool
	}{
		{
			name: "invalidID",
			args: args{
				req: &pb.GetProvisioningRecordsRequest{
					IdFilter: []string{"invalidID"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.GetProvisioningRecordsRequest{
					IdFilter: []string{provisionRecordID},
				},
			},
			want: []*pb.ProvisioningRecord{{Id: provisionRecordID, Owner: test.DPSOwner}},
		},
		{
			name: "all",
			args: args{
				req: &pb.GetProvisioningRecordsRequest{},
			},
			want: []*pb.ProvisioningRecord{{Id: provisionRecordID, Owner: test.DPSOwner}},
		},
	}

	store, closeStore := test.NewMongoStore(t)
	defer closeStore()

	err := store.UpdateProvisioningRecord(context.Background(), test.DPSOwner, &pb.ProvisioningRecord{
		Id:    provisionRecordID,
		Owner: test.DPSOwner,
	})
	require.NoError(t, err)
	err = store.UpdateProvisioningRecord(context.Background(), "anotherOwner", &pb.ProvisioningRecord{
		Id:    "anotherID",
		Owner: "anotherOwner",
	})
	require.NoError(t, err)
	err = store.FlushBulkWriter()
	require.NoError(t, err)

	ch := new(inprocgrpc.Channel)
	pb.RegisterDeviceProvisionServiceServer(ch, grpc.NewDeviceProvisionServiceServer(store, test.MakeAuthorizationConfig().OwnerClaim))
	grpcClient := pb.NewDeviceProvisionServiceClient(ch)

	ctx := pkgGrpc.CtxWithToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": test.DPSOwner,
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := grpcClient.GetProvisioningRecords(ctx, tt.args.req)
			require.NoError(t, err)
			var got pb.ProvisioningRecords
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
				require.Positive(t, r.GetCreationDate())
				r.CreationDate = 0
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
