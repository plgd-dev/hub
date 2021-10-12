package test

import (
	"sort"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/stretchr/testify/require"
)

func ResourceLinksToResources(deviceID string, s []schema.ResourceLink) []*commands.Resource {
	r := make([]*commands.Resource, 0, len(s))
	for _, l := range s {
		l.DeviceID = deviceID
		r = append(r, commands.SchemaResourceLinkToResource(l, time.Time{}))
	}
	CleanUpResourcesArray(r)
	return r
}

func ResourceLinksToResources2(deviceID string, s []schema.ResourceLink) []*pb.Resource {
	r := make([]*pb.Resource, 0, len(s))
	for _, l := range s {
		res := pb.Resource{
			Types: l.ResourceTypes,
			Data: &events.ResourceChanged{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     l.Href,
				},
			},
		}
		r = append(r, &res)
	}
	return r
}

func CleanUpResourcesArray(resources []*commands.Resource) []*commands.Resource {
	for _, r := range resources {
		r.ValidUntil = 0
	}
	SortResources(resources)
	return resources
}

func CleanUpResourceLinksSnapshotTaken(e *events.ResourceLinksSnapshotTaken) *events.ResourceLinksSnapshotTaken {
	e.EventMetadata = nil
	for _, r := range e.GetResources() {
		r.ValidUntil = 0
	}
	return e
}

func CleanUpResourceLinksPublished(e *events.ResourceLinksPublished) *events.ResourceLinksPublished {
	e.EventMetadata = nil
	e.AuditContext = nil
	CleanUpResourcesArray(e.GetResources())
	return e
}

type SortResourcesByHref []*commands.Resource

func (a SortResourcesByHref) Len() int      { return len(a) }
func (a SortResourcesByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortResourcesByHref) Less(i, j int) bool {
	return a[i].GetHref() < a[j].GetHref()
}

func SortResources(s []*commands.Resource) []*commands.Resource {
	v := SortResourcesByHref(s)
	sort.Sort(v)
	return v
}

func CmpResourceCreated(t *testing.T, expected, got *events.ResourceCreated) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)

	gotData, ok := DecodeCbor(t, got.GetContent().GetData()).(map[interface{}]interface{})
	require.True(t, ok)
	delete(gotData, "ins") // instance_id is a random value
	expectedData, ok := DecodeCbor(t, expected.GetContent().GetData()).(map[interface{}]interface{})
	require.True(t, ok)
	delete(gotData, "ins")
	require.Equal(t, gotData, expectedData)
	got.GetContent().Data = nil
	expected.GetContent().Data = nil

	expected.AuditContext = nil
	got.AuditContext = nil
	expected.EventMetadata = nil
	got.EventMetadata = nil

	CheckProtobufs(t, expected, got, RequireToCheckFunc(require.Equal))
}

func CmpResourceDeleted(t *testing.T, expected, got *events.ResourceDeleted) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)
	expected.AuditContext = nil
	got.AuditContext = nil
	expected.EventMetadata = nil
	got.EventMetadata = nil
	CheckProtobufs(t, expected, got, RequireToCheckFunc(require.Equal))
}

type resourceData map[interface{}]interface{}

func getResourceRetrievedData(t *testing.T, d *events.ResourceRetrieved) map[string]resourceData {
	resData := make(map[string]resourceData)
	addData := func(v interface{}) {
		res, ok := v.(map[interface{}]interface{})
		require.True(t, ok)
		href, _ := res["href"].(string)
		delete(res, "ins")
		delete(res, "piid")
		resData[href] = res
	}

	data := DecodeCbor(t, d.GetContent().GetData())
	resources, ok := data.([]interface{})
	if ok {
		for _, resource := range resources {
			addData(resource)
		}
		return resData
	}

	addData(data)
	return resData
}

func CmpResourceRetrieved(t *testing.T, expected, got *events.ResourceRetrieved) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)
	if expected.GetContent().GetData() != nil && got.GetContent().GetData() != nil {
		gotData := getResourceRetrievedData(t, got)
		require.NotEmpty(t, gotData)
		expectedData := getResourceRetrievedData(t, expected)
		require.NotEmpty(t, expectedData)
		require.Equal(t, gotData, expectedData)
		got.GetContent().Data = nil
		expected.GetContent().Data = nil
	}

	expected.AuditContext = nil
	got.AuditContext = nil
	expected.EventMetadata = nil
	got.EventMetadata = nil
	CheckProtobufs(t, expected, got, RequireToCheckFunc(require.Equal))
}

func CmpResourceUpdated(t *testing.T, expected, got *events.ResourceUpdated) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)
	if expected.GetContent().GetData() != nil && got.GetContent().GetData() != nil {
		gotData := DecodeCbor(t, got.GetContent().GetData())
		got.GetContent().Data = nil
		expectedData := DecodeCbor(t, expected.GetContent().GetData())
		expected.GetContent().Data = nil
		require.Equal(t, expectedData, gotData)
	}
	expected.AuditContext = nil
	got.AuditContext = nil
	expected.EventMetadata = nil
	got.EventMetadata = nil
	CheckProtobufs(t, expected, got, RequireToCheckFunc(require.Equal))
}

func CmpResourceValues(t *testing.T, expected, got []*pb.Resource) {
	require.Len(t, got, len(expected))
	for idx := range expected {
		dataExpected := DecodeCbor(t, expected[idx].GetData().GetContent().GetData())
		expected[idx].Data.Content.Data = nil
		dataGot := DecodeCbor(t, got[idx].GetData().GetContent().GetData())
		got[idx].Data.Content.Data = nil
		require.Equal(t, dataExpected, dataGot)
		expected[idx].GetData().AuditContext = nil
		got[idx].GetData().AuditContext = nil
		expected[idx].GetData().EventMetadata = nil
		got[idx].GetData().EventMetadata = nil
		CheckProtobufs(t, expected[idx], got[idx], RequireToCheckFunc(require.Equal))
	}
}

// compare only deviceId, href and resourceTypes
func CmpResourceValuesBasic(t *testing.T, expected, got []*pb.Resource) {
	require.Len(t, got, len(expected))
	expectedData := make(map[string][]string)
	gotData := make(map[string][]string)
	for idx := range expected {
		expectedData[expected[idx].GetData().GetResourceId().ToString()] = expected[idx].GetTypes()
		gotData[got[idx].GetData().GetResourceId().ToString()] = got[idx].GetTypes()
	}
	require.Equal(t, expectedData, gotData)
}
