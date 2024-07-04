package test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/service"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	hubTest "github.com/plgd-dev/hub/v2/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/stretchr/testify/require"
)

func DeviceID(i int) string {
	return "device" + strconv.Itoa(i)
}

func AppliedConfigurationID(i int) string {
	if id, ok := RuntimeConfig.appliedConfigurationIds[i]; ok {
		return id
	}
	id := uuid.NewString()
	RuntimeConfig.appliedConfigurationIds[i] = id
	return id
}

func SetAppliedConfigurationExecutedBy(ac *pb.AppliedConfiguration, i int) {
	if i%RuntimeConfig.NumConfigurations == 0 {
		ac.ExecutedBy = pb.MakeExecutedByOnDemand()
		return
	}
	ac.ExecutedBy = pb.MakeExecutedByConditionId(ConditionID(i), uint64(i%RuntimeConfig.NumConditions))
}

func AppliedConfigurationResource(t *testing.T, deviceID string, start, n int) []*pb.AppliedConfiguration_Resource {
	resources := make([]*pb.AppliedConfiguration_Resource, 0, n)
	for i := start; i < start+n; i++ {
		correlationID := "corID" + strconv.Itoa(i)
		resource := &pb.AppliedConfiguration_Resource{
			Href:          hubTest.TestResourceLightInstanceHref(strconv.Itoa(i)),
			CorrelationId: correlationID,
			Status:        pb.AppliedConfiguration_Resource_Status(1 + i%4),
		}
		if resource.GetStatus() == pb.AppliedConfiguration_Resource_PENDING {
			resource.ValidUntil = time.Now().Add(time.Minute * -3).Add(time.Minute * time.Duration(i)).UnixNano()
		}
		if resource.GetStatus() == pb.AppliedConfiguration_Resource_DONE {
			resource.ResourceUpdated = pbTest.MakeResourceUpdated(t,
				deviceID,
				resource.GetHref(),
				hubTest.TestResourceLightInstanceResourceTypes,
				correlationID,
				map[string]interface{}{
					"power": i,
				},
			)
		}
		resources = append(resources, resource)
	}
	return resources
}

func getAppliedConfigurations(t *testing.T) map[string]*store.AppliedConfiguration {
	owners := make(map[int]string, RuntimeConfig.NumConfigurations)
	acs := make(map[string]*store.AppliedConfiguration)
	i := 0
	for d := range RuntimeConfig.numDevices {
		for c := range RuntimeConfig.NumConfigurations {
			owner, ok := owners[i%RuntimeConfig.NumConfigurations]
			if !ok {
				owner = Owner(i % RuntimeConfig.NumOwners)
				owners[i%RuntimeConfig.NumConfigurations] = owner
			}
			deviceID := DeviceID(d)
			ac := store.MakeAppliedConfiguration(&pb.AppliedConfiguration{
				Id:       AppliedConfigurationID(i),
				DeviceId: deviceID,
				Owner:    owner,
				ConfigurationId: &pb.AppliedConfiguration_RelationTo{
					Id:      ConfigurationID(c),
					Version: uint64(i % RuntimeConfig.NumConfigurations),
				},
				Resources: AppliedConfigurationResource(t, deviceID, i%16, (i%5)+1),
				Timestamp: time.Now().UnixNano(),
			})
			SetAppliedConfigurationExecutedBy(ac.GetAppliedConfiguration(), i)
			acs[ac.GetId()] = &ac
			i++
		}
	}
	return acs
}

func AddAppliedConfigurationsToStore(ctx context.Context, t *testing.T, s store.Store) map[string]*store.AppliedConfiguration {
	acs := getAppliedConfigurations(t)
	acsToInsert := make([]*store.AppliedConfiguration, 0, len(acs))
	for _, c := range acs {
		acsToInsert = append(acsToInsert, c)
	}
	err := s.InsertAppliedConfigurations(ctx, acsToInsert...)
	require.NoError(t, err)
	return acs
}

func AddAppliedConfigurations(ctx context.Context, t *testing.T, ownerClaim string, ss *service.Service) map[string]*store.AppliedConfiguration {
	configurations := getAppliedConfigurations(t)
	for _, c := range configurations {
		ctxWithToken := pkgGrpc.CtxWithIncomingToken(ctx, GetTokenWithOwnerClaim(t, c.GetOwner(), ownerClaim))
		newConf, err := ss.SnippetServiceServer().CreateAppliedConfiguration(ctxWithToken, c.GetAppliedConfiguration())
		require.NoError(t, err)
		c.Timestamp = newConf.GetTimestamp()
		configurations[newConf.GetId()] = c
	}
	return configurations
}
