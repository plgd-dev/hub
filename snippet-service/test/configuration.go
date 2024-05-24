package test

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

var testConfigurationIDs = make(map[int]string)

func ConfigurationID(i int) string {
	if id, ok := testConfigurationIDs[i]; ok {
		return id
	}
	id := uuid.New().String()
	testConfigurationIDs[i] = id
	return id
}

func ConfigurationName(i int) string {
	return "cfg" + strconv.Itoa(i)
}

func ConfigurationOwner(i int) string {
	return "owner" + strconv.Itoa(i)
}

func ConfigurationResources(t *testing.T, start, n int) []*pb.Configuration_Resource {
	resources := make([]*pb.Configuration_Resource, 0, n)
	for i := start; i < start+n; i++ {
		resources = append(resources, &pb.Configuration_Resource{
			Href: hubTest.TestResourceLightInstanceHref(strconv.Itoa(i)),
			Content: &commands.Content{
				Data: hubTest.EncodeToCbor(t, map[string]interface{}{
					"power": i,
				}),
				ContentType:       message.AppOcfCbor.String(),
				CoapContentFormat: int32(message.AppOcfCbor),
			},
			TimeToLive: 1337,
		})
	}
	return resources
}

type (
	onCreateConfiguration         = func(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error)
	onUpdateConfiguration         = func(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error)
	calculateInitialVersionNumber = func(iteration int) uint64
)

func addConfigurations(ctx context.Context, t *testing.T, n int, calcVersion calculateInitialVersionNumber, create onCreateConfiguration, update onUpdateConfiguration) map[string]store.Configuration {
	const numConfigs = 10
	const numOwners = 3
	versions := make(map[int]uint64, numConfigs)
	owners := make(map[int]string, numConfigs)
	configurations := make(map[string]store.Configuration)
	for i := 0; i < n; i++ {
		version, ok := versions[i%numConfigs]
		if !ok {
			version = 0
			if calcVersion != nil {
				version = calcVersion(i)
			}
			versions[i%numConfigs] = version
		}
		versions[i%numConfigs]++
		owner, ok := owners[i%numConfigs]
		if !ok {
			owner = ConfigurationOwner(i % numOwners)
			owners[i%numConfigs] = owner
		}
		confIn := &pb.Configuration{
			Id:        ConfigurationID(i % numConfigs),
			Version:   version,
			Resources: ConfigurationResources(t, i%16, (i%5)+1),
			Owner:     owner,
		}
		var conf *pb.Configuration
		var err error
		if !ok {
			confIn.Name = ConfigurationName(i % numConfigs)
			conf, err = create(ctx, confIn)
			require.NoError(t, err)
		} else {
			conf, err = update(ctx, confIn)
			require.NoError(t, err)
		}

		configuration, ok := configurations[conf.GetId()]
		if !ok {
			configuration = store.Configuration{
				Id:    conf.GetId(),
				Owner: conf.GetOwner(),
				Name:  conf.GetName(),
			}
			configurations[conf.GetId()] = configuration
		}
		configuration.Versions = append(configuration.Versions, store.ConfigurationVersion{
			Version:   conf.GetVersion(),
			Resources: conf.GetResources(),
		})
		configurations[conf.GetId()] = configuration
	}
	return configurations
}

func AddConfigurationsToStore(ctx context.Context, t *testing.T, s store.Store, n int, calcVersion calculateInitialVersionNumber) map[string]store.Configuration {
	return addConfigurations(ctx, t, n, calcVersion, s.CreateConfiguration, s.UpdateConfiguration)
}

func AddConfigurations(ctx context.Context, t *testing.T, ownerClaim string, c pb.SnippetServiceClient, n int, calcVersion calculateInitialVersionNumber) map[string]store.Configuration {
	tokens := make(map[string]string)
	getTokenWithOwnerClaim := func(owner string) string {
		token, ok := tokens[owner]
		if ok {
			return token
		}
		token = oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
			ownerClaim: owner,
		})
		tokens[owner] = token
		return token
	}

	return addConfigurations(ctx, t, n, calcVersion, func(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
		ctxWithToken := pkgGrpc.CtxWithToken(ctx, getTokenWithOwnerClaim(conf.GetOwner()))
		return c.CreateConfiguration(ctxWithToken, conf)
	}, func(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
		ctxWithToken := pkgGrpc.CtxWithToken(ctx, getTokenWithOwnerClaim(conf.GetOwner()))
		return c.UpdateConfiguration(ctxWithToken, conf)
	},
	)
}
