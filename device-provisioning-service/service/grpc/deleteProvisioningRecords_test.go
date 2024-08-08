package grpc_test

import (
	"context"
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

const (
	provisionRecordID = "mfgID"
)

func TestDeviceProvisionServiceServerDeleteProvisioningRecords(t *testing.T) {
	type args struct {
		req *pb.DeleteProvisioningRecordsRequest
	}
	tests := []struct {
		name string
		args args
		want *pb.DeleteProvisioningRecordsResponse
	}{
		{
			name: "invalidID",
			args: args{
				req: &pb.DeleteProvisioningRecordsRequest{
					IdFilter: []string{"invalidID"},
				},
			},
			want: &pb.DeleteProvisioningRecordsResponse{
				Count: 0,
			},
		},
		{
			name: "valid",
			args: args{
				req: &pb.DeleteProvisioningRecordsRequest{
					IdFilter: []string{provisionRecordID},
				},
			},
			want: &pb.DeleteProvisioningRecordsResponse{
				Count: 1,
			},
		},
	}

	store, closeStore := test.NewMongoStore(t)
	defer closeStore()

	err := store.UpdateProvisioningRecord(context.Background(), test.DPSOwner, &pb.ProvisioningRecord{
		Id:    provisionRecordID,
		Owner: test.DPSOwner,
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
			resp, err := grpcClient.DeleteProvisioningRecords(ctx, tt.args.req)
			require.NoError(t, err)
			hubTest.CheckProtobufs(t, tt.want, resp, hubTest.RequireToCheckFunc(require.Equal))
		})
	}
}
