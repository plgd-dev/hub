package test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
)

func Owner(i int) string {
	return "owner" + strconv.Itoa(i)
}

func ConfigurationID(i int) string {
	if id, ok := RuntimeConfig.configurationIds[i]; ok {
		return id
	}
	id := uuid.NewString()
	RuntimeConfig.configurationIds[i] = id
	return id
}

func ConfigurationName(i int) string {
	return "cfg" + strconv.Itoa(i)
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
			TimeToLive: 1337 + int64(i),
		})
	}
	return resources
}

type (
	calculateInitialVersionNumber = func(iteration int) uint64
)

func getConfigurations(t *testing.T, n int, calcVersion calculateInitialVersionNumber) map[string]store.Configuration {
	versions := make(map[int]uint64, RuntimeConfig.NumConfigurations)
	owners := make(map[int]string, RuntimeConfig.NumConfigurations)
	configurations := make(map[string]store.Configuration)
	for i := range n {
		version, ok := versions[i%RuntimeConfig.NumConfigurations]
		if !ok {
			version = 0
			if calcVersion != nil {
				version = calcVersion(i)
			}
			versions[i%RuntimeConfig.NumConfigurations] = version
		}
		versions[i%RuntimeConfig.NumConfigurations]++
		owner, ok := owners[i%RuntimeConfig.NumConfigurations]
		if !ok {
			owner = Owner(i % RuntimeConfig.NumOwners)
			owners[i%RuntimeConfig.NumConfigurations] = owner
		}
		conf := &pb.Configuration{
			Id:        ConfigurationID(i % RuntimeConfig.NumConfigurations),
			Version:   version,
			Resources: ConfigurationResources(t, i%16, (i%5)+1),
			Owner:     owner,
			Timestamp: time.Now().UnixNano(),
		}
		conf.Normalize()
		configuration, ok := configurations[conf.GetId()]
		if !ok {
			conf.Name = ConfigurationName(i % RuntimeConfig.NumConfigurations)
			configuration = store.MakeFirstConfiguration(conf)
			configurations[conf.GetId()] = configuration
			continue
		}

		conf.Name = configuration.Latest.Name
		latest := store.ConfigurationVersion{
			Name:      conf.GetName(),
			Version:   conf.GetVersion(),
			Resources: conf.GetResources(),
			Timestamp: conf.GetTimestamp(),
		}
		configuration.Latest = &latest
		configuration.Versions = append(configuration.Versions, latest)
		configurations[conf.GetId()] = configuration
	}
	return configurations
}

func AddConfigurationsToStore(ctx context.Context, t *testing.T, s store.Store, n int, calcVersion calculateInitialVersionNumber) map[string]store.Configuration {
	configurations := getConfigurations(t, n, calcVersion)
	configurationsToInsert := make([]*store.Configuration, 0, len(configurations))
	for _, c := range configurations {
		configurationToInsert := &c
		configurationsToInsert = append(configurationsToInsert, configurationToInsert)
	}
	err := s.InsertConfigurations(ctx, configurationsToInsert...)
	require.NoError(t, err)
	return configurations
}

func AddConfigurations(ctx context.Context, t *testing.T, ownerClaim string, ssc pb.SnippetServiceClient, n int, calcVersion calculateInitialVersionNumber) map[string]store.Configuration {
	configurations := getConfigurations(t, n, calcVersion)
	for _, c := range configurations {
		ctxWithToken := pkgGrpc.CtxWithToken(ctx, GetTokenWithOwnerClaim(t, c.Owner, ownerClaim))
		c.RangeVersions(func(i int, conf *pb.Configuration) bool {
			if i == 0 {
				createdConf, err := ssc.CreateConfiguration(ctxWithToken, conf)
				require.NoError(t, err)
				c.Latest.Timestamp = createdConf.GetTimestamp()
				c.Versions[i].Timestamp = createdConf.GetTimestamp()
				return true
			}
			updatedConf, err := ssc.UpdateConfiguration(ctxWithToken, conf)
			require.NoError(t, err)
			c.Versions[i].Timestamp = updatedConf.GetTimestamp()
			return true
		})
		configurations[c.Id] = c
	}
	return configurations
}
