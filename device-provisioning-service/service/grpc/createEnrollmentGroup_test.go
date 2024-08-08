package grpc_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service/grpc"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestDeviceProvisionServiceServerCreateEnrollmentGroup(t *testing.T) {
	eg := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)

	certificateChain, err := urischeme.URIScheme(eg.GetAttestationMechanism().GetX509().GetCertificateChain()).Read()
	require.NoError(t, err)
	uriSchemeDataCertificateChain := "data:;base64," + base64.StdEncoding.EncodeToString(certificateChain)

	preSharedKey, err := urischeme.URIScheme(eg.GetPreSharedKey()).Read()
	require.NoError(t, err)
	uriSchemeDataPreSharedKey := "data:;base64," + base64.StdEncoding.EncodeToString(preSharedKey)

	type args struct {
		req *pb.CreateEnrollmentGroupRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.EnrollmentGroup
		wantErr bool
	}{
		{
			name: "valid-certificateChain-file",
			args: args{
				req: &pb.CreateEnrollmentGroupRequest{
					HubIds:               eg.GetHubIds(),
					AttestationMechanism: eg.GetAttestationMechanism(),
					PreSharedKey:         eg.GetPreSharedKey(),
					Name:                 eg.GetName(),
				},
			},
			want: &pb.EnrollmentGroup{
				HubIds:               eg.GetHubIds(),
				Owner:                eg.GetOwner(),
				AttestationMechanism: eg.GetAttestationMechanism(),
				PreSharedKey:         eg.GetPreSharedKey(),
				Name:                 eg.GetName(),
			},
		},
		{
			name: "valid-certificateChain-data",
			args: args{
				req: &pb.CreateEnrollmentGroupRequest{
					HubIds: eg.GetHubIds(),
					AttestationMechanism: &pb.AttestationMechanism{
						X509: &pb.X509Configuration{
							CertificateChain: uriSchemeDataCertificateChain,
						},
					},
					PreSharedKey: uriSchemeDataPreSharedKey,
					Name:         eg.GetName(),
				},
			},
			want: &pb.EnrollmentGroup{
				HubIds: eg.GetHubIds(),
				AttestationMechanism: &pb.AttestationMechanism{
					X509: &pb.X509Configuration{
						CertificateChain:    uriSchemeDataCertificateChain,
						LeadCertificateName: eg.GetAttestationMechanism().GetX509().GetLeadCertificateName(),
					},
				},
				PreSharedKey: uriSchemeDataPreSharedKey,
				Name:         eg.GetName(),
				Owner:        eg.GetOwner(),
			},
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
			got, err := grpcClient.CreateEnrollmentGroup(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got.GetId())
			got.Id = ""
			hubTest.CheckProtobufs(t, tt.want, got, hubTest.RequireToCheckFunc(require.Equal))
		})
	}
}
