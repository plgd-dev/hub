package grpc_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestCertificateAuthorityServerDeleteSigningRecords(t *testing.T) {
	owner := events.OwnerToUUID("owner")
	const ownerClaim = "sub"
	token := config.CreateJwtToken(t, jwt.MapClaims{
		ownerClaim: owner,
	})
	ctx := pkgGrpc.CtxWithToken(context.Background(), token)
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
			Serial:         big.NewInt(42).String(),
			IssuerId:       "42424242-4242-4242-4242-424242424242",
		},
	}
	type args struct {
		req *pb.DeleteSigningRecordsRequest
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "missing token with ownerClaim in ctx",
			args: args{
				req: &pb.DeleteSigningRecordsRequest{
					IdFilter: []string{r.GetId()},
				},
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "invalidID",
			args: args{
				req: &pb.DeleteSigningRecordsRequest{
					IdFilter: []string{"invalidID"},
				},
				ctx: ctx,
			},
		},
		{
			name: "valid",
			args: args{
				req: &pb.DeleteSigningRecordsRequest{
					IdFilter: []string{r.GetId()},
				},
				ctx: ctx,
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
	ca, err := grpc.NewCertificateAuthorityServer(ownerClaim, config.HubID(), "https://"+config.CERTIFICATE_AUTHORITY_HTTP_HOST, test.MakeConfig(t).Signer, store, fileWatcher, logger)
	require.NoError(t, err)
	defer ca.Close()

	pb.RegisterCertificateAuthorityServer(ch, ca)
	grpcClient := pb.NewCertificateAuthorityClient(ch)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grpcClient.DeleteSigningRecords(tt.args.ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.GetCount())
		})
	}
}
