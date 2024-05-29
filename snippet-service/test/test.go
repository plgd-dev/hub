package test

import (
	"cmp"
	"errors"
	"slices"
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
	RuntimeConfig.conditionIds = make(map[int]string)
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

func MergeConfigurations(c1, c2 *store.Configuration) error {
	if c1.Id != c2.Id || c1.Owner != c2.Owner {
		return errors.New("conditions to merge must have the same ID and owner")
	}

	if c2.Latest != nil {
		latest := c2.Latest.Copy()
		c1.Latest = &latest
	}
	c1.Versions = append(c1.Versions, c2.Versions...)
	slices.SortFunc(c1.Versions, func(i, j store.ConfigurationVersion) int {
		return cmp.Compare(i.Version, j.Version)
	})
	c1.Versions = slices.CompactFunc(c1.Versions, func(i, j store.ConfigurationVersion) bool {
		return i.Version == j.Version
	})

	if c1.Latest != nil {
		if !slices.ContainsFunc(c1.Versions, func(cv store.ConfigurationVersion) bool {
			return cv.Version == c1.Latest.Version
		}) {
			c1.Versions = append(c1.Versions, c1.Latest.Copy())
		}
	} else if len(c1.Versions) > 0 {
		latest := c1.Versions[len(c1.Versions)-1].Copy()
		c1.Latest = &latest
	}
	return nil
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

func ConfigurationContains(t *testing.T, storeConf store.Configuration, conf *pb.Configuration) {
	require.Equal(t, storeConf.Id, conf.GetId())
	require.Equal(t, storeConf.Owner, conf.GetOwner())
	for _, v := range storeConf.Versions {
		if v.Version != conf.GetVersion() {
			continue
		}
		test.CheckProtobufs(t, v.Resources, conf.GetResources(), test.RequireToCheckFunc(require.Equal))
		return
	}
	require.Fail(t, "version not found")
}

func CmpStoredConfiguration(t *testing.T, want, got *store.Configuration, ignoreTimestamp, ignoreLatest bool) {
	require.Len(t, got.Versions, len(want.Versions))
	if ignoreTimestamp || ignoreLatest {
		want = want.Clone()
		got = got.Clone()
	}
	if ignoreTimestamp {
		if want.Latest != nil && got.Latest != nil {
			want.Latest.Timestamp = got.Latest.Timestamp
		}
		for i := range want.Versions {
			want.Versions[i].Timestamp = got.Versions[i].Timestamp
		}
	}
	if ignoreLatest {
		want.Latest = got.Latest
	}
	CmpJSON(t, want, got)
}

func CmpStoredConfigurationMaps(t *testing.T, want, got map[string]store.Configuration) {
	require.Len(t, got, len(want))
	for _, v := range want {
		gotV, ok := got[v.Id]
		require.True(t, ok)
		CmpStoredConfiguration(t, &v, &gotV, true, false)
	}
}

func MergeConditions(c1, c2 *store.Condition) error {
	if c1.Id != c2.Id || c1.Owner != c2.Owner || c1.ConfigurationId != c2.ConfigurationId {
		return errors.New("conditions to merge must have the same ID, owner and configuration ID")
	}

	if c2.Latest != nil {
		latest := c2.Latest.Copy()
		c1.Latest = &latest
	}
	c1.Versions = append(c1.Versions, c2.Versions...)
	slices.SortFunc(c1.Versions, func(i, j store.ConditionVersion) int {
		return cmp.Compare(i.Version, j.Version)
	})
	c1.Versions = slices.CompactFunc(c1.Versions, func(i, j store.ConditionVersion) bool {
		return i.Version == j.Version
	})

	if c1.Latest != nil {
		if !slices.ContainsFunc(c1.Versions, func(cv store.ConditionVersion) bool {
			return cv.Version == c1.Latest.Version
		}) {
			c1.Versions = append(c1.Versions, c1.Latest.Copy())
		}
	} else if len(c1.Versions) > 0 {
		latest := c1.Versions[len(c1.Versions)-1].Copy()
		c1.Latest = &latest
	}
	return nil
}

func CmpCondition(t *testing.T, want, got *pb.Condition, ignoreTimestamp bool) {
	want = want.Clone()
	if ignoreTimestamp {
		want.Timestamp = got.GetTimestamp()
	}
	CmpJSON(t, want, got)
}

func ConditionContains(t *testing.T, storeCond store.Condition, cond *pb.Condition) {
	require.Equal(t, storeCond.Id, cond.GetId())
	require.Equal(t, storeCond.Owner, cond.GetOwner())
	require.Equal(t, storeCond.ConfigurationId, cond.GetConfigurationId())
	for _, v := range storeCond.Versions {
		if v.Version != cond.GetVersion() {
			continue
		}
		require.Equal(t, v.Name, cond.GetName())
		require.Equal(t, v.Version, cond.GetVersion())
		require.Equal(t, v.Enabled, cond.GetEnabled())
		require.Equal(t, v.Timestamp, cond.GetTimestamp())
		require.Equal(t, v.DeviceIdFilter, cond.GetDeviceIdFilter())
		require.Equal(t, v.ResourceTypeFilter, cond.GetResourceTypeFilter())
		require.Equal(t, v.ResourceHrefFilter, cond.GetResourceHrefFilter())
		require.Equal(t, v.JqExpressionFilter, cond.GetJqExpressionFilter())
		require.Equal(t, v.ApiAccessToken, cond.GetApiAccessToken())
		return
	}
	require.Fail(t, "version not found")
}

func CmpStoredCondition(t *testing.T, want, got *store.Condition, ignoreTimestamp, ignoreLatest bool) {
	require.Len(t, got.Versions, len(want.Versions))
	if ignoreTimestamp || ignoreLatest {
		want = want.Clone()
		got = got.Clone()
	}
	if ignoreTimestamp {
		if want.Latest != nil && got.Latest != nil {
			want.Latest.Timestamp = got.Latest.Timestamp
		}
		for i := range want.Versions {
			want.Versions[i].Timestamp = got.Versions[i].Timestamp
		}
	}
	if ignoreLatest {
		want.Latest = got.Latest
	}
	CmpJSON(t, want, got)
}

func CmpStoredConditionMaps(t *testing.T, want, got map[string]store.Condition) {
	require.Len(t, got, len(want))
	for _, v := range want {
		gotV, ok := got[v.Id]
		require.True(t, ok)
		CmpStoredCondition(t, &v, &gotV, false, false)
	}
}
