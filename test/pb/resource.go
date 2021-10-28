package pb

import (
	"sort"
	"testing"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/oauth-server/service"
	"github.com/stretchr/testify/require"
)

type sortResourcesByHref []*pb.Resource

func (a sortResourcesByHref) Len() int      { return len(a) }
func (a sortResourcesByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortResourcesByHref) Less(i, j int) bool {
	return a[i].GetData().GetResourceId().GetHref() < a[j].GetData().GetResourceId().GetHref()
}

func sortResources(s []*pb.Resource) []*pb.Resource {
	v := sortResourcesByHref(s)
	sort.Sort(v)
	return v
}

func MakeCreateLightResourceResponseData(id string) map[string]interface{} {
	return map[string]interface{}{
		"href": test.TestResourceSwitchesInstanceHref(id),
		"if":   []interface{}{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
		"rt":   []interface{}{types.BINARY_SWITCH},
		"rep": map[string]interface{}{
			"rt":    []interface{}{types.BINARY_SWITCH},
			"if":    []interface{}{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
			"value": false,
		},
		"p": map[string]interface{}{
			"bm": uint64(schema.Discoverable | schema.Observable),
		},
	}
}

func MakeResourceCreated(t *testing.T, deviceID, href string, data map[string]interface{}) *events.ResourceCreated {
	return &events.ResourceCreated{
		ResourceId: commands.NewResourceID(deviceID, href),
		Status:     commands.Status_CREATED,
		Content: &commands.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data:              test.EncodeToCbor(t, data),
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
	}
}

func CleanUpResourceCreated(e *events.ResourceCreated) *events.ResourceCreated {
	if e.GetAuditContext() != nil {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceCreated(t *testing.T, expected, got *events.ResourceCreated) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)

	if expected.GetContent().GetData() != nil && got.GetContent().GetData() != nil {
		cleanData := func(data map[interface{}]interface{}) {
			delete(data, "ins") // instance_id is a random value
		}
		gotData, ok := test.DecodeCbor(t, got.GetContent().GetData()).(map[interface{}]interface{})
		require.True(t, ok)
		expectedData, ok := test.DecodeCbor(t, expected.GetContent().GetData()).(map[interface{}]interface{})
		require.True(t, ok)
		cleanData(expectedData)
		cleanData(gotData)
		require.Equal(t, expectedData, gotData)
		got.GetContent().Data = nil
		expected.GetContent().Data = nil
	}

	CleanUpResourceCreated(expected)
	CleanUpResourceCreated(got)

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CleanUpResourceChanged(e *events.ResourceChanged) *events.ResourceChanged {
	if e.GetAuditContext() != nil {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceChanged(t *testing.T, expected, got *events.ResourceChanged) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)

	if expected.GetContent().GetData() != nil && got.GetContent().GetData() != nil {
		cleanUpLinks := func(data map[interface{}]interface{}) {
			if data == nil {
				return
			}
			links, ok := data["links"].([]interface{})
			if !ok {
				return
			}
			for i := range links {
				link, ok := links[i].(map[interface{}]interface{})
				if !ok {
					continue
				}
				delete(link, "eps")
				delete(link, "ins")
			}
		}

		gotData, ok := test.DecodeCbor(t, got.GetContent().GetData()).(map[interface{}]interface{})
		require.True(t, ok)
		expectedData, ok := test.DecodeCbor(t, expected.GetContent().GetData()).(map[interface{}]interface{})
		require.True(t, ok)
		cleanUpLinks(expectedData)
		cleanUpLinks(gotData)
		require.Equal(t, expectedData, gotData)
		got.GetContent().Data = nil
		expected.GetContent().Data = nil
	}

	CleanUpResourceChanged(expected)
	CleanUpResourceChanged(got)

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceChanged(t *testing.T, deviceID, href string, data interface{}) *events.ResourceChanged {
	return &events.ResourceChanged{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Status: commands.Status_OK,
		Content: &commands.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data:              test.EncodeToCbor(t, data),
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
	}
}

func MakeResourceDeleted(t *testing.T, deviceID, href string) *events.ResourceDeleted {
	return &events.ResourceDeleted{
		ResourceId: commands.NewResourceID(deviceID, href),
		Status:     commands.Status_OK,
		Content: &commands.Content{
			CoapContentFormat: int32(-1),
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
	}
}

func CleanResourceDeleted(e *events.ResourceDeleted) *events.ResourceDeleted {
	if e.GetAuditContext() != nil {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceDeleted(t *testing.T, expected, got *events.ResourceDeleted) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)
	CleanResourceDeleted(expected)
	CleanResourceDeleted(got)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceRetrieved(t *testing.T, deviceID, href string, data interface{}) *events.ResourceRetrieved {
	return &events.ResourceRetrieved{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Status: commands.Status_OK,
		Content: &commands.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data:              test.EncodeToCbor(t, data),
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
	}
}

func CleanUpResourceRetrieved(e *events.ResourceRetrieved) *events.ResourceRetrieved {
	if e.GetAuditContext() != nil {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
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

	data := test.DecodeCbor(t, d.GetContent().GetData())
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
	CleanUpResourceRetrieved(expected)
	CleanUpResourceRetrieved(got)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceUpdated(deviceID, href string) *events.ResourceUpdated {
	return &events.ResourceUpdated{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Status: commands.Status_OK,
		Content: &commands.Content{
			CoapContentFormat: -1,
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
	}
}

func CleanUpResourceUpdated(e *events.ResourceUpdated) *events.ResourceUpdated {
	if e.GetAuditContext() != nil {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceUpdated(t *testing.T, expected, got *events.ResourceUpdated) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)
	if expected.GetContent().GetData() != nil && got.GetContent().GetData() != nil {
		gotData := test.DecodeCbor(t, got.GetContent().GetData())
		got.GetContent().Data = nil
		expectedData := test.DecodeCbor(t, expected.GetContent().GetData())
		expected.GetContent().Data = nil
		require.Equal(t, expectedData, gotData)
	}
	CleanUpResourceUpdated(expected)
	CleanUpResourceUpdated(got)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CmpResourceValues(t *testing.T, expected, got []*pb.Resource) {
	getData := func(t *testing.T, res *pb.Resource) interface{} {
		d := test.DecodeCbor(t, res.GetData().GetContent().GetData())
		if m, ok := d.(map[interface{}]interface{}); ok {
			delete(m, "pi")
			delete(m, "piid")
		}
		return d
	}

	require.Len(t, got, len(expected))
	expectedSorted := sortResources(expected)
	gotSorted := sortResources(got)

	for k := range expectedSorted {
		e := expectedSorted[k]
		g := gotSorted[k]
		dataExpected := getData(t, e)
		e.Data.Content.Data = nil
		dataGot := getData(t, g)
		g.Data.Content.Data = nil
		require.Equal(t, dataExpected, dataGot)
		CleanUpResourceChanged(e.GetData())
		CleanUpResourceChanged(g.GetData())
		test.CheckProtobufs(t, e, g, test.RequireToCheckFunc(require.Equal))
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

func CmpHubConfigurationResponse(t *testing.T, expected, got *pb.HubConfigurationResponse) {
	require.NotEmpty(t, got.CertificateAuthorities)
	got.CertificateAuthorities = ""
	require.NotEqual(t, int64(0), got.CurrentTime)
	got.CurrentTime = 0
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CleanUpResourceLinksSnapshotTaken(e *events.ResourceLinksSnapshotTaken) *events.ResourceLinksSnapshotTaken {
	e.EventMetadata = nil
	for _, r := range e.GetResources() {
		r.ValidUntil = 0
	}
	return e
}

func CleanUpResourceLinksPublished(e *events.ResourceLinksPublished) *events.ResourceLinksPublished {
	if e.GetAuditContext() != nil {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	test.CleanUpResourcesArray(e.GetResources())
	return e
}

func CleanUpResourceLinksUnpublished(e *events.ResourceLinksUnpublished) *events.ResourceLinksUnpublished {
	if e.GetAuditContext() != nil {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	sort.Strings(e.GetHrefs())
	return e
}
