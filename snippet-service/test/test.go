package test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

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

func CmpConfiguration(t *testing.T, want, got *pb.Configuration) {
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
