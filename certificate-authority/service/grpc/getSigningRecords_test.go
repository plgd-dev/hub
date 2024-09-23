package grpc_test

import (
	"context"
	"errors"
	"io"
	"math/big"
	"testing"
	"time"

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
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func constDate() time.Time {
	return time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
}

func TestCertificateAuthorityServerGetSigningRecords(t *testing.T) {
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
			Serial:         big.NewInt(42).String(),
			IssuerId:       "42424242-4242-4242-4242-424242424242",
		},
	}
	type args struct {
		req *pb.GetSigningRecordsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    pb.SigningRecords
		wantErr bool
	}{
		{
			name: "invalidID",
			args: args{
				req: &pb.GetSigningRecordsRequest{
					IdFilter: []string{"invalidID"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.GetSigningRecordsRequest{
					IdFilter: []string{r.GetId()},
				},
			},
			want: []*pb.SigningRecord{r},
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
	token := config.CreateJwtToken(t, jwt.MapClaims{
		ownerClaim: owner,
	})
	ctx := pkgGrpc.CtxWithToken(context.Background(), token)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := grpcClient.GetSigningRecords(ctx, tt.args.req)
			require.NoError(t, err)
			var got pb.SigningRecords
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
			require.Equal(t, len(tt.want), len(got))
			tt.want.Sort()
			got.Sort()
			for i := range got {
				hubTest.CheckProtobufs(t, tt.want[i], got[i], hubTest.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
