package service

import (
	"context"
	"errors"
	"time"

	ca "github.com/plgd-dev/hub/v2/certificate-authority/service"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	c2cgwService "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	coapgw "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	grpcgwConfig "github.com/plgd-dev/hub/v2/grpc-gateway/service"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	isService "github.com/plgd-dev/hub/v2/identity-store/service"
	isTest "github.com/plgd-dev/hub/v2/identity-store/test"
	m2mOauthService "github.com/plgd-dev/hub/v2/m2m-oauth-server/service"
	m2mOauthTest "github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdService "github.com/plgd-dev/hub/v2/resource-directory/service"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace/noop"
)

var filterOutClearDB = map[string]bool{
	"admin":  true,
	"config": true,
	"local":  true,
}

func clearMongoDB(ctx context.Context, t require.TestingT, certManager *cmClient.CertManager, logger *log.WrapSuggarLogger) {
	// clear mongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017").SetTLSConfig(certManager.GetTLSConfig()))
	if err != nil {
		logger.Infof("cannot connect to mongoDB: %v", err)
		// if mongoDB is not running, we can skip clearing it
		return
	}
	dbs, err := client.ListDatabaseNames(ctx, bson.M{})
	if errors.Is(err, mongo.ErrNilDocument) {
		return
	}
	require.NoError(t, err)
	for _, db := range dbs {
		if filterOutClearDB[db] {
			continue
		}
		err = client.Database(db).Drop(ctx)
		require.NoError(t, err)
	}
	err = client.Disconnect(ctx)
	require.NoError(t, err)
}

func clearCqlDB(ctx context.Context, t require.TestingT, certManager *cmClient.CertManager, logger *log.WrapSuggarLogger) {
	// clear cqlDB
	cqlCfg := config.MakeEventsStoreCqlDBConfig()
	cql, err := cqldb.New(ctx, cqlCfg.Embedded, certManager.GetTLSConfig(), logger, noop.NewTracerProvider())
	if err != nil {
		logger.Infof("cannot connect to cqlDB: %v", err)
		// if cqlDB is not running, we can skip clearing it
		return
	}
	require.NoError(t, err)
	defer cql.Close()

	// we need to use same key-space for all services
	err = cql.DropKeyspace(ctx)
	require.NoError(t, err)
}

func ClearDB(ctx context.Context, t require.TestingT) {
	logCfg := log.MakeDefaultConfig()
	logger := log.NewLogger(logCfg)
	tlsConfig := config.MakeTLSClientConfig()
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()
	certManager, err := cmClient.New(tlsConfig, fileWatcher, logger, noop.NewTracerProvider())
	require.NoError(t, err)
	defer certManager.Close()

	clearMongoDB(ctx, t, certManager, logger)
	clearCqlDB(ctx, t, certManager, logger)
}

type Config struct {
	COAPGW   coapgw.Config
	RD       rdService.Config
	GRPCGW   grpcgwConfig.Config
	RA       raService.Config
	IS       isService.Config
	CA       ca.Config
	OAUTH    oauthService.Config
	M2MOAUTH m2mOauthService.Config
}

func WithCOAPGWConfig(coapgwCfg coapgw.Config) SetUpOption {
	return func(cfg *Config) {
		cfg.COAPGW = coapgwCfg
	}
}

func WithRDConfig(rd rdService.Config) SetUpOption {
	return func(cfg *Config) {
		cfg.RD = rd
	}
}

func WithGRPCGWConfig(grpcCfg grpcgwConfig.Config) SetUpOption {
	return func(cfg *Config) {
		cfg.GRPCGW = grpcCfg
	}
}

func WithRAConfig(ra raService.Config) SetUpOption {
	return func(cfg *Config) {
		cfg.RA = ra
	}
}

func WithISConfig(is isService.Config) SetUpOption {
	return func(cfg *Config) {
		cfg.IS = is
	}
}

func WithCAConfig(ca ca.Config) SetUpOption {
	return func(cfg *Config) {
		cfg.CA = ca
	}
}

func WithOAuthConfig(oauth oauthService.Config) SetUpOption {
	return func(cfg *Config) {
		cfg.OAUTH = oauth
	}
}

func WithM2MOAuthConfig(oauth m2mOauthService.Config) SetUpOption {
	return func(cfg *Config) {
		cfg.M2MOAUTH = oauth
	}
}

type SetUpOption = func(cfg *Config)

func SetUp(ctx context.Context, t require.TestingT, opts ...SetUpOption) func() {
	config := Config{
		COAPGW:   coapgwTest.MakeConfig(t),
		RD:       rdTest.MakeConfig(t),
		GRPCGW:   grpcgwTest.MakeConfig(t),
		RA:       raTest.MakeConfig(t),
		IS:       isTest.MakeConfig(t),
		CA:       caService.MakeConfig(t),
		OAUTH:    oauthTest.MakeConfig(t),
		M2MOAUTH: m2mOauthTest.MakeConfig(t),
	}

	for _, o := range opts {
		o(&config)
	}

	var tearDown fn.FuncList
	deferedTearDown := true
	defer func() {
		if deferedTearDown {
			tearDown.Execute()
		}
	}()

	ClearDB(ctx, t)
	oauthShutdown := oauthTest.New(t, config.OAUTH)
	tearDown.AddFunc(oauthShutdown)
	m2mTearDown := m2mOauthTest.New(t, config.M2MOAUTH)
	tearDown.AddFunc(m2mTearDown)
	isShutdown := isTest.New(t, config.IS)
	tearDown.AddFunc(isShutdown)
	raShutdown := raTest.New(t, config.RA)
	tearDown.AddFunc(raShutdown)
	rdShutdown := rdTest.New(t, config.RD)
	tearDown.AddFunc(rdShutdown)
	grpcShutdown := grpcgwTest.New(t, config.GRPCGW)
	tearDown.AddFunc(grpcShutdown)
	c2cgwShutdown := c2cgwService.SetUp(t)
	tearDown.AddFunc(c2cgwShutdown)
	caShutdown := caService.New(t, config.CA)
	tearDown.AddFunc(caShutdown)
	secureGWShutdown := coapgwTest.New(t, config.COAPGW)
	tearDown.AddFunc(secureGWShutdown)

	// wait for all services to start
	time.Sleep(time.Second)

	deferedTearDown = false
	return func() {
		tearDown.Execute()

		// wait for all services to be closed
		time.Sleep(time.Second)
	}
}

type SetUpServicesConfig uint16

const (
	SetUpServicesOAuth SetUpServicesConfig = 1 << iota
	SetUpServicesMachine2MachineOAuth
	SetUpServicesId
	SetUpServicesResourceAggregate
	SetUpServicesResourceDirectory
	SetUpServicesCertificateAuthority
	SetUpServicesCloud2CloudGateway
	SetUpServicesCoapGateway
	SetUpServicesGrpcGateway
	// need to be last
	SetUpServicesMax
)

var setupServicesMap = map[SetUpServicesConfig]func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption){
	SetUpServicesOAuth: func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption) {
		config := Config{
			OAUTH: oauthTest.MakeConfig(t),
		}
		for _, o := range opts {
			o(&config)
		}
		err := config.OAUTH.Validate()
		require.NoError(t, err)
		oauthShutdown := oauthTest.New(t, config.OAUTH)
		tearDown.AddFunc(oauthShutdown)
	},
	SetUpServicesMachine2MachineOAuth: func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption) {
		config := Config{
			M2MOAUTH: m2mOauthTest.MakeConfig(t),
		}
		for _, o := range opts {
			o(&config)
		}
		err := config.M2MOAUTH.Validate()
		require.NoError(t, err)
		m2mTearDown := m2mOauthTest.New(t, config.M2MOAUTH)
		tearDown.AddFunc(m2mTearDown)
	},
	SetUpServicesId: func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption) {
		config := Config{
			IS: isTest.MakeConfig(t),
		}
		for _, o := range opts {
			o(&config)
		}
		err := config.IS.Validate()
		require.NoError(t, err)
		isShutdown := isTest.New(t, config.IS)
		tearDown.AddFunc(isShutdown)
	},
	SetUpServicesResourceAggregate: func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption) {
		config := Config{
			RA: raTest.MakeConfig(t),
		}
		for _, o := range opts {
			o(&config)
		}
		err := config.RA.Validate()
		require.NoError(t, err)
		raShutdown := raTest.New(t, config.RA)
		tearDown.AddFunc(raShutdown)
	},
	SetUpServicesResourceDirectory: func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption) {
		config := Config{
			RD: rdTest.MakeConfig(t),
		}
		for _, o := range opts {
			o(&config)
		}
		err := config.RD.Validate()
		require.NoError(t, err)
		rdShutdown := rdTest.New(t, config.RD)
		tearDown.AddFunc(rdShutdown)
	},
	SetUpServicesGrpcGateway: func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption) {
		config := Config{
			GRPCGW: grpcgwTest.MakeConfig(t),
		}
		for _, o := range opts {
			o(&config)
		}
		err := config.GRPCGW.Validate()
		require.NoError(t, err)
		grpcShutdown := grpcgwTest.New(t, config.GRPCGW)
		tearDown.AddFunc(grpcShutdown)
	},
	SetUpServicesCloud2CloudGateway: func(t require.TestingT, tearDown *fn.FuncList, _ ...SetUpOption) {
		c2cgwShutdown := c2cgwService.SetUp(t)
		tearDown.AddFunc(c2cgwShutdown)
	},
	SetUpServicesCertificateAuthority: func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption) {
		config := Config{
			CA: caService.MakeConfig(t),
		}
		for _, o := range opts {
			o(&config)
		}
		err := config.CA.Validate()
		require.NoError(t, err)
		caShutdown := caService.New(t, config.CA)
		tearDown.AddFunc(caShutdown)
	},
	SetUpServicesCoapGateway: func(t require.TestingT, tearDown *fn.FuncList, opts ...SetUpOption) {
		config := Config{
			COAPGW: coapgwTest.MakeConfig(t),
		}
		for _, o := range opts {
			o(&config)
		}
		err := config.COAPGW.Validate()
		require.NoError(t, err)
		secureGWShutdown := coapgwTest.New(t, config.COAPGW)
		tearDown.AddFunc(secureGWShutdown)
	},
}

func SetUpServices(ctx context.Context, t require.TestingT, servicesConfig SetUpServicesConfig, opts ...SetUpOption) func() {
	var tearDown fn.FuncList
	deferedTearDown := true
	defer func() {
		if deferedTearDown {
			tearDown.Execute()
		}
	}()
	ClearDB(ctx, t)

	for i := SetUpServicesConfig(1); i < SetUpServicesMax; i <<= 1 {
		if servicesConfig&i != 0 {
			if f, ok := setupServicesMap[i]; ok {
				f(t, &tearDown, opts...)
			}
		}
	}
	deferedTearDown = false
	return tearDown.ToFunction()
}
