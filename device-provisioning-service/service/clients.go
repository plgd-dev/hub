package service

import (
	"context"
	"fmt"

	pbCA "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func newGrpcClient(config client.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, service string) (*grpc.ClientConn, func(), error) {
	idConn, err := client.New(config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create connection to %v: %w", service, err)
	}
	closeIDConn := func() {
		if err := idConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to %v: %w", service, err)
		}
	}
	return idConn.GRPC(), closeIDConn, nil
}

func newCertificateAuthorityClient(config client.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbCA.CertificateAuthorityClient, func(), error) {
	client, closeClient, err := newGrpcClient(config, fileWatcher, logger, tracerProvider, "certificate-authority")
	if err != nil {
		return nil, nil, err
	}
	return pbCA.NewCertificateAuthorityClient(client), closeClient, err
}

func NewStore(ctx context.Context, config mongodb.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*mongodb.Store, func(), error) {
	var fl fn.FuncList
	certManager, err := cmClient.New(config.Mongo.TLS, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	fl.AddFunc(certManager.Close)

	db, err := mongodb.NewStore(ctx, config, certManager.GetTLSConfig(), logger, tracerProvider)
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create mongodb store: %w", err)
	}
	fl.AddFunc(func() {
		if err := db.Close(ctx); err != nil {
			log.Errorf("failed to close mongodb store: %w", err)
		}
	})

	return db, fl.ToFunction(), nil
}
