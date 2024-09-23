package service_test

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/plgd-dev/hub/v2/test/config"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestCertificateAuthorityServerCleanUpSigningRecords(t *testing.T) {
	cfg := test.MakeConfig(t)
	cfg.Clients.Storage.ExtendCronParserBySeconds = true
	cfg.Clients.Storage.CleanUpRecords = "*/1 * * * * *"
	fmt.Printf("%v\n\n", test.MakeConfig(t))

	shutDown := testService.SetUpServices(context.Background(), t, testService.SetUpServicesCertificateAuthority|testService.SetUpServicesOAuth|testService.SetUpServicesMachine2MachineOAuth, testService.WithCAConfig(cfg))
	defer shutDown()

	storeDB, closeStore := test.NewStore(t)
	defer closeStore()

	date := time.Now().Add(time.Second * 2)
	owner := events.OwnerToUUID("owner")
	const ownerClaim = "sub"
	r := &store.SigningRecord{
		Id:           "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
		Owner:        owner,
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		CreationDate: date.UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate1",
			Date:           date.UnixNano(),
			ValidUntilDate: date.UnixNano(),
			Serial:         big.NewInt(42).String(),
			IssuerId:       "42424242-4242-4242-4242-424242424242",
		},
	}

	err := storeDB.CreateSigningRecord(context.Background(), r)
	require.NoError(t, err)

	logger := log.NewLogger(log.MakeDefaultConfig())

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()

	ch := new(inprocgrpc.Channel)
	ca, err := grpc.NewCertificateAuthorityServer(ownerClaim, config.HubID(), "https://"+config.CERTIFICATE_AUTHORITY_HTTP_HOST, test.MakeConfig(t).Signer, storeDB, fileWatcher, logger)
	require.NoError(t, err)
	defer ca.Close()

	pb.RegisterCertificateAuthorityServer(ch, ca)
	grpcClient := pb.NewCertificateAuthorityClient(ch)
	token := config.CreateJwtToken(t, jwt.MapClaims{
		ownerClaim: owner,
	})
	ctx := pkgGrpc.CtxWithToken(context.Background(), token)
	client, err := grpcClient.GetSigningRecords(ctx, &pb.GetSigningRecordsRequest{})
	require.NoError(t, err)
	var got pb.SigningRecords
	for {
		r, errR := client.Recv()
		if errors.Is(errR, io.EOF) {
			break
		}
		require.NoError(t, errR)
		got = append(got, r)
	}
	require.Len(t, got, 1)
	time.Sleep(4 * time.Second)
	client, err = grpcClient.GetSigningRecords(ctx, &pb.GetSigningRecordsRequest{})
	require.NoError(t, err)
	got = nil
	for {
		r, errR := client.Recv()
		if errors.Is(errR, io.EOF) {
			break
		}
		require.NoError(t, errR)
		got = append(got, r)
	}
	require.Empty(t, got)
}
