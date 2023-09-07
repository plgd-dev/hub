package service_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

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
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func constDate() time.Time {
	return time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
}

func TestCertificateAuthorityServerCleanUpSigningRecords(t *testing.T) {
	owner := events.OwnerToUUID("owner")
	const ownerClaim = "sub"
	r := &store.SigningRecord{
		Id:           "id",
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

	cfg := test.MakeConfig(t)
	cfg.Clients.Storage.ExtendCronParserBySeconds = true
	cfg.Clients.Storage.CleanUpRecords = "*/2 * * * * *"

	shutDown := testService.SetUpServices(context.Background(), t, testService.SetUpServicesCertificateAuthority|testService.SetUpServicesOAuth, testService.WithCAConfig(cfg))
	defer shutDown()

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
	client, err := grpcClient.GetSigningRecords(ctx, &pb.GetSigningRecordsRequest{})
	require.NoError(t, err)
	var got pb.SigningRecords
	for {
		r, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		got = append(got, r)
	}
	require.Equal(t, 1, len(got))
	time.Sleep(4 * time.Second)
	client, err = grpcClient.GetSigningRecords(ctx, &pb.GetSigningRecordsRequest{})
	require.NoError(t, err)
	got = nil
	for {
		r, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		got = append(got, r)
	}
	require.Equal(t, 0, len(got))
}
