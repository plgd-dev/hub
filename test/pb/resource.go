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
	"github.com/plgd-dev/hub/test/pb/baseline"
	"github.com/stretchr/testify/require"
)

type sortResourcesIdsByHref []*commands.ResourceId

func (a sortResourcesIdsByHref) Len() int      { return len(a) }
func (a sortResourcesIdsByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortResourcesIdsByHref) Less(i, j int) bool {
	return a[i].GetHref() < a[j].GetHref()
}

func sortResourceIds(s []*commands.ResourceId) []*commands.ResourceId {
	v := sortResourcesIdsByHref(s)
	sort.Sort(v)
	return v
}

func CmpResourceIds(t *testing.T, expected, got []*commands.ResourceId) {
	require.Len(t, got, len(expected))
	expectedSorted := sortResourceIds(expected)
	gotSorted := sortResourceIds(got)
	test.CheckProtobufs(t, expectedSorted, gotSorted, test.RequireToCheckFunc(require.Equal))
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

func MakeResourceCreated(t *testing.T, deviceID, href, correlationId string, data map[string]interface{}) *events.ResourceCreated {
	return &events.ResourceCreated{
		ResourceId: commands.NewResourceID(deviceID, href),
		Status:     commands.Status_CREATED,
		Content: &commands.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data: func() []byte {
				if data == nil {
					return nil
				}
				return test.EncodeToCbor(t, data)
			}(),
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, correlationId),
	}
}

func CleanUpResourceCreated(e *events.ResourceCreated, resetCorrelationId bool) *events.ResourceCreated {
	if e.GetAuditContext() != nil && resetCorrelationId {
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

	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceCreated(expected, resetCorrelationId)
	CleanUpResourceCreated(got, resetCorrelationId)

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CleanUpResourceChanged(e *events.ResourceChanged, resetCorrelationId bool) *events.ResourceChanged {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceChangedData(t *testing.T, expected, got []byte) {
	cleanUp := func(v interface{}) {
		data := v.(map[interface{}]interface{})
		delete(data, "ins")
		delete(data, "eps")
		delete(data, "pi")
		delete(data, "piid")
	}

	if gotData, ok := test.DecodeCbor(t, got).([]interface{}); ok {
		expectedData := test.DecodeCbor(t, expected).([]interface{})
		for _, v := range gotData {
			cleanUp(v)
		}
		require.Equal(t, expectedData, gotData)
		return
	}

	gotData := test.DecodeCbor(t, got)
	cleanUp(gotData)
	expectedData := test.DecodeCbor(t, expected)
	cleanUp(expectedData)
	require.Equal(t, expectedData, gotData)
}

func CmpResourceChanged(t *testing.T, expected, got *events.ResourceChanged, cmpInterface string) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)

	if expected.GetContent().GetData() != nil && got.GetContent().GetData() != nil {
		cmpFn := CmpResourceChangedData
		if cmpInterface == interfaces.OC_IF_BASELINE {
			cmpFn = baseline.CmpResourceChangedData
		}
		cmpFn(t, expected.GetContent().GetData(), got.GetContent().GetData())

		got.GetContent().Data = nil
		expected.GetContent().Data = nil
	}

	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceChanged(expected, resetCorrelationId)
	CleanUpResourceChanged(got, resetCorrelationId)

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceChanged(t *testing.T, deviceID, href, correlationId string, data interface{}) *events.ResourceChanged {
	return &events.ResourceChanged{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Status: commands.Status_OK,
		Content: &commands.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data: func() []byte {
				if data == nil {
					return nil
				}
				return test.EncodeToCbor(t, data)
			}(),
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, correlationId),
	}
}

func MakeResourceDeleted(t *testing.T, deviceID, href, correlationId string) *events.ResourceDeleted {
	return &events.ResourceDeleted{
		ResourceId: commands.NewResourceID(deviceID, href),
		Status:     commands.Status_OK,
		Content: &commands.Content{
			CoapContentFormat: int32(-1),
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, correlationId),
	}
}

func CleanResourceDeleted(e *events.ResourceDeleted, resetCorrelationId bool) *events.ResourceDeleted {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceDeleted(t *testing.T, expected, got *events.ResourceDeleted) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanResourceDeleted(expected, resetCorrelationId)
	CleanResourceDeleted(got, resetCorrelationId)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceRetrieved(t *testing.T, deviceID, href, correlationId string, data interface{}) *events.ResourceRetrieved {
	return &events.ResourceRetrieved{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Status: commands.Status_OK,
		Content: &commands.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data: func() []byte {
				if data == nil {
					return nil
				}
				return test.EncodeToCbor(t, data)
			}(),
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, correlationId),
	}
}

func CleanUpResourceRetrieved(e *events.ResourceRetrieved, resetCorrelationId bool) *events.ResourceRetrieved {
	if e.GetAuditContext() != nil && resetCorrelationId {
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
		delete(res, "eps")
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
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceRetrieved(expected, resetCorrelationId)
	CleanUpResourceRetrieved(got, resetCorrelationId)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceUpdated(deviceID, href, correlationId string) *events.ResourceUpdated {
	return &events.ResourceUpdated{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Status: commands.Status_OK,
		Content: &commands.Content{
			CoapContentFormat: -1,
		},
		AuditContext: commands.NewAuditContext(service.DeviceUserID, correlationId),
	}
}

func CleanUpResourceUpdated(e *events.ResourceUpdated, resetCorrelationId bool) *events.ResourceUpdated {
	if e.GetAuditContext() != nil && resetCorrelationId {
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
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceUpdated(expected, resetCorrelationId)
	CleanUpResourceUpdated(got, resetCorrelationId)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

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

func CmpResourceValues(t *testing.T, expected, got []*pb.Resource) {
	cleanUpData := func(d map[interface{}]interface{}) {
		delete(d, "eps")
		delete(d, "ins")
		delete(d, "pi")
		delete(d, "piid")
	}

	getData := func(t *testing.T, res *pb.Resource) interface{} {
		d := test.DecodeCbor(t, res.GetData().GetContent().GetData())
		if m, ok := d.(map[interface{}]interface{}); ok {
			cleanUpData(m)
			return d
		}
		if a, ok := d.([]interface{}); ok {
			for _, m := range a {
				cleanUpData(m.(map[interface{}]interface{}))
			}
			return d
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
		resetCorrelationId := e.GetData().GetAuditContext().GetCorrelationId() == ""
		CleanUpResourceChanged(e.GetData(), resetCorrelationId)
		CleanUpResourceChanged(g.GetData(), resetCorrelationId)
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

func CleanUpResourceLinksPublished(e *events.ResourceLinksPublished, resetCorrelationId bool) *events.ResourceLinksPublished {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	test.CleanUpResourcesArray(e.GetResources())
	return e
}

func CleanUpResourceLinksUnpublished(e *events.ResourceLinksUnpublished, resetCorrelationId bool) *events.ResourceLinksUnpublished {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	sort.Strings(e.GetHrefs())
	return e
}
