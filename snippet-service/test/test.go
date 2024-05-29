package test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

var (
	RuntimeConfig struct {
		numOwners         int
		numDevices        int
		numResources      int
		numResourceTypes  int
		NumConfigurations int
		NumConditions     int
		configurationIds  map[int]string
		conditionIds      map[int]string
	}

	tokens = make(map[string]string)
)

func init() {
	RuntimeConfig.configurationIds = make(map[int]string)
	RuntimeConfig.numOwners = 3
	RuntimeConfig.numDevices = 5
	RuntimeConfig.numResources = 5
	RuntimeConfig.numResourceTypes = 7
	RuntimeConfig.NumConfigurations = 10
	RuntimeConfig.NumConditions = RuntimeConfig.NumConfigurations * 2
}

func GetTokenWithOwnerClaim(t *testing.T, owner, ownerClaim string) string {
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

func CmpJSON(t *testing.T, want, got interface{}) {
	wantJson, err := json.Encode(want)
	require.NoError(t, err)
	gotJson, err := json.Encode(got)
	require.NoError(t, err)
	require.JSONEq(t, string(wantJson), string(gotJson))
}

func cmpConfigurationResources(t *testing.T, want, got []*pb.Configuration_Resource) {
	require.Len(t, got, len(want))
	for i := range want {
		wantData, ok := test.DecodeCbor(t, want[i].GetContent().GetData()).(map[interface{}]interface{})
		require.True(t, ok)
		gotData, ok := test.DecodeCbor(t, got[i].GetContent().GetData()).(map[interface{}]interface{})
		require.True(t, ok)
		require.Equal(t, wantData, gotData)
		want[i].Content.Data = nil
		got[i].Content.Data = nil
	}
	CmpJSON(t, want, got)
}

func CmpConfiguration(t *testing.T, want, got *pb.Configuration, ignoreTimestamp bool) {
	want = want.Clone()
	got = got.Clone()
	if ignoreTimestamp {
		want.Timestamp = got.GetTimestamp()
	}
	if want.GetResources() != nil && got.GetResources() != nil {
		cmpConfigurationResources(t, want.GetResources(), got.GetResources())
		want.Resources = nil
		got.Resources = nil
	}
	CmpJSON(t, want, got)
}

func CmpStoredConfiguration(t *testing.T, want, got *store.Configuration, ignoreTimestamp bool) {
	want = want.Clone()
	if ignoreTimestamp {
		want.Timestamp = got.Timestamp
	}
	CmpJSON(t, want, got)
}

func ConfigurationContains(t *testing.T, storeConf store.Configuration, conf *pb.Configuration) {
	require.Equal(t, storeConf.Id, conf.GetId())
	require.Equal(t, storeConf.Owner, conf.GetOwner())
	require.Equal(t, storeConf.Name, conf.GetName())
	for _, v := range storeConf.Versions {
		if v.Version != conf.GetVersion() {
			continue
		}
		test.CheckProtobufs(t, v.Resources, conf.GetResources(), test.RequireToCheckFunc(require.Equal))
		return
	}
	require.Fail(t, "version not found")
}

func CmpStoredConfigurationMaps(t *testing.T, want, got map[string]store.Configuration) {
	require.Len(t, got, len(want))
	for _, v := range want {
		gotV, ok := got[v.Id]
		require.True(t, ok)
		CmpStoredConfiguration(t, &v, &gotV, true)
	}
}

func CmpCondition(t *testing.T, want, got *pb.Condition, ignoreTimestamp bool) {
	want = want.Clone()
	if ignoreTimestamp {
		want.Timestamp = got.GetTimestamp()
	}
	CmpJSON(t, want, got)
}

func CmpStoredCondition(t *testing.T, want, got *store.Condition, ignoreTimestamp bool) {
	want = want.Clone()
	if ignoreTimestamp {
		want.Timestamp = got.Timestamp
	}
	CmpJSON(t, want, got)
}

func ConditionContains(t *testing.T, storeCond store.Condition, cond *pb.Condition) {
	require.Equal(t, storeCond.Id, cond.GetId())
	require.Equal(t, storeCond.Name, cond.GetName())
	require.Equal(t, storeCond.Owner, cond.GetOwner())
	require.Equal(t, storeCond.Enabled, cond.GetEnabled())
	require.Equal(t, storeCond.ConfigurationId, cond.GetConfigurationId())
	require.Equal(t, storeCond.ApiAccessToken, cond.GetApiAccessToken())
	require.Equal(t, storeCond.Timestamp, cond.GetTimestamp())
	for _, v := range storeCond.Versions {
		if v.Version != cond.GetVersion() {
			continue
		}
		require.Equal(t, v.DeviceIdFilter, cond.GetDeviceIdFilter())
		require.Equal(t, v.ResourceTypeFilter, cond.GetResourceTypeFilter())
		require.Equal(t, v.ResourceHrefFilter, cond.GetResourceHrefFilter())
		require.Equal(t, v.JqExpressionFilter, cond.GetJqExpressionFilter())
		return
	}
	require.Fail(t, "version not found")
}

func CmpStoredConditionsMaps(t *testing.T, want, got map[string]store.Condition) {
	require.Len(t, got, len(want))
	for _, v := range want {
		gotV, ok := got[v.Id]
		require.True(t, ok)
		CmpJSON(t, v, gotV)
	}
}
