package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	oauthsigner "github.com/plgd-dev/hub/v2/m2m-oauth-server/oauthSigner"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	grpcService "github.com/plgd-dev/hub/v2/m2m-oauth-server/service/grpc"
	httpService "github.com/plgd-dev/hub/v2/m2m-oauth-server/service/http"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/store"
	storeConfig "github.com/plgd-dev/hub/v2/m2m-oauth-server/store/config"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	certManagerServer "github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "m2m-oauth-server"

type Service struct {
	*service.Service

	store store.Store
}

func createStore(ctx context.Context, config storeConfig.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (store.Store, error) {
	if config.Use != database.MongoDB {
		return nil, fmt.Errorf("invalid store use('%v')", config.Use)
	}
	s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("mongodb: %w", err)
	}
	if config.CleanUpDeletedTokens != "" {
		scheduler, err2 := NewExpiredUpdatesChecker(config.CleanUpDeletedTokens, config.ExtendCronParserBySeconds, func() {
			err2 := s.DeleteBlacklistedTokens(ctx, time.Now())
			if err2 != nil {
				log.Errorf("cannot delete expired tokens: %v", err2)
			}
		})
		if err2 != nil {
			s.Close(ctx)
			return nil, fmt.Errorf("cannot create scheduler: %w", err2)
		}
		s.AddCloseFunc(func() {
			err2 := scheduler.Shutdown()
			if err2 != nil {
				log.Errorf("failed to shutdown scheduler: %w", err2)
			}
		})
	}
	return s, nil
}

func newHttpService(ctx context.Context, config HTTPConfig, validatorConfig validator.Config, getOpenIDConfiguration validator.GetOpenIDConfigurationFunc, trustVerification map[string]jwt.TokenIssuerClient, tlsConfig certManagerServer.Config, ss *grpcService.M2MOAuthServiceServer, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*httpService.Service, func(), error) {
	httpValidator, err := validator.New(ctx, validatorConfig, fileWatcher, logger, tracerProvider, validator.WithGetOpenIDConfiguration(getOpenIDConfiguration), validator.WithCustomTokenIssuerClients(trustVerification))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create http validator: %w", err)
	}
	httpService, err := httpService.New(serviceName, httpService.Config{
		Connection: listener.Config{
			Addr: config.Addr,
			TLS:  tlsConfig,
		},
		Authorization: validatorConfig,
		Server:        config.Server,
	}, ss, httpValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		httpValidator.Close()
		return nil, nil, fmt.Errorf("cannot create http service: %w", err)
	}
	return httpService, httpValidator.Close, nil
}

func newGrpcService(ctx context.Context, config grpcService.Config, getOpenIDConfiguration validator.GetOpenIDConfigurationFunc, trustVerification map[string]jwt.TokenIssuerClient, ss *grpcService.M2MOAuthServiceServer, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*grpcService.Service, func(), error) {
	grpcValidator, err := validator.New(ctx, config.Authorization.Config, fileWatcher, logger, tracerProvider, validator.WithGetOpenIDConfiguration(getOpenIDConfiguration), validator.WithCustomTokenIssuerClients(trustVerification))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}
	grpcService, err := grpcService.New(config, ss, grpcValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		grpcValidator.Close()
		return nil, nil, fmt.Errorf("cannot create grpc service: %w", err)
	}
	return grpcService, grpcValidator.Close, nil
}

type tokenIssuerClient struct {
	store      store.Store
	ownerClaim string
}

func (c *tokenIssuerClient) VerifyTokenByRequest(ctx context.Context, accessToken, tokenID string) (*pb.Token, error) {
	owner, err := grpc.ParseOwnerFromJwtToken(c.ownerClaim, accessToken)
	if err != nil {
		return nil, fmt.Errorf("cannot parse owner from token: %w", err)
	}
	var token *pb.Token
	err = c.store.GetTokens(ctx, owner, &pb.GetTokensRequest{
		IdFilter:           []string{tokenID},
		IncludeBlacklisted: true,
	}, func(v *pb.Token) error {
		token = v
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get token(%v): %w", tokenID, err)
	}
	if token == nil {
		return nil, fmt.Errorf("token(%v) not found", tokenID)
	}
	return token, nil
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector.Config, serviceName, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	var closerFn fn.FuncList
	closerFn.AddFunc(otelClient.Close)
	tracerProvider := otelClient.GetTracerProvider()

	db, err := createStore(ctx, config.Clients.Storage, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create store: %w", err)
	}
	closerFn.AddFunc(func() {
		if errC := db.Close(ctx); errC != nil {
			log.Errorf("failed to close store: %w", errC)
		}
	})

	getOpenIDCfg := func(ctx context.Context, c *http.Client, authority string) (openid.Config, error) {
		if authority == config.OAuthSigner.GetAuthority() {
			return httpService.GetOpenIDConfiguration(config.OAuthSigner.GetDomain()), nil
		}
		return openid.GetConfiguration(ctx, c, authority)
	}
	customTokenIssuerClients := map[string]jwt.TokenIssuerClient{
		config.OAuthSigner.GetAuthority(): &tokenIssuerClient{
			store:      db,
			ownerClaim: config.OAuthSigner.OwnerClaim,
		},
	}

	signer, err := oauthsigner.New(ctx, config.OAuthSigner, getOpenIDCfg, customTokenIssuerClients, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create oauth signer: %w", err)
	}
	closerFn.AddFunc(signer.Close)

	m2mOAuthService := grpcService.NewM2MOAuthServerServer(db, signer, logger)

	grpcService, grpcServiceClose, err := newGrpcService(ctx, config.APIs.GRPC, getOpenIDCfg, customTokenIssuerClients, m2mOAuthService, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, err
	}
	closerFn.AddFunc(grpcServiceClose)

	httpService, httpServiceClose, err := newHttpService(ctx, config.APIs.HTTP, config.APIs.GRPC.Authorization.Config, getOpenIDCfg, customTokenIssuerClients, config.APIs.GRPC.TLS,
		m2mOAuthService, fileWatcher, logger, tracerProvider)
	if err != nil {
		grpcService.Close()
		closerFn.Execute()
		return nil, err
	}
	closerFn.AddFunc(httpServiceClose)

	s := service.New(grpcService, httpService)
	s.AddCloseFunc(closerFn.Execute)
	return &Service{
		Service: s,
		store:   db,
	}, nil
}
