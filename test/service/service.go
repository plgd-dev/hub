package service

import (
	"context"
	"errors"
	"testing"

	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	c2cgwService "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	coapgw "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	grpcgwConfig "github.com/plgd-dev/hub/v2/grpc-gateway/service"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	isService "github.com/plgd-dev/hub/v2/identity-store/service"
	isTest "github.com/plgd-dev/hub/v2/identity-store/test"
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
	logger := log.NewLogger(logCfg)
	tlsConfig := config.MakeTLSClientConfig()
	certManager, err := cmClient.New(tlsConfig, logger)
	require.NoError(t, err)
	defer certManager.Close()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017").SetTLSConfig(certManager.GetTLSConfig()))
	require.NoError(t, err)
	dbs, err := client.ListDatabaseNames(ctx, bson.M{})
	if errors.Is(err, mongo.ErrNilDocument) {
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
	IS     isService.Config
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

type SetUpOption = func(cfg *Config)

func SetUp(ctx context.Context, t *testing.T, opts ...SetUpOption) (TearDown func()) {
	config := Config{
		COAPGW: coapgwTest.MakeConfig(t),
		RD:     rdTest.MakeConfig(t),
		GRPCGW: grpcgwTest.MakeConfig(t),
		RA:     raTest.MakeConfig(t),
		IS:     isTest.MakeConfig(t),
	}

	for _, o := range opts {
		o(&config)
	}

	ClearDB(ctx, t)
	oauthShutdown := oauthTest.SetUp(t)
	isShutdown := isTest.New(t, config.IS)
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
		isShutdown()
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

func SetUpServices(ctx context.Context, t *testing.T, servicesConfig SetUpServicesConfig, opts ...SetUpOption) func() {
	var tearDown fn.FuncList
	config := Config{
		COAPGW: coapgwTest.MakeConfig(t),
		RD:     rdTest.MakeConfig(t),
		GRPCGW: grpcgwTest.MakeConfig(t),
		RA:     raTest.MakeConfig(t),
		IS:     isTest.MakeConfig(t),
	}

	for _, o := range opts {
		o(&config)
	}

	ClearDB(ctx, t)
	if servicesConfig&SetUpServicesOAuth != 0 {
		oauthShutdown := oauthTest.SetUp(t)
		tearDown.AddFunc(oauthShutdown)
	}
	if servicesConfig&SetUpServicesId != 0 {
		isShutdown := isTest.New(t, config.IS)
		tearDown.AddFunc(isShutdown)
	}
	if servicesConfig&SetUpServicesResourceAggregate != 0 {
		raShutdown := raTest.New(t, config.RA)
		tearDown.AddFunc(raShutdown)
	}
	if servicesConfig&SetUpServicesResourceDirectory != 0 {
		rdShutdown := rdTest.New(t, config.RD)
		tearDown.AddFunc(rdShutdown)
	}
	if servicesConfig&SetUpServicesGrpcGateway != 0 {
		grpcShutdown := grpcgwTest.New(t, config.GRPCGW)
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
		secureGWShutdown := coapgwTest.New(t, config.COAPGW)
		tearDown.AddFunc(secureGWShutdown)
	}
	return tearDown.ToFunction()
}
