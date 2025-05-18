package test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/stretchr/testify/require"
)

func ConditionID(i int) string {
	if id, ok := RuntimeConfig.conditionIds[i]; ok {
		return id
	}
	id := uuid.NewString()
	RuntimeConfig.conditionIds[i] = id
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
	return strings.Unique(stringSlice("device", start, n))
}

func ConditionResourceTypeFilter(start, n int) []string {
	return strings.Unique(stringSlice("rt", start, n))
}

func ConditionResourceHrefFilter(start, n int) []string {
	return strings.Unique(stringSlice("/href/", start, n))
}

func ConditionJqExpressionFilter(i int) string {
	return "jq" + strconv.Itoa(i)
}

func ConditionApiAccessToken(i int) string {
	return "token" + strconv.Itoa(i)
}

func getConditions(n int, calcVersion calculateInitialVersionNumber) map[string]store.Condition {
	versions := make(map[int]uint64, RuntimeConfig.NumConditions)
	owners := make(map[int]string, RuntimeConfig.NumConditions)
	conditions := make(map[string]store.Condition)
	for i := range n {
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
			owner = Owner(i % RuntimeConfig.NumOwners)
			owners[i%RuntimeConfig.NumConditions] = owner
		}
		cond := &pb.Condition{
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
			Timestamp:          time.Now().UnixNano(),
		}
		cond.Normalize()
		condition, ok := conditions[cond.GetId()]
		if !ok {
			cond.Name = ConditionName(i % RuntimeConfig.NumConditions)
			condition = store.MakeFirstCondition(cond)
			conditions[cond.GetId()] = condition
			continue
		}

		cond.Name = condition.Latest.Name
		latest := store.ConditionVersion{
			Name:               cond.GetName(),
			Version:            cond.GetVersion(),
			Enabled:            cond.GetEnabled(),
			Timestamp:          cond.GetTimestamp(),
			DeviceIdFilter:     cond.GetDeviceIdFilter(),
			ResourceTypeFilter: cond.GetResourceTypeFilter(),
			ResourceHrefFilter: cond.GetResourceHrefFilter(),
			JqExpressionFilter: cond.GetJqExpressionFilter(),
			ApiAccessToken:     cond.GetApiAccessToken(),
		}
		condition.Latest = &latest
		condition.Versions = append(condition.Versions, latest)
		conditions[cond.GetId()] = condition
	}
	return conditions
}

func AddConditionsToStore(ctx context.Context, t *testing.T, s store.Store, n int, calcVersion calculateInitialVersionNumber) map[string]store.Condition {
	conditions := getConditions(n, calcVersion)
	conditionsToInsert := make([]*store.Condition, 0, len(conditions))
	for _, condition := range conditions {
		conditionToInsert := &condition
		conditionsToInsert = append(conditionsToInsert, conditionToInsert)
	}
	err := s.InsertConditions(ctx, conditionsToInsert...)
	require.NoError(t, err)
	return conditions
}

func AddConditions(ctx context.Context, t *testing.T, ownerClaim string, ssc pb.SnippetServiceClient, n int, calcVersion calculateInitialVersionNumber) map[string]store.Condition {
	conditions := getConditions(n, calcVersion)
	for _, c := range conditions {
		ctxWithToken := pkgGrpc.CtxWithToken(ctx, GetTokenWithOwnerClaim(t, c.Owner, ownerClaim))
		c.RangeVersions(func(i int, cond *pb.Condition) bool {
			if i == 0 {
				createdCond, err := ssc.CreateCondition(ctxWithToken, cond)
				require.NoError(t, err)
				c.Latest.Timestamp = createdCond.GetTimestamp()
				c.Versions[i].Timestamp = createdCond.GetTimestamp()
				return true
			}
			updatedCond, err := ssc.UpdateCondition(ctxWithToken, cond)
			require.NoError(t, err)
			c.Versions[i].Timestamp = updatedCond.GetTimestamp()
			return true
		})
		conditions[c.Id] = c
	}
	return conditions
}
