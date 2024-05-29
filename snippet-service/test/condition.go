package test

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/stretchr/testify/require"
)

func ConditionID(i int) string {
	if id, ok := RuntimeConfig.configurationIds[i]; ok {
		return id
	}
	id := uuid.NewString()
	RuntimeConfig.configurationIds[i] = id
	return id
}

func ConditionName(i int) string {
	return "cond" + strconv.Itoa(i)
}

func stringSlice(prefix string, start, n int) []string {
	slice := make([]string, n)
	for i := start; i < start+n; i++ {
		slice[i-start] = prefix + strconv.Itoa(i)
	}
	return slice
}

func ConditionDeviceIdFilter(start, n int) []string {
	return stringSlice("device", start, n)
}

func ConditionResourceTypeFilter(start, n int) []string {
	return stringSlice("rt", start, n)
}

func ConditionResourceHrefFilter(start, n int) []string {
	return stringSlice("/href/", start, n)
}

func ConditionJqExpressionFilter(i int) string {
	return "jq" + strconv.Itoa(i)
}

func ConditionApiAccessToken(i int) string {
	return "token" + strconv.Itoa(i)
}

type (
	onCreateCondition = func(ctx context.Context, conf *pb.Condition) (*pb.Condition, error)
	onUpdateCondition = func(ctx context.Context, conf *pb.Condition) (*pb.Condition, error)
)

func addConditions(ctx context.Context, t *testing.T, n int, calcVersion calculateInitialVersionNumber, create onCreateCondition, update onUpdateCondition) map[string]store.Condition {
	versions := make(map[int]uint64, RuntimeConfig.NumConditions)
	owners := make(map[int]string, RuntimeConfig.NumConditions)
	conditions := make(map[string]store.Condition)
	for i := 0; i < n; i++ {
		version, ok := versions[i%RuntimeConfig.NumConditions]
		if !ok {
			version = 0
			if calcVersion != nil {
				version = calcVersion(i)
			}
			versions[i%RuntimeConfig.NumConditions] = version
		}
		versions[i%RuntimeConfig.NumConditions]++
		owner, ok := owners[i%RuntimeConfig.NumConditions]
		if !ok {
			owner = Owner(i % RuntimeConfig.numOwners)
			owners[i%RuntimeConfig.NumConditions] = owner
		}
		condIn := &pb.Condition{
			Id:                 ConditionID(i % RuntimeConfig.NumConditions),
			ConfigurationId:    ConfigurationID(i % RuntimeConfig.NumConfigurations),
			Enabled:            i%2 == 0,
			Version:            version,
			Owner:              owner,
			DeviceIdFilter:     ConditionDeviceIdFilter(i%RuntimeConfig.numDevices, RuntimeConfig.numDevices),
			ResourceTypeFilter: ConditionResourceTypeFilter(i%RuntimeConfig.numResourceTypes, RuntimeConfig.numResourceTypes),
			ResourceHrefFilter: ConditionResourceHrefFilter(i%RuntimeConfig.numResources, RuntimeConfig.numResources),
			JqExpressionFilter: ConditionJqExpressionFilter(i),
			ApiAccessToken:     ConditionApiAccessToken(i % RuntimeConfig.NumConditions),
		}
		var cond *pb.Condition
		var err error
		if !ok {
			condIn.Name = ConditionName(i % RuntimeConfig.NumConditions)
			cond, err = create(ctx, condIn)
			require.NoError(t, err)
		} else {
			cond, err = update(ctx, condIn)
			require.NoError(t, err)
		}

		condition, ok := conditions[cond.GetId()]
		if !ok {
			condition = store.Condition{
				Id:              cond.GetId(),
				ConfigurationId: cond.GetConfigurationId(),
				Owner:           cond.GetOwner(),
				Name:            cond.GetName(),
			}
		}
		condition.Enabled = cond.GetEnabled()
		condition.Timestamp = cond.GetTimestamp()
		condition.ApiAccessToken = cond.GetApiAccessToken()
		condition.Versions = append(condition.Versions, store.ConditionVersion{
			Version:            cond.GetVersion(),
			DeviceIdFilter:     cond.GetDeviceIdFilter(),
			ResourceTypeFilter: cond.GetResourceTypeFilter(),
			ResourceHrefFilter: cond.GetResourceHrefFilter(),
			JqExpressionFilter: cond.GetJqExpressionFilter(),
		})
		conditions[cond.GetId()] = condition
	}
	return conditions
}

func AddConditionsToStore(ctx context.Context, t *testing.T, s store.Store, n int, calcVersion calculateInitialVersionNumber) map[string]store.Condition {
	return addConditions(ctx, t, n, calcVersion, s.CreateCondition, s.UpdateCondition)
}

func AddConditions(ctx context.Context, t *testing.T, ownerClaim string, c pb.SnippetServiceClient, n int, calcVersion calculateInitialVersionNumber) map[string]store.Condition {
	return addConditions(ctx, t, n, calcVersion, func(ctx context.Context, cond *pb.Condition) (*pb.Condition, error) {
		ctxWithToken := pkgGrpc.CtxWithToken(ctx, GetTokenWithOwnerClaim(t, cond.GetOwner(), ownerClaim))
		return c.CreateCondition(ctxWithToken, cond)
	}, func(ctx context.Context, cond *pb.Condition) (*pb.Condition, error) {
		ctxWithToken := pkgGrpc.CtxWithToken(ctx, GetTokenWithOwnerClaim(t, cond.GetOwner(), ownerClaim))
		return c.UpdateCondition(ctxWithToken, cond)
	})
}
