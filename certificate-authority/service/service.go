package service

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	httpService "github.com/plgd-dev/hub/v2/certificate-authority/service/http"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/uri"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	storeConfig "github.com/plgd-dev/hub/v2/certificate-authority/store/config"
	"github.com/plgd-dev/hub/v2/certificate-authority/store/cqldb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "certificate-authority"

func createStore(ctx context.Context, config storeConfig.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (store.Store, error) {
	switch config.Use {
	case database.MongoDB:
		s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("mongodb: %w", err)
		}
		return s, nil
	case database.CqlDB:
		s, err := cqldb.New(ctx, config.CqlDB, fileWatcher, logger, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("cqldb: %w", err)
		}
		return s, nil
	}
	return nil, fmt.Errorf("invalid store use('%v')", config.Use)
}

func newStore(ctx context.Context, config StorageConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (store.Store, func(), error) {
	var fl fn.FuncList
	db, err := createStore(ctx, config.Embedded, fileWatcher, logger, tracerProvider)
	if err != nil {
		fl.Execute()
		return nil, nil, err
	}
	fl.AddFunc(func() {
		if errC := db.Close(ctx); errC != nil {
			log.Errorf("failed to close mongodb store: %w", errC)
		}
	})
	if config.CleanUpRecords == "" {
		return db, fl.ToFunction(), nil
	}
	s, err := gocron.NewScheduler(gocron.WithLocation(time.Local)) //nolint:gosmopolitan
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create cron job: %w", err)
	}

	_, err = s.NewJob(gocron.CronJob(config.CleanUpRecords, config.ExtendCronParserBySeconds), gocron.NewTask(func() {
		_, errDel := db.DeleteNonDeviceExpiredRecords(ctx, time.Now())
		if errDel != nil && !errors.Is(errDel, store.ErrNotSupported) {
			log.Errorf("failed to delete expired signing records: %w", errDel)
		}
	}))
	if err != nil {
		fl.Execute()
		return nil, nil, fmt.Errorf("cannot create cron job: %w", err)
	}
	fl.AddFunc(func() {
		if errS := s.Shutdown(); errS != nil {
			log.Errorf("failed to shutdown cron job: %w", errS)
		}
	})
	s.Start()
	return db, fl.ToFunction(), nil
}

func errVerifyDistributionPoint(err error) error {
	return fmt.Errorf("failed to verify distribution point: %w", err)
}

func getIssuerFromEndpoint(addr string) (string, error) {
	prefix := uri.SigningRevocationListBase + "/"
	var issuerID string
	if strings.HasPrefix(addr, prefix) {
		issuerID = strings.TrimPrefix(addr, prefix)
		// ignore everything after next "/"
		if len(issuerID) > 36 && issuerID[36] == '/' {
			issuerID = issuerID[:36]
		}
	}
	// uuid string is 36 chars long
	if len(issuerID) != 36 {
		return "", errVerifyDistributionPoint(fmt.Errorf("invalid issuerID(%s)", issuerID))
	}
	return issuerID, nil
}

func getVerifyCertificateByCRLFromStorage(s store.Store, logger log.Logger) func(ctx context.Context, certificate *x509.Certificate, endpoint string) error {
	return func(ctx context.Context, certificate *x509.Certificate, endpoint string) error {
		// get issuer from endpoint /certificate-authority/api/v1/signing/crl/{$issuerID}
		issuerID, err := getIssuerFromEndpoint(endpoint)
		if err != nil {
			return err
		}

		rl, err := s.GetRevocationList(ctx, issuerID, false)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil
			}
			return errVerifyDistributionPoint(err)
		}

		for _, revoked := range rl.Certificates {
			serial := &big.Int{}
			_, ok := serial.SetString(revoked.Serial, 10)
			if !ok {
				logger.Debugf("invalid serial number: %s", revoked.Serial)
				continue
			}
			if certificate.SerialNumber.Cmp(serial) == 0 {
				return pkgX509.ErrRevoked
			}
		}
		return nil
	}
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "certificate-authority", fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	var closerFn fn.FuncList
	closerFn.AddFunc(otelClient.Close)
	tracerProvider := otelClient.GetTracerProvider()

	dbStorage, closeStore, err := newStore(ctx, config.Clients.Storage, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create store: %w", err)
	}
	closerFn.AddFunc(closeStore)

	externalAddress := pkgHttpUri.CanonicalURI(config.APIs.HTTP.ExternalAddress)
	crlEnabled := externalAddress != "" && dbStorage.SupportsRevocationList() && config.Signer.CRL.Enabled
	config.Signer.CRL.Enabled = crlEnabled
	ca, err := grpcService.NewCertificateAuthorityServer(config.APIs.GRPC.Authorization.OwnerClaim, config.HubID, externalAddress, config.Signer, dbStorage, fileWatcher, logger)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create grpc certificate authority server: %w", err)
	}
	closerFn.AddFunc(ca.Close)

	var customDistributionPointCRLVerification pkgX509.CustomDistributionPointVerification
	if crlEnabled {
		customDistributionPointCRLVerification = pkgX509.CustomDistributionPointVerification{
			externalAddress: getVerifyCertificateByCRLFromStorage(dbStorage, logger),
		}
	}
	httpValidator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, fileWatcher, logger, tracerProvider, validator.WithCustomDistributionPointVerification(customDistributionPointCRLVerification))
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create http validator: %w", err)
	}
	closerFn.AddFunc(httpValidator.Close)
	httpService, err := httpService.New(serviceName, httpService.Config{
		Connection: listener.Config{
			Addr: config.APIs.HTTP.Addr,
			TLS:  config.APIs.GRPC.TLS,
		},
		Authorization: config.APIs.GRPC.Authorization.Config,
		Server:        config.APIs.HTTP.Server,
		CRLEnabled:    crlEnabled,
	}, dbStorage, ca, httpValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create http service: %w", err)
	}
	grpcValidator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, fileWatcher, logger, tracerProvider, validator.WithCustomDistributionPointVerification(customDistributionPointCRLVerification))
	if err != nil {
		closerFn.Execute()
		_ = httpService.Close()
		return nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}
	grpcService, err := grpcService.New(config.APIs.GRPC, ca, grpcValidator, fileWatcher, logger, tracerProvider, customDistributionPointCRLVerification)
	if err != nil {
		closerFn.Execute()
		_ = httpService.Close()
		return nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}
	s := service.New(httpService, grpcService)
	s.AddCloseFunc(closerFn.Execute)
	return s, nil
}
