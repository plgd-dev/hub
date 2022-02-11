package service

import (
	"context"
	"testing"

	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	c2cgwService "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	coapgw "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	grpcgwConfig "github.com/plgd-dev/hub/v2/grpc-gateway/service"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	idService "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/log"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdService "github.com/plgd-dev/hub/v2/resource-directory/service"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ClearDB(ctx context.Context, t *testing.T) {
	logCfg := log.MakeDefaultConfig()
	logCfg.Debug = true
	logger := log.NewLogger(logCfg)
	tlsConfig := config.MakeTLSClientConfig()
	certManager, err := cmClient.New(tlsConfig, logger)
	require.NoError(t, err)
	defer certManager.Close()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017").SetTLSConfig(certManager.GetTLSConfig()))
	require.NoError(t, err)
	dbs, err := client.ListDatabaseNames(ctx, bson.M{})
	if mongo.ErrNilDocument == err {
		return
	}
	require.NoError(t, err)
	for _, db := range dbs {
		if db == "admin" {
			continue
		}
		err = client.Database(db).Drop(ctx)
		require.NoError(t, err)
	}
	err = client.Disconnect(ctx)
	require.NoError(t, err)
}

type Config struct {
	COAPGW coapgw.Config
	RD     rdService.Config
	GRPCGW grpcgwConfig.Config
	RA     raService.Config
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

type SetUpOption = func(cfg *Config)

func SetUp(ctx context.Context, t *testing.T, opts ...SetUpOption) (TearDown func()) {
	config := Config{
		COAPGW: coapgwTest.MakeConfig(t),
		RD:     rdTest.MakeConfig(t),
		GRPCGW: grpcgwTest.MakeConfig(t),
		RA:     raTest.MakeConfig(t),
	}

	for _, o := range opts {
		o(&config)
	}

	ClearDB(ctx, t)
	oauthShutdown := oauthTest.SetUp(t)
	idShutdown := idService.SetUp(t)
	raShutdown := raTest.New(t, config.RA)
	rdShutdown := rdTest.New(t, config.RD)
	grpcShutdown := grpcgwTest.New(t, config.GRPCGW)
	c2cgwShutdown := c2cgwService.SetUp(t)
	caShutdown := caService.SetUp(t)
	secureGWShutdown := coapgwTest.New(t, config.COAPGW)

	return func() {
		caShutdown()
		c2cgwShutdown()
		grpcShutdown()
		secureGWShutdown()
		rdShutdown()
		raShutdown()
		idShutdown()
		oauthShutdown()
	}
}

type SetUpServicesConfig uint16

const (
	SetUpServicesOAuth SetUpServicesConfig = 1 << iota
	SetUpServicesId
	SetUpServicesCertificateAuthority
	SetUpServicesCloud2CloudGateway
	SetUpServicesCoapGateway
	SetUpServicesGrpcGateway
	SetUpServicesResourceAggregate
	SetUpServicesResourceDirectory
)

func SetUpServices(ctx context.Context, t *testing.T, servicesConfig SetUpServicesConfig) func() {
	var tearDown fn.FuncList

	ClearDB(ctx, t)
	if servicesConfig&SetUpServicesOAuth != 0 {
		oauthShutdown := oauthTest.SetUp(t)
		tearDown.AddFunc(oauthShutdown)
	}
	if servicesConfig&SetUpServicesId != 0 {
		idShutdown := idService.SetUp(t)
		tearDown.AddFunc(idShutdown)
	}
	if servicesConfig&SetUpServicesResourceAggregate != 0 {
		raShutdown := raTest.SetUp(t)
		tearDown.AddFunc(raShutdown)
	}
	if servicesConfig&SetUpServicesResourceDirectory != 0 {
		rdShutdown := rdTest.SetUp(t)
		tearDown.AddFunc(rdShutdown)
	}
	if servicesConfig&SetUpServicesGrpcGateway != 0 {
		grpcShutdown := grpcgwTest.SetUp(t)
		tearDown.AddFunc(grpcShutdown)
	}
	if servicesConfig&SetUpServicesCloud2CloudGateway != 0 {
		c2cgwShutdown := c2cgwService.SetUp(t)
		tearDown.AddFunc(c2cgwShutdown)
	}
	if servicesConfig&SetUpServicesCertificateAuthority != 0 {
		caShutdown := caService.SetUp(t)
		tearDown.AddFunc(caShutdown)
	}
	if servicesConfig&SetUpServicesCoapGateway != 0 {
		secureGWShutdown := coapgwTest.SetUp(t)
		tearDown.AddFunc(secureGWShutdown)
	}
	return tearDown.ToFunction()
}
