package grpc_test

import (
	"context"
	"testing"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestCertificateAuthorityServerDeleteSigningRecords(t *testing.T) {
	owner := events.OwnerToUUID("owner")
	const ownerClaim = "sub"
	r := &store.SigningRecord{
		Id:           "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
		Owner:        owner,
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		CreationDate: constDate().UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate1",
			Date:           constDate().UnixNano(),
			ValidUntilDate: constDate().UnixNano(),
		},
	}
	type args struct {
		req *pb.DeleteSigningRecordsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "invalidID",
			args: args{
				req: &pb.DeleteSigningRecordsRequest{
					IdFilter: []string{"invalidID"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.DeleteSigningRecordsRequest{
					IdFilter: []string{r.Id},
				},
			},
			want: 1,
		},
	}

	store, closeStore := test.NewMongoStore(t)
	defer closeStore()

	err := store.CreateSigningRecord(context.Background(), r)
	require.NoError(t, err)

	logger := log.NewLogger(log.MakeDefaultConfig())

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()

	ch := new(inprocgrpc.Channel)
	ca, err := grpc.NewCertificateAuthorityServer(ownerClaim, config.HubID(), test.MakeConfig(t).Signer, store, fileWatcher, logger)
	require.NoError(t, err)
	defer ca.Close()

	pb.RegisterCertificateAuthorityServer(ch, ca)
	grpcClient := pb.NewCertificateAuthorityClient(ch)
	token := config.CreateJwtToken(t, jwt.MapClaims{
		ownerClaim: owner,
	})
	ctx := kitNetGrpc.CtxWithToken(context.Background(), token)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grpcClient.DeleteSigningRecords(ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, got.GetCount())
		})
	}
}
